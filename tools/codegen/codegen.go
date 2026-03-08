// Codegen tool: reads Go model structs from models_v2.go and assignment.go,
// generates TypeScript interfaces in tools/shared/types.generated.ts and a
// typed API client in tools/shared/client.generated.ts.
//
// Usage: go run tools/codegen/codegen.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type goField struct {
	Name     string
	GoType   string
	JSONTag  string
	Optional bool
}

type goStruct struct {
	Name   string
	Fields []goField
}

type goConst struct {
	TypeName string
	Values   []string
}

func main() {
	root := findProjectRoot()
	modelFiles := []string{
		filepath.Join(root, "models_v2.go"),
		filepath.Join(root, "assignment.go"),
	}

	var structs []goStruct
	var consts []goConst
	for _, f := range modelFiles {
		s, c := parseGoFile(f)
		structs = append(structs, s...)
		consts = append(consts, c...)
	}

	outDir := filepath.Join(root, "tools", "shared")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output dir: %v\n", err)
		os.Exit(1)
	}

	typesPath := filepath.Join(outDir, "types.generated.ts")
	if err := generateTypes(typesPath, structs, consts); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate types: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s (%d structs, %d const groups)\n", typesPath, len(structs), len(consts))

	clientPath := filepath.Join(outDir, "client.generated.ts")
	if err := generateClient(clientPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate client: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s\n", clientPath)
}

func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	cwd, _ := os.Getwd()
	return cwd
}

var (
	reStructStart = regexp.MustCompile("^type\\s+(\\w+)\\s+struct\\s*\\{")
	reField       = regexp.MustCompile("^\\s+(\\w+)\\s+([\\w\\[\\]*\\.]+)\\s+" + string(rune(96)) + "json:\"([^\"]+)\"" + string(rune(96)))
	reConstBlock  = regexp.MustCompile("^const\\s*\\(")
	reConstVal    = regexp.MustCompile("^\\s+(\\w+)\\s+\\w+\\s*=\\s*\"([^\"]+)\"")
	reTypeDef     = regexp.MustCompile("^type\\s+(\\w+)\\s+(string|int)")
)

func parseGoFile(path string) ([]goStruct, []goConst) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot open %s: %v\n", path, err)
		return nil, nil
	}
	defer f.Close()

	var (
		structs     []goStruct
		consts      []goConst
		current     *goStruct
		inConst     bool
		constType   string
		constValues []string
		typeAliases = map[string]string{}
	)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if m := reTypeDef.FindStringSubmatch(line); m != nil {
			typeAliases[m[1]] = m[2]
		}

		if m := reStructStart.FindStringSubmatch(line); m != nil {
			current = &goStruct{Name: m[1]}
			continue
		}
		if current != nil {
			if strings.TrimSpace(line) == "}" {
				structs = append(structs, *current)
				current = nil
				continue
			}
			if m := reField.FindStringSubmatch(line); m != nil {
				tag := m[3]
				jsonName := strings.Split(tag, ",")[0]
				if jsonName == "-" {
					continue
				}
				optional := strings.Contains(tag, "omitempty") || strings.HasPrefix(m[2], "*")
				current.Fields = append(current.Fields, goField{
					Name:     m[1],
					GoType:   m[2],
					JSONTag:  jsonName,
					Optional: optional,
				})
			}
			continue
		}

		if reConstBlock.MatchString(line) {
			inConst = true
			constType = ""
			constValues = nil
			continue
		}
		if inConst {
			if strings.TrimSpace(line) == ")" {
				if constType != "" && len(constValues) > 0 {
					consts = append(consts, goConst{TypeName: constType, Values: constValues})
				}
				inConst = false
				continue
			}
			if m := reConstVal.FindStringSubmatch(line); m != nil {
				for tn := range typeAliases {
					if strings.HasPrefix(m[1], stripSuffix(tn)) {
						constType = tn
						break
					}
				}
				constValues = append(constValues, m[2])
			}
		}
	}

	return structs, consts
}

func stripSuffix(name string) string {
	return name
}

func goTypeToTS(goType string) string {
	goType = strings.TrimPrefix(goType, "*")

	switch goType {
	case "string", "sql.NullString":
		return "string"
	case "int", "int64", "int32", "float64", "float32", "sql.NullInt64":
		return "number"
	case "bool":
		return "boolean"
	case "time.Time":
		return "string"
	case "json.RawMessage":
		return "Record<string, unknown>"
	case "[]string":
		return "string[]"
	case "[]byte":
		return "string"
	case "[]int":
		return "number[]"
	}

	if strings.HasPrefix(goType, "map[string]") {
		return "Record<string, unknown>"
	}

	if strings.HasPrefix(goType, "[]") {
		inner := goTypeToTS(goType[2:])
		return inner + "[]"
	}

	return goType
}

func generateTypes(path string, structs []goStruct, consts []goConst) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	fmt.Fprintf(w, "/**\n")
	fmt.Fprintf(w, " * AUTO-GENERATED by tools/codegen/codegen.go -- DO NOT EDIT.\n")
	fmt.Fprintf(w, " * Generated at: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(w, " * Source files: models_v2.go, assignment.go\n")
	fmt.Fprintf(w, " */\n\n")

	for _, c := range consts {
		if len(c.Values) == 0 {
			continue
		}
		fmt.Fprintf(w, "export type %s =\n", c.TypeName)
		for i, v := range c.Values {
			if i < len(c.Values)-1 {
				fmt.Fprintf(w, "  | \"%s\"\n", v)
			} else {
				fmt.Fprintf(w, "  | \"%s\";\n", v)
			}
		}
		fmt.Fprintln(w)
	}

	for _, s := range structs {
		if len(s.Fields) == 0 {
			continue
		}
		fmt.Fprintf(w, "export interface %s {\n", s.Name)
		for _, field := range s.Fields {
			tsType := goTypeToTS(field.GoType)
			opt := ""
			if field.Optional {
				opt = "?"
			}
			fmt.Fprintf(w, "  %s%s: %s;\n", field.JSONTag, opt, tsType)
		}
		fmt.Fprintf(w, "}\n\n")
	}

	return nil
}

func generateClient(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	ts := time.Now().UTC().Format(time.RFC3339)
	bt := string(rune(96)) // backtick character

	lines := []string{
		"/**",
		" * AUTO-GENERATED typed API client -- DO NOT EDIT.",
		" * Generated at: " + ts,
		" * Source: tools/codegen/codegen.go",
		" */",
		"",
		"import type {",
		"  Project, TaskV2, Run, Event, Agent, Memory, Policy,",
		"  AssignmentEnvelope, ContextPack, RepoFile, DashboardData,",
		"  TaskTemplate,",
		"} from './types.generated';",
		"",
		"export interface ClientOptions {",
		"  baseURL: string;",
		"  apiKey?: string;",
		"}",
		"",
		"export class CocopilotClient {",
		"  private baseURL: string;",
		"  private headers: Record<string, string>;",
		"",
		"  constructor(opts: ClientOptions) {",
		"    this.baseURL = opts.baseURL.replace(/\\/+$/, '');",
		"    this.headers = { 'Content-Type': 'application/json' };",
		"    if (opts.apiKey) {",
		"      this.headers['X-API-Key'] = opts.apiKey;",
		"    }",
		"  }",
		"",
		"  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {",
		"    const url = this.baseURL + path;",
		"    const init: RequestInit = { method, headers: this.headers };",
		"    if (body !== undefined) {",
		"      init.body = JSON.stringify(body);",
		"    }",
		"    const resp = await fetch(url, init);",
		"    if (!resp.ok) {",
		"      const text = await resp.text();",
		"      throw new Error(" + bt + "HTTP ${resp.status}: ${text}" + bt + ");",
		"    }",
		"    if (resp.status === 204) return undefined as T;",
		"    return resp.json() as Promise<T>;",
		"  }",
		"",
		"  // Health",
		"  health() { return this.request<{ ok: boolean }>('GET', '/api/v2/health'); }",
		"",
		"  // Projects",
		"  listProjects() { return this.request<{ projects: Project[] }>('GET', '/api/v2/projects'); }",
		"  getProject(id: string) { return this.request<Project>('GET', " + bt + "/api/v2/projects/${id}" + bt + "); }",
		"  createProject(name: string, workdir: string) {",
		"    return this.request<Project>('POST', '/api/v2/projects', { name, workdir });",
		"  }",
		"",
		"  // Tasks",
		"  listTasks(projectId: string, params?: Record<string, string>) {",
		"    const qs = params ? '?' + new URLSearchParams(params).toString() : '';",
		"    return this.request<{ tasks: TaskV2[]; total: number }>('GET', " + bt + "/api/v2/projects/${projectId}/tasks${qs}" + bt + ");",
		"  }",
		"  getTask(id: number) { return this.request<TaskV2>('GET', " + bt + "/api/v2/tasks/${id}" + bt + "); }",
		"  createTask(projectId: string, body: Partial<TaskV2>) {",
		"    return this.request<TaskV2>('POST', " + bt + "/api/v2/projects/${projectId}/tasks" + bt + ", body);",
		"  }",
		"  updateTask(id: number, body: Partial<TaskV2>) {",
		"    return this.request<TaskV2>('PATCH', " + bt + "/api/v2/tasks/${id}" + bt + ", body);",
		"  }",
		"  deleteTask(id: number) { return this.request<void>('DELETE', " + bt + "/api/v2/tasks/${id}" + bt + "); }",
		"",
		"  // Claims",
		"  claimTask(id: number, agentId: string) {",
		"    return this.request<AssignmentEnvelope>('POST', " + bt + "/api/v2/tasks/${id}/claim" + bt + ", { agent_id: agentId });",
		"  }",
		"  claimNext(projectId: string, agentId: string) {",
		"    return this.request<AssignmentEnvelope>('POST', " + bt + "/api/v2/projects/${projectId}/tasks/claim-next" + bt + ", { agent_id: agentId });",
		"  }",
		"  completeTask(id: number, body: Record<string, unknown>) {",
		"    return this.request<void>('POST', " + bt + "/api/v2/tasks/${id}/complete" + bt + ", body);",
		"  }",
		"  failTask(id: number, body: Record<string, unknown>) {",
		"    return this.request<void>('POST', " + bt + "/api/v2/tasks/${id}/fail" + bt + ", body);",
		"  }",
		"",
		"  // Runs",
		"  getRun(id: string) { return this.request<Run>('GET', " + bt + "/api/v2/runs/${id}" + bt + "); }",
		"",
		"  // Events",
		"  listEvents(params?: Record<string, string>) {",
		"    const qs = params ? '?' + new URLSearchParams(params).toString() : '';",
		"    return this.request<{ events: Event[] }>('GET', " + bt + "/api/v2/events${qs}" + bt + ");",
		"  }",
		"",
		"  // Agents",
		"  listAgents() { return this.request<{ agents: Agent[] }>('GET', '/api/v2/agents'); }",
		"",
		"  // Memory",
		"  listMemories(projectId: string) {",
		"    return this.request<{ memories: Memory[] }>('GET', " + bt + "/api/v2/projects/${projectId}/memory" + bt + ");",
		"  }",
		"  putMemory(projectId: string, body: Partial<Memory>) {",
		"    return this.request<Memory>('PUT', " + bt + "/api/v2/projects/${projectId}/memory" + bt + ", body);",
		"  }",
		"",
		"  // Policies",
		"  listPolicies(projectId: string) {",
		"    return this.request<{ policies: Policy[] }>('GET', " + bt + "/api/v2/projects/${projectId}/policies" + bt + ");",
		"  }",
		"",
		"  // Dashboard",
		"  getDashboard(projectId: string) {",
		"    return this.request<DashboardData>('GET', " + bt + "/api/v2/projects/${projectId}/dashboard" + bt + ");",
		"  }",
		"",
		"  // Repo Files",
		"  listRepoFiles(projectId: string) {",
		"    return this.request<{ files: RepoFile[]; total: number }>('GET', " + bt + "/api/v2/projects/${projectId}/files" + bt + ");",
		"  }",
		"  scanRepoFiles(projectId: string) {",
		"    return this.request<{ files: RepoFile[] }>('POST', " + bt + "/api/v2/projects/${projectId}/files/scan" + bt + ");",
		"  }",
		"",
		"  // Context Packs",
		"  listContextPacks(projectId: string) {",
		"    return this.request<{ context_packs: ContextPack[] }>('GET', " + bt + "/api/v2/projects/${projectId}/context-packs" + bt + ");",
		"  }",
		"",
		"  // Templates",
		"  listTemplates(projectId: string) {",
		"    return this.request<{ templates: TaskTemplate[] }>('GET', " + bt + "/api/v2/projects/${projectId}/templates" + bt + ");",
		"  }",
		"",
		"  // Notifications",
		"  listNotifications(projectId: string, params?: Record<string, string>) {",
		"    const qs = params ? '?' + new URLSearchParams(params).toString() : '';",
		"    return this.request<{ events: Event[] }>('GET', " + bt + "/api/v2/projects/${projectId}/notifications${qs}" + bt + ");",
		"  }",
		"}",
	}

	for _, line := range lines {
		fmt.Fprintln(w, line)
	}

	return nil
}
