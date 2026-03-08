package server

import (
	"fmt"
	"html"
	"net/http"
	"strings"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlTemplate)
}

// subPageHead returns a consistent HTML head + nav bar for all sub-pages,
// matching the Kanban's VSCode-inspired dark theme.
func subPageHead(title string) string {
	return `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8">` +
		`<meta name="viewport" content="width=device-width, initial-scale=1.0">` +
		`<title>` + html.EscapeString(title) + ` - Cocopilot</title>` +
		`<style>` +
		`*{box-sizing:border-box;margin:0;padding:0;}` +
		`body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;font-size:13px;background:#1e1e1e;color:#ccc;min-height:100vh;}` +
		`.top-nav{background:#252526;border-bottom:1px solid #3c3c3c;padding:10px 20px;display:flex;align-items:center;gap:12px;}` +
		`.top-nav .nav-icon{color:#0078d4;}` +
		`.top-nav .nav-title{font-size:14px;font-weight:600;color:#ccc;text-decoration:none;}` +
		`.top-nav .nav-title:hover{color:#fff;}` +
		`.top-nav nav{display:flex;align-items:center;gap:8px;font-size:12px;}` +
		`.top-nav nav a{color:#858585;text-decoration:none;padding:4px 6px;border-radius:4px;transition:all .15s;}` +
		`.top-nav nav a:hover,.top-nav nav a.active{color:#ccc;background:#3c3c3c;}` +
		`.page-body{max-width:1100px;margin:0 auto;padding:24px 20px;}` +
		`.card{background:#252526;border:1px solid #3c3c3c;border-radius:8px;padding:20px;}` +
		`h1{font-size:20px;margin:0 0 6px;color:#e0e0e0;}` +
		`h2{font-size:15px;margin:18px 0 8px;color:#e0e0e0;border-bottom:1px solid #3c3c3c;padding-bottom:6px;}` +
		`p{margin:0 0 14px;color:#858585;font-size:13px;}` +
		`.meta{display:flex;flex-wrap:wrap;align-items:center;gap:12px;margin-bottom:14px;font-size:12px;color:#b0b0b0;}` +
		`.muted{color:#858585;}` +
		`.btn{background:#3c3c3c;border:1px solid #505050;color:#ccc;padding:6px 10px;border-radius:4px;cursor:pointer;font-size:12px;transition:all .15s;}` +
		`.btn:hover{background:#0078d4;border-color:#0078d4;color:#fff;}` +
		`.btn.active{background:#505050;border-color:#606060;}` +
		`table{width:100%;border-collapse:collapse;font-size:12px;}` +
		`th,td{padding:8px;border-bottom:1px solid #3c3c3c;text-align:left;}` +
		`th{color:#0078d4;font-weight:600;font-size:11px;text-transform:uppercase;letter-spacing:.05em;}` +
		`tbody tr:hover{background:#2a2d2e;}` +
		`.field{display:flex;align-items:center;gap:6px;font-size:12px;color:#b0b0b0;}` +
		`.input,.textarea,.select{background:#3c3c3c;border:1px solid #505050;color:#ccc;border-radius:4px;padding:6px 8px;font-size:12px;}` +
		`.input:focus,.textarea:focus,.select:focus{border-color:#0078d4;outline:none;}` +
		`.textarea{min-height:100px;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;}` +
		`.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:10px;margin-bottom:14px;}` +
		`.stat{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:12px;text-align:center;}` +
		`.stat .label{font-size:11px;color:#858585;text-transform:uppercase;letter-spacing:.05em;margin-bottom:4px;}` +
		`.stat .value{font-size:22px;font-weight:700;color:#ccc;}` +
		`.stat.ok .value{color:#89d185;} .stat.warn .value{color:#cca700;} .stat.error .value{color:#f14c4c;}` +
		`.pill{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:999px;padding:4px 10px;font-size:12px;}` +
		`.footer{background:#252526;border-top:1px solid #3c3c3c;padding:10px 20px;text-align:center;font-size:12px;color:#858585;margin-top:32px;}` +
		`.footer a{color:#0078d4;text-decoration:none;}.footer a:hover{text-decoration:underline;}` +
		`</style></head><body>` +
		`<div class="top-nav">` +
		`<svg class="nav-icon" width="18" height="18" viewBox="0 0 16 16" fill="currentColor"><path d="M8.5 1a6.5 6.5 0 1 1 0 13 6.5 6.5 0 0 1 0-13zm0 1a5.5 5.5 0 1 0 0 11 5.5 5.5 0 0 0 0-11zm-2 3.5a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm0 2a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm0 2a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5z"/></svg>` +
		`<a class="nav-title" href="/">Cocopilot</a>` +
		`<nav>` +
		`<a href="/">Kanban</a>` +
		`<a href="/agents">Agents</a>` +
		`<a href="/runs">Runs</a>` +
		`<a href="/memory">Memory</a>` +
		`<a href="/context-packs">Context Packs</a>` +
		`<a href="/graphs/tasks">Task DAG</a>` +
		`<a href="/graphs/repo">Repo Graph</a>` +
		`<a href="/repo">Repo</a>` +
		`<a href="/audit">Audit</a>` +
		`<a href="/health">Health</a>` +
		`</nav></div>` +
		`<div class="page-body">`
}

// subPageFoot returns a consistent footer + closing tags for sub-pages.
func subPageFoot() string {
	return `</div><div class="footer">Created by <a href="https://dganev.com" target="_blank" rel="noopener">syl</a></div></body></html>`
}

func uiPlaceholderPage(title string, subtitle string, details []string) string {
	var b strings.Builder
	b.WriteString(subPageHead(title))
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>")
	if subtitle != "" {
		b.WriteString("<p>")
		b.WriteString(html.EscapeString(subtitle))
		b.WriteString("</p>")
	}
	if len(details) > 0 {
		b.WriteString("<ul style=\"margin:12px 0 0;padding-left:20px;\">")
		for _, item := range details {
			b.WriteString("<li style=\"margin:4px 0;\">")
			b.WriteString(html.EscapeString(item))
			b.WriteString("</li>")
		}
		b.WriteString("</ul>")
	}
	b.WriteString("<p class=\"pill\" style=\"display:inline-block;margin-top:10px;\">Placeholder UI route</p>")
	b.WriteString("</div>")
	b.WriteString(subPageFoot())
	return b.String()
}

func runViewerPage(runID string) string {
	var b strings.Builder
	b.WriteString(subPageHead("Run Viewer"))
	b.WriteString("<style>")
	b.WriteString(".pill.s-RUNNING{border-color:#0078d4;color:#0078d4;} .pill.s-SUCCEEDED{border-color:#89d185;color:#89d185;}")
	b.WriteString(".pill.s-FAILED{border-color:#f14c4c;color:#f14c4c;} .pill.s-PENDING{border-color:#858585;color:#858585;}")
	b.WriteString(".stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(120px,1fr));gap:10px;margin-bottom:14px;}")
	// Step timeline
	b.WriteString(".step-timeline{display:flex;flex-direction:column;gap:0;margin-bottom:14px;}")
	b.WriteString(".step-item{display:flex;gap:12px;padding:8px 0;border-left:2px solid #3c3c3c;margin-left:8px;padding-left:16px;position:relative;}")
	b.WriteString(".step-item::before{content:'';position:absolute;left:-5px;top:12px;width:8px;height:8px;border-radius:50%;background:#505050;}")
	b.WriteString(".step-item.s-completed::before{background:#89d185;} .step-item.s-running::before{background:#0078d4;} .step-item.s-failed::before{background:#f14c4c;}")
	b.WriteString(".step-info{flex:1;} .step-name{font-size:13px;font-weight:600;color:#ccc;} .step-meta{font-size:11px;color:#858585;margin-top:2px;}")
	b.WriteString(".step-dur{font-size:11px;color:#858585;white-space:nowrap;}")
	// Log console
	b.WriteString(".log-console{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:0;margin-bottom:14px;max-height:350px;overflow:auto;font-family:monospace;font-size:12px;}")
	b.WriteString(".log-console .log-line{padding:2px 12px;border-bottom:1px solid #252526;white-space:pre-wrap;word-break:break-all;}")
	b.WriteString(".log-console .log-line:nth-child(odd){background:#1a1a1a;}")
	b.WriteString(".log-console .log-ts{color:#858585;margin-right:8px;} .log-console .log-lvl{font-weight:600;margin-right:6px;}")
	b.WriteString(".log-console .log-lvl.l-error{color:#f14c4c;} .log-console .log-lvl.l-warn{color:#cca700;} .log-console .log-lvl.l-info{color:#3794ff;}")
	b.WriteString(".log-console .log-msg{color:#ccc;}")
	// Artifacts
	b.WriteString(".artifact-list{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:10px;margin-bottom:14px;}")
	b.WriteString(".artifact-card{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:10px 12px;}")
	b.WriteString(".artifact-card .art-name{font-size:13px;font-weight:600;color:#0078d4;margin-bottom:4px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;}")
	b.WriteString(".artifact-card .art-meta{font-size:11px;color:#858585;}")
	b.WriteString(".artifact-card .art-link{display:inline-block;margin-top:6px;font-size:11px;color:#0078d4;text-decoration:none;}")
	b.WriteString(".artifact-card .art-link:hover{text-decoration:underline;}")
	// Tabs
	b.WriteString(".tabs{display:flex;gap:0;margin-bottom:14px;border-bottom:1px solid #3c3c3c;}")
	b.WriteString(".tab-btn{background:none;border:none;color:#858585;padding:8px 16px;cursor:pointer;font-size:13px;border-bottom:2px solid transparent;}")
	b.WriteString(".tab-btn.active{color:#0078d4;border-bottom-color:#0078d4;} .tab-btn:hover{color:#ccc;}")
	b.WriteString(".tab-content{display:none;} .tab-content.active{display:block;}")
	b.WriteString("pre{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:12px;font-size:12px;overflow:auto;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\" data-run-id=\"")
	b.WriteString(html.EscapeString(runID))
	b.WriteString("\">")
	b.WriteString("<h1>Run Viewer</h1>")
	b.WriteString("<p>Run ID: ")
	b.WriteString(html.EscapeString(runID))
	b.WriteString("</p>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<span class=\"pill\" id=\"run-status\">Loading...</span>")
	b.WriteString("<span class=\"muted\" id=\"run-timing\"></span>")
	b.WriteString("<span class=\"muted\">/api/v2/runs/")
	b.WriteString(html.EscapeString(runID))
	b.WriteString("</span>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"stats\">")
	b.WriteString("<div class=\"stat\"><span>Steps</span><strong id=\"run-steps\">-</strong></div>")
	b.WriteString("<div class=\"stat\"><span>Artifacts</span><strong id=\"run-artifacts\">-</strong></div>")
	b.WriteString("<div class=\"stat\"><span>Logs</span><strong id=\"run-logs\">-</strong></div>")
	b.WriteString("<div class=\"stat\"><span>Tool Calls</span><strong id=\"run-tools\">-</strong></div>")
	b.WriteString("</div>")
	// Tabs
	b.WriteString("<div class=\"tabs\">")
	b.WriteString("<button class=\"tab-btn active\" data-tab=\"steps\">Steps</button>")
	b.WriteString("<button class=\"tab-btn\" data-tab=\"logs\">Logs</button>")
	b.WriteString("<button class=\"tab-btn\" data-tab=\"artifacts\">Artifacts</button>")
	b.WriteString("<button class=\"tab-btn\" data-tab=\"json\">Raw JSON</button>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"tab-content active\" id=\"tab-steps\"><div class=\"step-timeline\" id=\"step-timeline\"><span class=\"muted\">Loading steps...</span></div></div>")
	b.WriteString("<div class=\"tab-content\" id=\"tab-logs\"><div class=\"log-console\" id=\"log-console\"><div class=\"log-line\"><span class=\"log-msg muted\">Loading logs...</span></div></div></div>")
	b.WriteString("<div class=\"tab-content\" id=\"tab-artifacts\"><div class=\"artifact-list\" id=\"artifact-list\"><span class=\"muted\">Loading artifacts...</span></div></div>")
	b.WriteString("<div class=\"tab-content\" id=\"tab-json\"><pre id=\"run-json\">Loading...</pre></div>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const card=document.querySelector('[data-run-id]');")
	b.WriteString("const runId=card?card.dataset.runId:'';")
	b.WriteString("const statusEl=document.getElementById('run-status');")
	b.WriteString("const timingEl=document.getElementById('run-timing');")
	b.WriteString("const stepsEl=document.getElementById('run-steps');")
	b.WriteString("const artifactsEl=document.getElementById('run-artifacts');")
	b.WriteString("const logsEl=document.getElementById('run-logs');")
	b.WriteString("const toolsEl=document.getElementById('run-tools');")
	b.WriteString("const jsonEl=document.getElementById('run-json');")
	b.WriteString("const stepTimeline=document.getElementById('step-timeline');")
	b.WriteString("const logConsole=document.getElementById('log-console');")
	b.WriteString("const artifactList=document.getElementById('artifact-list');")
	b.WriteString("const countOf=(v)=>Array.isArray(v)?v.length:0;")
	b.WriteString(`const escapeHtml=(v)=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)
	// Tab switching
	b.WriteString("document.querySelectorAll('.tab-btn').forEach(btn=>{btn.addEventListener('click',()=>{")
	b.WriteString("document.querySelectorAll('.tab-btn').forEach(b=>b.classList.remove('active'));")
	b.WriteString("document.querySelectorAll('.tab-content').forEach(c=>c.classList.remove('active'));")
	b.WriteString("btn.classList.add('active');document.getElementById('tab-'+btn.dataset.tab).classList.add('active');});});")
	// Render steps
	b.WriteString("function renderSteps(steps){stepTimeline.innerHTML='';if(!steps||!steps.length){stepTimeline.innerHTML='<span class=\"muted\">No steps recorded</span>';return;}")
	b.WriteString("steps.forEach((s,i)=>{const div=document.createElement('div');")
	b.WriteString("const st=(s.status||'pending').toLowerCase();div.className='step-item s-'+st;")
	b.WriteString("const name=s.name||s.type||'Step '+(i+1);")
	b.WriteString("const ts=s.started_at?new Date(s.started_at).toLocaleTimeString():'';")
	b.WriteString("const dur=s.started_at&&s.completed_at?((new Date(s.completed_at)-new Date(s.started_at))/1000).toFixed(1)+'s':'';")
	b.WriteString("div.innerHTML='<div class=\"step-info\"><div class=\"step-name\">'+escapeHtml(name)+'</div><div class=\"step-meta\">'+escapeHtml(st)+' &middot; '+escapeHtml(ts)+'</div></div><div class=\"step-dur\">'+escapeHtml(dur)+'</div>';")
	b.WriteString("stepTimeline.appendChild(div);});}")
	// Render logs
	b.WriteString("function renderLogs(logs){logConsole.innerHTML='';if(!logs||!logs.length){logConsole.innerHTML='<div class=\"log-line\"><span class=\"log-msg muted\">No logs</span></div>';return;}")
	b.WriteString("logs.forEach(log=>{const div=document.createElement('div');div.className='log-line';")
	b.WriteString("const ts=log.timestamp?new Date(log.timestamp).toLocaleTimeString():'';")
	b.WriteString("const lvl=(log.level||'info').toLowerCase();")
	b.WriteString("div.innerHTML='<span class=\"log-ts\">'+escapeHtml(ts)+'</span><span class=\"log-lvl l-'+lvl+'\">'+escapeHtml(lvl)+'</span><span class=\"log-msg\">'+escapeHtml(log.message||log.content||JSON.stringify(log))+'</span>';")
	b.WriteString("logConsole.appendChild(div);});logConsole.scrollTop=logConsole.scrollHeight;}")
	// Render artifacts
	b.WriteString("function renderArtifacts(artifacts){artifactList.innerHTML='';if(!artifacts||!artifacts.length){artifactList.innerHTML='<span class=\"muted\">No artifacts</span>';return;}")
	b.WriteString("artifacts.forEach(a=>{const div=document.createElement('div');div.className='artifact-card';")
	b.WriteString("const name=a.name||a.path||a.id||'Artifact';const mime=a.mime_type||a.content_type||'';")
	b.WriteString("const size=a.size?((a.size/1024).toFixed(1)+' KB'):'';")
	b.WriteString("div.innerHTML='<div class=\"art-name\">'+escapeHtml(name)+'</div><div class=\"art-meta\">'+escapeHtml(mime+(size?' &middot; '+size:''))+'</div>'")
	b.WriteString("+(a.id?'<a class=\"art-link\" href=\"/api/v2/artifacts/'+encodeURIComponent(a.id)+'/content\" target=\"_blank\">Download</a>':'');")
	b.WriteString("artifactList.appendChild(div);});}")
	// Load run
	b.WriteString("async function loadRun(){statusEl.textContent='Loading...';")
	b.WriteString("try{const res=await fetch('/api/v2/runs/'+encodeURIComponent(runId));")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const run=data&&data.run?data.run:data;")
	b.WriteString("const status=(run&&run.status?run.status:'UNKNOWN').toUpperCase();")
	b.WriteString("statusEl.textContent=status;statusEl.className='pill s-'+status;")
	b.WriteString("stepsEl.textContent=countOf(run&&run.steps);")
	b.WriteString("artifactsEl.textContent=countOf(run&&run.artifacts);logsEl.textContent=countOf(run&&run.logs);")
	b.WriteString("toolsEl.textContent=countOf(run&&run.tool_invocations);")
	b.WriteString("if(run&&run.started_at){let t='Started: '+new Date(run.started_at).toLocaleString();if(run.completed_at)t+=' | Ended: '+new Date(run.completed_at).toLocaleString();timingEl.textContent=t;}")
	b.WriteString("renderSteps(run&&run.steps);renderLogs(run&&run.logs);renderArtifacts(run&&run.artifacts);")
	b.WriteString("jsonEl.textContent=JSON.stringify(run,null,2);")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load run';")
	b.WriteString("jsonEl.textContent=String(err&&err.message?err.message:err);}")
	b.WriteString("}")
	b.WriteString("loadRun();setInterval(loadRun,10000);")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	return b.String()
}

func contextPackDetailPage(packID string) string {
	var b strings.Builder
	b.WriteString(subPageHead("Context Pack"))
	b.WriteString("<style>")
	b.WriteString("ul{list-style:none;padding:0;margin:0;display:grid;gap:6px;font-size:12px;}")
	b.WriteString("li{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:4px;padding:6px 8px;}")
	b.WriteString(".panel{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;padding:12px;}")
	b.WriteString(".panel span{display:block;font-size:11px;color:#858585;margin-bottom:6px;}")
	b.WriteString(".summary{font-size:13px;color:#ccc;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\" data-pack-id=\"")
	b.WriteString(html.EscapeString(packID))
	b.WriteString("\">")
	b.WriteString("<h1>Context Pack</h1>")
	b.WriteString("<p>Pack ID: ")
	b.WriteString(html.EscapeString(packID))
	b.WriteString("</p>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<span class=\"pill\" id=\"pack-status\">Loading...</span>")
	b.WriteString("<span class=\"muted\">/api/v2/context-packs/")
	b.WriteString(html.EscapeString(packID))
	b.WriteString("</span>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"grid\">")
	b.WriteString("<div class=\"panel\"><span>Summary</span><div id=\"pack-summary\" class=\"summary\">-</div></div>")
	b.WriteString("<div class=\"panel\"><span>Files</span><ul id=\"pack-files\"></ul></div>")
	b.WriteString("</div>")
	b.WriteString("<pre id=\"pack-json\">Loading...</pre>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const card=document.querySelector('[data-pack-id]');")
	b.WriteString("const packId=card?card.dataset.packId:'';")
	b.WriteString("const statusEl=document.getElementById('pack-status');")
	b.WriteString("const summaryEl=document.getElementById('pack-summary');")
	b.WriteString("const filesEl=document.getElementById('pack-files');")
	b.WriteString("const jsonEl=document.getElementById('pack-json');")
	b.WriteString("function setFiles(files){filesEl.innerHTML='';")
	b.WriteString("if(!files.length){const li=document.createElement('li');li.className='muted';li.textContent='No files';filesEl.appendChild(li);return;}")
	b.WriteString("files.forEach((file)=>{const li=document.createElement('li');")
	b.WriteString("const path=file&&file.path?file.path:'(unknown path)';")
	b.WriteString("const snippets=Array.isArray(file&&file.snippets)?file.snippets.length:0;")
	b.WriteString("li.textContent=snippets>0?path+' ('+snippets+' snippets)':path;filesEl.appendChild(li);});}")
	b.WriteString("async function loadPack(){statusEl.textContent='Loading...';")
	b.WriteString("summaryEl.textContent='-';jsonEl.textContent='Loading...';")
	b.WriteString("try{const res=await fetch('/api/v2/context-packs/'+encodeURIComponent(packId));")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const pack=data&&data.context_pack?data.context_pack:data;")
	b.WriteString("statusEl.textContent='Loaded';")
	b.WriteString("summaryEl.textContent=pack&&pack.summary?pack.summary:'No summary';")
	b.WriteString("const contents=pack&&pack.contents?pack.contents:{};")
	b.WriteString("const files=Array.isArray(contents.files)?contents.files:[];setFiles(files);")
	b.WriteString("jsonEl.textContent=JSON.stringify(pack,null,2);")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load pack';")
	b.WriteString("summaryEl.textContent='-';filesEl.innerHTML='';")
	b.WriteString("const li=document.createElement('li');li.className='muted';")
	b.WriteString("li.textContent='Error loading files';filesEl.appendChild(li);")
	b.WriteString("jsonEl.textContent=String(err&&err.message?err.message:err);}")
	b.WriteString("}")
	b.WriteString("loadPack();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	return b.String()
}

func uiPlaceholderHandler(title string, subtitle string, details []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, uiPlaceholderPage(title, subtitle, details))
	}
}

func taskGraphsPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/graphs/tasks" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Task DAG"))
	b.WriteString("<style>")
	b.WriteString("*{box-sizing:border-box;}")
	b.WriteString(".legend{display:flex;gap:12px;flex-wrap:wrap;font-size:11px;margin-bottom:12px;}")
	b.WriteString(".legend-item{display:flex;align-items:center;gap:4px;}")
	b.WriteString(".legend-dot{width:10px;height:10px;border-radius:50%;}")
	b.WriteString(".graph-canvas{position:relative;min-height:400px;background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;overflow:auto;}")
	b.WriteString("svg.dag{width:100%;height:100%;position:absolute;top:0;left:0;pointer-events:none;}")
	b.WriteString("svg.dag line{stroke-width:1.5;}")
	b.WriteString("svg.dag polygon{}")
	b.WriteString(".node{position:absolute;background:#252526;border:2px solid #3c3c3c;border-radius:8px;padding:8px 10px;font-size:11px;cursor:pointer;min-width:120px;z-index:1;transition:box-shadow 0.15s;}")
	b.WriteString(".node:hover{box-shadow:0 0 0 2px #0078d4;z-index:10;}")
	b.WriteString(".node .node-id{font-weight:700;color:#0078d4;margin-bottom:2px;}")
	b.WriteString(".node .node-title{color:#ccc;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:160px;}")
	b.WriteString(".node .node-status{font-size:10px;margin-top:3px;font-weight:600;text-transform:uppercase;}")
	b.WriteString(".node.s-PENDING{border-color:#858585;} .node.s-PENDING .node-status{color:#858585;}")
	b.WriteString(".node.s-READY{border-color:#3794ff;} .node.s-READY .node-status{color:#3794ff;}")
	b.WriteString(".node.s-IN_PROGRESS{border-color:#0078d4;} .node.s-IN_PROGRESS .node-status{color:#0078d4;}")
	b.WriteString(".node.s-BLOCKED{border-color:#f14c4c;} .node.s-BLOCKED .node-status{color:#f14c4c;}")
	b.WriteString(".node.s-DONE,.node.s-SUCCEEDED{border-color:#89d185;} .node.s-DONE .node-status,.node.s-SUCCEEDED .node-status{color:#89d185;}")
	b.WriteString(".node.s-CANCELED{border-color:#858585;} .node.s-CANCELED .node-status{color:#858585;}")
	b.WriteString(".node.s-FAILED{border-color:#f14c4c;} .node.s-FAILED .node-status{color:#f14c4c;}")
	b.WriteString(".node.s-REVIEW{border-color:#cca700;} .node.s-REVIEW .node-status{color:#cca700;}")
	b.WriteString(".node.critical-path{box-shadow:0 0 0 3px #cca700;} .node.cycle-node{box-shadow:0 0 0 3px #f14c4c;} .node.parallel-node{box-shadow:0 0 0 3px #3794ff;}")
	b.WriteString(".analysis-bar{background:#252526;border:1px solid #3c3c3c;border-radius:8px;padding:10px 14px;margin-bottom:12px;font-size:12px;color:#858585;display:flex;gap:18px;flex-wrap:wrap;}")
	b.WriteString(".analysis-bar .ab-item{display:flex;align-items:center;gap:5px;} .analysis-bar .ab-dot{width:8px;height:8px;border-radius:50%;}")
	b.WriteString(".info-panel{display:none;position:fixed;top:80px;right:32px;width:280px;background:#252526;border:1px solid #3c3c3c;border-radius:10px;padding:14px;font-size:12px;z-index:100;box-shadow:0 8px 32px rgba(0,0,0,0.5);}")
	b.WriteString(".info-panel.open{display:block;}")
	b.WriteString(".info-panel h3{margin:0 0 8px;font-size:14px;color:#ccc;}")
	b.WriteString(".info-panel .close{position:absolute;top:8px;right:10px;cursor:pointer;color:#858585;font-size:16px;}")
	b.WriteString(".info-panel .row{display:flex;justify-content:space-between;padding:3px 0;border-bottom:1px solid #3c3c3c;}")
	b.WriteString(".info-panel .label{color:#858585;} .info-panel .value{color:#ccc;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Task DAG</h1>")
	b.WriteString("<p>Interactive dependency graph from <span class=\"muted\" id=\"graph-query\">/api/v2/projects/proj_default/graphs/tasks</span></p>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<label class=\"field\">Project<input class=\"input\" id=\"graph-project\" value=\"proj_default\"></label>")
	b.WriteString("<label class=\"field\">Status<select class=\"select\" id=\"graph-status-filter\">")
	b.WriteString("<option value=\"\">All</option>")
	b.WriteString("<option value=\"PENDING\">PENDING</option><option value=\"READY\">READY</option>")
	b.WriteString("<option value=\"IN_PROGRESS\">IN_PROGRESS</option><option value=\"BLOCKED\">BLOCKED</option>")
	b.WriteString("<option value=\"DONE\">DONE</option><option value=\"CANCELED\">CANCELED</option>")
	b.WriteString("<option value=\"FAILED\">FAILED</option></select></label>")
	b.WriteString("<span id=\"graph-status\">Loading...</span>")
	b.WriteString("<button class=\"btn\" id=\"graph-refresh\" type=\"button\">Refresh</button></div>")
	b.WriteString("<div class=\"legend\">")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#858585\"></span>Pending</span>")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#3794ff\"></span>Ready</span>")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#0078d4\"></span>In Progress</span>")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#f14c4c\"></span>Blocked</span>")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#89d185\"></span>Done</span>")
	b.WriteString("<span class=\"legend-item\"><span class=\"legend-dot\" style=\"background:#f14c4c\"></span>Failed</span>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"graph-canvas\" id=\"graph-canvas\"><svg class=\"dag\" id=\"dag-svg\"></svg></div>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"info-panel\" id=\"info-panel\"><span class=\"close\" id=\"info-close\">&times;</span>")
	b.WriteString("<h3 id=\"info-title\"></h3><div id=\"info-body\"></div></div>")
	b.WriteString("<script>")
	b.WriteString("const canvasEl=document.getElementById('graph-canvas');")
	b.WriteString("const svgEl=document.getElementById('dag-svg');")
	b.WriteString("const statusEl=document.getElementById('graph-status');")
	b.WriteString("const queryEl=document.getElementById('graph-query');")
	b.WriteString("const refreshBtn=document.getElementById('graph-refresh');")
	b.WriteString("const projectEl=document.getElementById('graph-project');")
	b.WriteString("const statusFilterEl=document.getElementById('graph-status-filter');")
	b.WriteString("const infoPanel=document.getElementById('info-panel');")
	b.WriteString("const infoTitle=document.getElementById('info-title');")
	b.WriteString("const infoBody=document.getElementById('info-body');")
	b.WriteString("const infoClose=document.getElementById('info-close');")
	b.WriteString(`const escapeHtml=(v)=>String(v??'').replace(/[&<>"']/g,(c)=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)
	b.WriteString("infoClose.addEventListener('click',()=>infoPanel.classList.remove('open'));")
	b.WriteString("const statusColors={PENDING:'#858585',READY:'#3794ff',IN_PROGRESS:'#0078d4',BLOCKED:'#f14c4c',DONE:'#89d185',SUCCEEDED:'#89d185',CANCELED:'#858585',FAILED:'#f14c4c',REVIEW:'#cca700'};")
	// Layout algorithm: Kahn's topological sort to assign layers, then position nodes
	b.WriteString("function layoutDAG(nodes,edges){")
	b.WriteString("const nodeMap=new Map();nodes.forEach(n=>nodeMap.set(String(n.id),n));")
	b.WriteString("const inDeg=new Map();const adj=new Map();")
	b.WriteString("nodes.forEach(n=>{const id=String(n.id);inDeg.set(id,0);adj.set(id,[]);});")
	b.WriteString("edges.forEach(e=>{const from=String(e.from);const to=String(e.to);")
	b.WriteString("if(adj.has(from)&&inDeg.has(to)){adj.get(from).push(to);inDeg.set(to,(inDeg.get(to)||0)+1);}});")
	// Kahn's algorithm for topological layers
	b.WriteString("const layers=[];const queue=[];")
	b.WriteString("inDeg.forEach((d,id)=>{if(d===0)queue.push(id);});")
	b.WriteString("const visited=new Set();")
	b.WriteString("while(queue.length>0){const layer=[...queue];queue.length=0;layers.push(layer);")
	b.WriteString("layer.forEach(id=>{visited.add(id);(adj.get(id)||[]).forEach(to=>{")
	b.WriteString("inDeg.set(to,(inDeg.get(to)||0)-1);if(inDeg.get(to)===0&&!visited.has(to))queue.push(to);});});}")
	// Handle nodes not in any layer (cycles or disconnected)
	b.WriteString("nodes.forEach(n=>{if(!visited.has(String(n.id))){")
	b.WriteString("if(layers.length===0)layers.push([]);layers[layers.length-1].push(String(n.id));}});")
	// Assign positions
	b.WriteString("const positions=new Map();const nodeW=140;const nodeH=70;const padX=40;const padY=30;")
	b.WriteString("const gapX=nodeW+padX;const gapY=nodeH+padY;")
	b.WriteString("let maxX=0;")
	b.WriteString("layers.forEach((layer,li)=>{layer.forEach((id,ni)=>{")
	b.WriteString("const x=ni*gapX+padX;const y=li*gapY+padY;")
	b.WriteString("positions.set(id,{x,y});if(x+nodeW>maxX)maxX=x+nodeW;});});")
	b.WriteString("const totalH=layers.length*gapY+padY;const totalW=maxX+padX;")
	b.WriteString("return {positions,totalW,totalH,nodeW,nodeH};}")

	// Critical path analysis: longest path in DAG using dynamic programming
	b.WriteString("function findCriticalPath(nodes,edges){")
	b.WriteString("const adj=new Map();const ids=new Set();")
	b.WriteString("nodes.forEach(n=>{const id=String(n.id);ids.add(id);adj.set(id,[]);});")
	b.WriteString("edges.forEach(e=>{const f=String(e.from);const t=String(e.to);if(adj.has(f))adj.get(f).push(t);});")
	b.WriteString("const dist=new Map();const prev=new Map();ids.forEach(id=>{dist.set(id,0);prev.set(id,null);});")
	b.WriteString("const visited=new Set();const order=[];")
	b.WriteString("function dfs(id){if(visited.has(id))return;visited.add(id);(adj.get(id)||[]).forEach(t=>dfs(t));order.push(id);}")
	b.WriteString("ids.forEach(id=>dfs(id));order.reverse();")
	b.WriteString("order.forEach(id=>{(adj.get(id)||[]).forEach(t=>{if(dist.get(id)+1>dist.get(t)){dist.set(t,dist.get(id)+1);prev.set(t,id);}});});")
	b.WriteString("let maxDist=0;let endNode=null;dist.forEach((d,id)=>{if(d>maxDist){maxDist=d;endNode=id;}});")
	b.WriteString("const path=[];let cur=endNode;while(cur!==null){path.unshift(cur);cur=prev.get(cur);}return {path,length:maxDist};}")

	// Cycle detection using DFS coloring
	b.WriteString("function detectCycles(nodes,edges){")
	b.WriteString("const adj=new Map();nodes.forEach(n=>adj.set(String(n.id),[]));")
	b.WriteString("edges.forEach(e=>{const f=String(e.from);if(adj.has(f))adj.get(f).push(String(e.to));});")
	b.WriteString("const white=new Set(nodes.map(n=>String(n.id)));const gray=new Set();const black=new Set();")
	b.WriteString("const cycles=[];")
	b.WriteString("function dfs(id,stack){white.delete(id);gray.add(id);stack.push(id);")
	b.WriteString("for(const t of (adj.get(id)||[])){if(gray.has(t)){const ci=stack.indexOf(t);if(ci>=0)cycles.push(stack.slice(ci));}")
	b.WriteString("else if(white.has(t))dfs(t,[...stack]);}gray.delete(id);black.add(id);}")
	b.WriteString("while(white.size>0){const id=white.values().next().value;dfs(id,[]);}return cycles;}")

	// Parallel execution: nodes at same topological layer with no dependencies between them
	b.WriteString("function findParallelGroups(nodes,edges){")
	b.WriteString("const inDeg=new Map();const adj=new Map();")
	b.WriteString("nodes.forEach(n=>{const id=String(n.id);inDeg.set(id,0);adj.set(id,[]);});")
	b.WriteString("edges.forEach(e=>{const f=String(e.from);const t=String(e.to);if(adj.has(f)&&inDeg.has(t)){adj.get(f).push(t);inDeg.set(t,(inDeg.get(t)||0)+1);}});")
	b.WriteString("const layers=[];const queue=[];const visited=new Set();")
	b.WriteString("inDeg.forEach((d,id)=>{if(d===0)queue.push(id);});")
	b.WriteString("while(queue.length>0){const layer=[...queue];queue.length=0;layers.push(layer);")
	b.WriteString("layer.forEach(id=>{visited.add(id);(adj.get(id)||[]).forEach(t=>{inDeg.set(t,(inDeg.get(t)||0)-1);if(inDeg.get(t)===0&&!visited.has(t))queue.push(t);});});}")
	b.WriteString("return layers.filter(l=>l.length>1);}")

	// Render function
	b.WriteString("function renderGraph(data){")
	b.WriteString("const nodes=data.nodes||[];const edges=data.edges||[];")
	b.WriteString("const statusFilter=statusFilterEl.value;")
	b.WriteString("const filteredNodes=statusFilter?nodes.filter(n=>(n.status||'').toUpperCase()===statusFilter):nodes;")
	b.WriteString("const filteredIds=new Set(filteredNodes.map(n=>String(n.id)));")
	b.WriteString("const filteredEdges=edges.filter(e=>filteredIds.has(String(e.from))&&filteredIds.has(String(e.to)));")
	// Run analysis
	b.WriteString("const cp=findCriticalPath(filteredNodes,filteredEdges);")
	b.WriteString("const cycles=detectCycles(filteredNodes,filteredEdges);")
	b.WriteString("const parallel=findParallelGroups(filteredNodes,filteredEdges);")
	b.WriteString("const cpSet=new Set(cp.path);")
	b.WriteString("const cycleNodes=new Set();cycles.forEach(c=>c.forEach(id=>cycleNodes.add(id)));")
	b.WriteString("const parallelNodes=new Set();parallel.forEach(g=>g.forEach(id=>parallelNodes.add(id)));")
	// Build analysis summary
	b.WriteString("let analysis='';")
	b.WriteString("if(cp.path.length>1)analysis+=' | Critical path: '+cp.path.length+' nodes (len '+cp.length+')';")
	b.WriteString("if(cycles.length>0)analysis+=' | \\u26A0 '+cycles.length+' cycle(s) detected!';")
	b.WriteString("if(parallel.length>0)analysis+=' | '+parallel.length+' parallel group(s)';")
	b.WriteString("statusEl.textContent=filteredNodes.length+' nodes, '+filteredEdges.length+' edges'+analysis;")
	b.WriteString("canvasEl.querySelectorAll('.node').forEach(el=>el.remove());")
	b.WriteString("svgEl.innerHTML='';")
	b.WriteString("if(filteredNodes.length===0){canvasEl.style.minHeight='100px';return;}")
	b.WriteString("const {positions,totalW,totalH,nodeW,nodeH}=layoutDAG(filteredNodes,filteredEdges);")
	b.WriteString("canvasEl.style.width=totalW+'px';canvasEl.style.height=totalH+'px';canvasEl.style.minHeight=totalH+'px';")
	b.WriteString("svgEl.setAttribute('width',totalW);svgEl.setAttribute('height',totalH);svgEl.setAttribute('viewBox','0 0 '+totalW+' '+totalH);")
	// Render analysis bar above the canvas
	b.WriteString("let oldBar=document.getElementById('analysis-bar');if(oldBar)oldBar.remove();")
	b.WriteString("if(cp.path.length>1||cycles.length>0||parallel.length>0){")
	b.WriteString("const bar=document.createElement('div');bar.id='analysis-bar';bar.className='analysis-bar';")
	b.WriteString("let barHtml='';")
	b.WriteString("if(cp.path.length>1)barHtml+='<span class=\"ab-item\"><span class=\"ab-dot\" style=\"background:#f59e0b\"></span>Critical Path: '+cp.path.length+' nodes</span>';")
	b.WriteString("if(cycles.length>0)barHtml+='<span class=\"ab-item\"><span class=\"ab-dot\" style=\"background:#ef4444\"></span>\\u26A0 '+cycles.length+' cycle(s)</span>';")
	b.WriteString("if(parallel.length>0)barHtml+='<span class=\"ab-item\"><span class=\"ab-dot\" style=\"background:#38bdf8\"></span>'+parallel.length+' parallel group(s)</span>';")
	b.WriteString("bar.innerHTML=barHtml;canvasEl.parentNode.insertBefore(bar,canvasEl);}")
	// Draw edges as SVG lines with arrowheads
	b.WriteString("const ns='http://www.w3.org/2000/svg';")
	b.WriteString("const defs=document.createElementNS(ns,'defs');")
	b.WriteString("const depMarker=document.createElementNS(ns,'marker');")
	b.WriteString("depMarker.setAttribute('id','arrow-dep');depMarker.setAttribute('viewBox','0 0 10 10');")
	b.WriteString("depMarker.setAttribute('refX','10');depMarker.setAttribute('refY','5');")
	b.WriteString("depMarker.setAttribute('markerWidth','6');depMarker.setAttribute('markerHeight','6');")
	b.WriteString("depMarker.setAttribute('orient','auto-start-reverse');")
	b.WriteString("const poly=document.createElementNS(ns,'path');poly.setAttribute('d','M 0 0 L 10 5 L 0 10 z');poly.setAttribute('fill','#0078d4');")
	b.WriteString("depMarker.appendChild(poly);defs.appendChild(depMarker);")
	b.WriteString("const pcMarker=document.createElementNS(ns,'marker');")
	b.WriteString("pcMarker.setAttribute('id','arrow-pc');pcMarker.setAttribute('viewBox','0 0 10 10');")
	b.WriteString("pcMarker.setAttribute('refX','10');pcMarker.setAttribute('refY','5');")
	b.WriteString("pcMarker.setAttribute('markerWidth','6');pcMarker.setAttribute('markerHeight','6');")
	b.WriteString("pcMarker.setAttribute('orient','auto-start-reverse');")
	b.WriteString("const poly2=document.createElementNS(ns,'path');poly2.setAttribute('d','M 0 0 L 10 5 L 0 10 z');poly2.setAttribute('fill','#3c3c3c');")
	b.WriteString("pcMarker.appendChild(poly2);defs.appendChild(pcMarker);")
	b.WriteString("svgEl.appendChild(defs);")
	b.WriteString("filteredEdges.forEach(e=>{const from=positions.get(String(e.from));const to=positions.get(String(e.to));if(!from||!to)return;")
	b.WriteString("const line=document.createElementNS(ns,'line');")
	b.WriteString("line.setAttribute('x1',from.x+nodeW/2);line.setAttribute('y1',from.y+nodeH);")
	b.WriteString("line.setAttribute('x2',to.x+nodeW/2);line.setAttribute('y2',to.y);")
	b.WriteString("const isDep=e.type==='depends_on';")
	b.WriteString("const isCpEdge=cpSet.has(String(e.from))&&cpSet.has(String(e.to));")
	b.WriteString("line.setAttribute('stroke',isCpEdge?'#cca700':isDep?'#0078d4':'#3c3c3c');")
	b.WriteString("line.setAttribute('stroke-width',isCpEdge?'3':'1.5');")
	b.WriteString("line.setAttribute('stroke-dasharray',isDep?'':'5,3');")
	b.WriteString("line.setAttribute('marker-end',isDep?'url(#arrow-dep)':'url(#arrow-pc)');")
	b.WriteString("svgEl.appendChild(line);});")
	// Draw nodes as positioned divs
	b.WriteString("filteredNodes.forEach(n=>{const pos=positions.get(String(n.id));if(!pos)return;")
	b.WriteString("const div=document.createElement('div');")
	b.WriteString("const st=(n.status||'PENDING').toUpperCase();")
	b.WriteString("const nid=String(n.id);")
	b.WriteString("let extraCls='';if(cpSet.has(nid))extraCls+=' critical-path';if(cycleNodes.has(nid))extraCls+=' cycle-node';if(parallelNodes.has(nid))extraCls+=' parallel-node';")
	b.WriteString("div.className='node s-'+st+extraCls;")
	b.WriteString("div.style.left=pos.x+'px';div.style.top=pos.y+'px';div.style.width=nodeW+'px';")
	b.WriteString("let badges='';if(cpSet.has(nid))badges+='<span style=\"color:#f59e0b;font-size:9px;\">\\u2B50 CP</span> ';")
	b.WriteString("if(cycleNodes.has(nid))badges+='<span style=\"color:#ef4444;font-size:9px;\">\\u26A0 CYCLE</span> ';")
	b.WriteString("if(parallelNodes.has(nid))badges+='<span style=\"color:#38bdf8;font-size:9px;\">\\u2693 PAR</span> ';")
	b.WriteString("div.innerHTML='<div class=\"node-id\">#'+escapeHtml(n.id)+'</div><div class=\"node-title\">'+escapeHtml(n.title||'')+'</div><div class=\"node-status\">'+badges+escapeHtml(st)+'</div>';")
	b.WriteString("div.addEventListener('click',()=>{infoTitle.textContent='Task #'+n.id;")
	b.WriteString("infoBody.innerHTML='';const fields={Status:st,Type:n.type||'-',Title:n.title||'-',Parent:n.parent_task_id||'-','Critical Path':cpSet.has(nid)?'Yes':'No','In Cycle':cycleNodes.has(nid)?'Yes':'No','Can Parallelize':parallelNodes.has(nid)?'Yes':'No'};")
	b.WriteString("Object.entries(fields).forEach(([k,v])=>{const row=document.createElement('div');row.className='row';")
	b.WriteString("row.innerHTML='<span class=\"label\">'+k+'</span><span class=\"value\">'+escapeHtml(v)+'</span>';infoBody.appendChild(row);});")
	b.WriteString("infoPanel.classList.add('open');});")
	b.WriteString("canvasEl.appendChild(div);});}")

	// Load data
	b.WriteString("let graphData=null;")
	b.WriteString("async function loadGraph(){statusEl.textContent='Loading...';try{")
	b.WriteString("const pid=String(projectEl.value||'proj_default').trim();")
	b.WriteString("const url='/api/v2/projects/'+encodeURIComponent(pid)+'/graphs/tasks';queryEl.textContent=url;")
	b.WriteString("const res=await fetch(url);if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("graphData=await res.json();renderGraph(graphData);")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load graph';}}")
	b.WriteString("refreshBtn.addEventListener('click',loadGraph);")
	b.WriteString("projectEl.addEventListener('change',loadGraph);")
	b.WriteString("statusFilterEl.addEventListener('change',()=>{if(graphData)renderGraph(graphData);});")
	b.WriteString("loadGraph();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func memoryPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/memory" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Memory"))
	b.WriteString("<style>")
	b.WriteString(".form{display:grid;gap:10px;margin:12px 0 16px;}")
	b.WriteString(".row{display:flex;gap:12px;flex-wrap:wrap;}")
	b.WriteString(".form-actions{display:flex;align-items:center;gap:12px;flex-wrap:wrap;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Memory</h1>")
	b.WriteString("<div class=\"row\" style=\"margin-bottom:10px;\"><label class=\"field\">Project<input class=\"input\" id=\"memory-project\" value=\"proj_default\" style=\"width:200px;\"></label></div>")
	b.WriteString("<p>Latest memory from <span class=\"muted\" id=\"memory-endpoint\">/api/v2/projects/proj_default/memory?limit=50</span></p>")
	b.WriteString("<div class=\"meta\"><span id=\"memory-status\">Loading...</span>")
	b.WriteString("<button class=\"btn\" id=\"memory-refresh\" type=\"button\">Refresh</button></div>")
	b.WriteString("<form class=\"form\" id=\"memory-form\">")
	b.WriteString("<div class=\"row\">")
	b.WriteString("<label class=\"field\">Scope<input class=\"input\" id=\"memory-scope\" name=\"scope\" placeholder=\"GLOBAL\" required></label>")
	b.WriteString("<label class=\"field\">Key<input class=\"input\" id=\"memory-key\" name=\"key\" placeholder=\"release_notes\" required></label>")
	b.WriteString("</div>")
	b.WriteString("<label class=\"field\">Value JSON<textarea class=\"textarea\" id=\"memory-value\" name=\"value\" placeholder='{\"note\":\"...\"}' required></textarea></label>")
	b.WriteString("<div class=\"form-actions\"><button class=\"btn\" id=\"memory-submit\" type=\"submit\">Save</button><span class=\"muted\" id=\"memory-form-status\"></span></div>")
	b.WriteString("</form>")
	b.WriteString("<table><thead><tr><th>Scope</th><th>Key</th><th>Updated</th></tr></thead>")
	b.WriteString("<tbody id=\"memory-body\"><tr><td colspan=\"3\">Loading...</td></tr></tbody></table>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const bodyEl=document.getElementById('memory-body');")
	b.WriteString("const statusEl=document.getElementById('memory-status');")
	b.WriteString("const refreshBtn=document.getElementById('memory-refresh');")
	b.WriteString("const formEl=document.getElementById('memory-form');")
	b.WriteString("const scopeEl=document.getElementById('memory-scope');")
	b.WriteString("const keyEl=document.getElementById('memory-key');")
	b.WriteString("const valueEl=document.getElementById('memory-value');")
	b.WriteString("const formStatusEl=document.getElementById('memory-form-status');")
	b.WriteString("const projectEl=document.getElementById('memory-project');")
	b.WriteString("const endpointEl=document.getElementById('memory-endpoint');")
	b.WriteString("function getProjectID(){return String(projectEl.value||'proj_default').trim();}")
	b.WriteString(`const escapeHtml=(v)=>String(v??'').replace(/[&<>"']/g,(c)=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)
	b.WriteString("async function loadMemory(){const pid=getProjectID();statusEl.textContent='Loading...';")
	b.WriteString("endpointEl.textContent='/api/v2/projects/'+pid+'/memory?limit=50';")
	b.WriteString("bodyEl.innerHTML='<tr><td colspan=\"3\">Loading...</td></tr>';try{")
	b.WriteString("const res=await fetch('/api/v2/projects/'+encodeURIComponent(pid)+'/memory?limit=50');")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const items=Array.isArray(data.items)?data.items:[];")
	b.WriteString("statusEl.textContent=items.length+' items';")
	b.WriteString("if(items.length===0){bodyEl.innerHTML='<tr><td colspan=\"3\">No memory</td></tr>';return;}")
	b.WriteString("bodyEl.innerHTML='';items.forEach((item)=>{")
	b.WriteString("const tr=document.createElement('tr');")
	b.WriteString("const scope=escapeHtml(item.scope);const key=escapeHtml(item.key);")
	b.WriteString("const updated=escapeHtml(item.updated_at||'');")
	b.WriteString("tr.innerHTML='<td>'+scope+'</td><td>'+key+'</td><td>'+updated+'</td>';bodyEl.appendChild(tr);});")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load memory';")
	b.WriteString("bodyEl.innerHTML='<tr><td colspan=\"3\">Error loading memory</td></tr>';}}")
	b.WriteString("async function submitMemory(event){event.preventDefault();")
	b.WriteString("const scope=String(scopeEl.value||'').trim();")
	b.WriteString("const key=String(keyEl.value||'').trim();")
	b.WriteString("const rawValue=String(valueEl.value||'').trim();")
	b.WriteString("if(!scope||!key||!rawValue){formStatusEl.textContent='Scope, key, and value are required.';return;}")
	b.WriteString("let valueJson=null;try{valueJson=JSON.parse(rawValue);}catch(err){formStatusEl.textContent='Value must be valid JSON.';return;}")
	b.WriteString("formStatusEl.textContent='Saving...';")
	b.WriteString("try{const pid=getProjectID();const res=await fetch('/api/v2/projects/'+encodeURIComponent(pid)+'/memory',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({scope:scope,key:key,value:valueJson})});")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("formStatusEl.textContent='Saved.';await loadMemory();}catch(err){formStatusEl.textContent='Save failed.';}}")
	b.WriteString("refreshBtn.addEventListener('click',loadMemory);")
	b.WriteString("formEl.addEventListener('submit',submitMemory);")
	b.WriteString("loadMemory();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func agentsPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/agents" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Agents"))
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Agents</h1>")
	b.WriteString("<p>Latest agents from <span class=\"muted\">/api/v2/agents</span></p>")
	b.WriteString("<div class=\"meta\"><span id=\"agents-status\">Loading...</span>")
	b.WriteString("<label class=\"field\">Status<select class=\"select\" id=\"agents-status-filter\"><option value=\"\">All</option><option value=\"active\">Active</option><option value=\"stale\">Stale</option></select></label>")
	b.WriteString("<label class=\"field\">Limit<input class=\"input\" id=\"agents-limit\" type=\"number\" min=\"1\" max=\"200\" step=\"1\" value=\"50\"></label>")
	b.WriteString("<button class=\"btn\" id=\"agents-refresh\" type=\"button\">Refresh</button></div>")
	b.WriteString("<table><thead><tr><th>ID</th><th>Status</th><th>Last Seen</th></tr></thead>")
	b.WriteString("<tbody id=\"agents-body\"><tr><td colspan=\"3\">Loading...</td></tr></tbody></table>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const bodyEl=document.getElementById('agents-body');")
	b.WriteString("const statusEl=document.getElementById('agents-status');")
	b.WriteString("const refreshBtn=document.getElementById('agents-refresh');")
	b.WriteString("const statusFilterEl=document.getElementById('agents-status-filter');")
	b.WriteString("const limitEl=document.getElementById('agents-limit');")
	b.WriteString(`const escapeHtml=(v)=>String(v??'').replace(/[&<>"']/g,(c)=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)
	b.WriteString("function getLimit(){const raw=Number.parseInt(limitEl.value,10);if(Number.isNaN(raw)||raw<1){return 50;}return Math.min(raw,200);}")
	b.WriteString("async function loadAgents(){statusEl.textContent='Loading...';")
	b.WriteString("bodyEl.innerHTML='<tr><td colspan=\"3\">Loading...</td></tr>';try{")
	b.WriteString("const params=new URLSearchParams();const limit=getLimit();params.set('limit',String(limit));const status=String(statusFilterEl.value||'').trim();if(status){params.set('status',status);}const res=await fetch('/api/v2/agents?'+params.toString());if(!res.ok)throw new Error();")
	b.WriteString("const data=await res.json();const agents=Array.isArray(data.agents)?data.agents:[];")
	b.WriteString("statusEl.textContent=agents.length+' agents';")
	b.WriteString("if(agents.length===0){bodyEl.innerHTML='<tr><td colspan=\"3\">No agents</td></tr>';return;}")
	b.WriteString("bodyEl.innerHTML='';agents.forEach((agent)=>{")
	b.WriteString("const tr=document.createElement('tr');")
	b.WriteString("const id=escapeHtml(agent.id);const status=escapeHtml(agent.status);const lastSeen=escapeHtml(agent.last_seen||'');")
	b.WriteString("tr.innerHTML='<td>'+id+'</td><td>'+status+'</td><td>'+lastSeen+'</td>';bodyEl.appendChild(tr);});")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load agents';")
	b.WriteString("bodyEl.innerHTML='<tr><td colspan=\"3\">Error loading agents</td></tr>';}}")
	b.WriteString("refreshBtn.addEventListener('click',loadAgents);")
	b.WriteString("statusFilterEl.addEventListener('change',loadAgents);")
	b.WriteString("limitEl.addEventListener('change',loadAgents);")
	b.WriteString("loadAgents();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func auditPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/audit" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Audit"))
	b.WriteString("<style>")
	b.WriteString("ul{list-style:none;padding:0;margin:0;}")
	b.WriteString("li{padding:10px 8px;border-bottom:1px solid #3c3c3c;font-size:12px;display:grid;grid-template-columns:2fr 2fr 2fr 2fr 3fr;gap:12px;}")
	b.WriteString("li.header{color:#0078d4;font-weight:600;font-size:11px;text-transform:uppercase;letter-spacing:.05em;border-bottom:1px solid #505050;}")
	b.WriteString(".payload-cell{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:11px;color:#a0a0a0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:400px;cursor:pointer;}")
	b.WriteString(".payload-cell:hover{white-space:normal;overflow:visible;color:#ccc;}")
	b.WriteString(".audit-filters{display:flex;gap:12px;flex-wrap:wrap;align-items:end;margin-bottom:12px;}")
	b.WriteString(".audit-filters label{display:flex;flex-direction:column;gap:4px;font-size:11px;color:#858585;}")
	b.WriteString(".audit-filters select,.audit-filters input{padding:4px 8px;background:#2a2d2e;border:1px solid #3c3c3c;color:#ccc;border-radius:4px;font-size:12px;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Audit Log</h1>")
	b.WriteString("<p class=\"muted\">Audit events tracking sensitive operations: policy changes, project updates, task deletions, workdir changes.</p>")
	b.WriteString("<div class=\"audit-filters\">")
	b.WriteString("<label>Event Type<select id=\"audit-type\">")
	b.WriteString("<option value=\"\">All audit events</option>")
	b.WriteString("<option value=\"audit.policy.changed\">Policy Changed</option>")
	b.WriteString("<option value=\"audit.project.updated\">Project Updated</option>")
	b.WriteString("<option value=\"audit.task.deleted\">Task Deleted</option>")
	b.WriteString("<option value=\"audit.workdir.changed\">Workdir Changed</option>")
	b.WriteString("</select></label>")
	b.WriteString("<label>Project ID<input class=\"input\" id=\"audit-project\" placeholder=\"all projects\"></label>")
	b.WriteString("<button class=\"btn\" id=\"audit-refresh\" type=\"button\">Refresh</button>")
	b.WriteString("</div>")
	b.WriteString("<div class=\"meta\"><span id=\"audit-status\">Loading...</span></div>")
	b.WriteString("<ul id=\"audit-list\"><li>Loading...</li></ul>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const listEl=document.getElementById('audit-list');")
	b.WriteString("const statusEl=document.getElementById('audit-status');")
	b.WriteString("const refreshBtn=document.getElementById('audit-refresh');")
	b.WriteString("const typeEl=document.getElementById('audit-type');")
	b.WriteString("const projectEl=document.getElementById('audit-project');")
	b.WriteString(`const escapeHtml=(v)=>String(v??'').replace(/[&<>"']/g,(c)=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)
	b.WriteString("function renderHeader(){listEl.innerHTML='';")
	b.WriteString("const header=document.createElement('li');header.className='header';")
	b.WriteString("header.innerHTML='<span>Kind</span><span>Entity</span><span>Project</span><span>Time</span><span>Details</span>';listEl.appendChild(header);}")
	b.WriteString("async function loadAudit(){statusEl.textContent='Loading...';")
	b.WriteString("listEl.innerHTML='<li>Loading...</li>';try{")
	b.WriteString("const params=new URLSearchParams();params.set('limit','100');")
	b.WriteString("const type=String(typeEl.value||'').trim();if(type){params.set('type',type);}")
	b.WriteString("const projectId=String(projectEl.value||'').trim();if(projectId){params.set('project_id',projectId);}")
	b.WriteString("const url='/api/v2/audit?'+params.toString();")
	b.WriteString("const res=await fetch(url);if(!res.ok)throw new Error();")
	b.WriteString("const data=await res.json();const events=Array.isArray(data.events)?data.events:[];")
	b.WriteString("statusEl.textContent=events.length+' of '+data.total+' audit events';renderHeader();")
	b.WriteString("if(events.length===0){const empty=document.createElement('li');")
	b.WriteString("empty.innerHTML='<span class=\"muted\">No audit events</span>';listEl.appendChild(empty);return;}")
	b.WriteString("events.forEach((ev)=>{const item=document.createElement('li');")
	b.WriteString("const kind=escapeHtml(ev.kind);const entity=escapeHtml(ev.entity_type+'/'+ev.entity_id);")
	b.WriteString("const proj=escapeHtml(ev.project_id);const created=escapeHtml(ev.created_at);")
	b.WriteString("const payload=ev.payload?escapeHtml(JSON.stringify(ev.payload)):'';")
	b.WriteString("item.innerHTML='<span>'+kind+'</span><span>'+entity+'</span><span>'+proj+'</span><span>'+created+'</span><span class=\"payload-cell\" title=\"'+payload+'\">'+payload+'</span>';listEl.appendChild(item);});")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load audit events';")
	b.WriteString("listEl.innerHTML='<li>Error loading audit events</li>';}}")
	b.WriteString("function handleEnter(e){if(e.key==='Enter'){loadAudit();}}")
	b.WriteString("refreshBtn.addEventListener('click',loadAudit);")
	b.WriteString("typeEl.addEventListener('change',loadAudit);")
	b.WriteString("projectEl.addEventListener('keydown',handleEnter);")
	b.WriteString("loadAudit();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func repoPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/repo" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Repo Panel"))
	b.WriteString("<style>")
	b.WriteString(".columns{display:grid;grid-template-columns:1fr 1fr;gap:16px;}")
	b.WriteString("@media(max-width:700px){.columns{grid-template-columns:1fr;}}")
	b.WriteString(".tree{list-style:none;padding:0;margin:0;font-size:12px;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;}")
	b.WriteString(".tree li{padding:4px 6px;border-bottom:1px solid #3c3c3c;display:flex;gap:6px;align-items:center;}")
	b.WriteString(".tree li:hover{background:#2a2d2e;}")
	b.WriteString(".tree .icon{width:14px;text-align:center;color:#858585;}")
	b.WriteString(".tree .dir{color:#0078d4;cursor:pointer;}")
	b.WriteString(".tree .file{color:#ccc;}")
	b.WriteString(".change-list{list-style:none;padding:0;margin:0;font-size:12px;}")
	b.WriteString(".change-list li{padding:6px 8px;border-bottom:1px solid #3c3c3c;display:flex;gap:10px;align-items:center;}")
	b.WriteString(".change-list .badge{display:inline-block;padding:1px 6px;border-radius:4px;font-size:10px;font-weight:600;text-transform:uppercase;}")
	b.WriteString(".badge.modified{background:#854d0e;color:#fef08a;}")
	b.WriteString(".badge.added{background:#166534;color:#bbf7d0;}")
	b.WriteString(".badge.deleted{background:#991b1b;color:#fecaca;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Repo Panel</h1>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<label class=\"field\">Project ID<input class=\"input\" id=\"repo-project\" value=\"proj_default\"></label>")
	b.WriteString("<button class=\"btn\" id=\"repo-refresh\" type=\"button\">Refresh</button>")
	b.WriteString("<span id=\"repo-status\" class=\"muted\">Loading...</span></div>")
	b.WriteString("<div class=\"columns\">")
	b.WriteString("<div><h2>File Tree</h2><ul class=\"tree\" id=\"repo-tree\"><li>Loading...</li></ul></div>")
	b.WriteString("<div><h2>Recent Changes</h2><ul class=\"change-list\" id=\"repo-changes\"><li>Loading...</li></ul></div>")
	b.WriteString("</div></div>")
	b.WriteString("<script>")
	b.WriteString("const treeEl=document.getElementById('repo-tree');")
	b.WriteString("const changesEl=document.getElementById('repo-changes');")
	b.WriteString("const statusEl=document.getElementById('repo-status');")
	b.WriteString("const refreshBtn=document.getElementById('repo-refresh');")
	b.WriteString("const projectEl=document.getElementById('repo-project');")
	b.WriteString(`const escapeHtml=(v)=>String(v??'').replace(/[&<>"']/g,(c)=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)
	b.WriteString("function getProjectID(){return String(projectEl.value||'proj_default').trim();}")
	b.WriteString("async function loadTree(){treeEl.innerHTML='<li>Loading tree...</li>';try{")
	b.WriteString("const res=await fetch('/api/v2/projects/'+encodeURIComponent(getProjectID())+'/tree');")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const entries=Array.isArray(data.entries)?data.entries:[];")
	b.WriteString("if(entries.length===0){treeEl.innerHTML='<li class=\"muted\">No entries</li>';return;}")
	b.WriteString("treeEl.innerHTML='';entries.forEach((e)=>{")
	b.WriteString("const li=document.createElement('li');")
	b.WriteString("const isDir=e.is_dir||e.kind==='dir';")
	b.WriteString("const icon=isDir?'\\uD83D\\uDCC1':'\\uD83D\\uDCC4';")
	b.WriteString("const cls=isDir?'dir':'file';")
	b.WriteString("li.innerHTML='<span class=\"icon\">'+icon+'</span><span class=\"'+cls+'\">'+escapeHtml(e.name||e.path)+'</span>';")
	b.WriteString("treeEl.appendChild(li);});")
	b.WriteString("}catch(err){treeEl.innerHTML='<li class=\"muted\">Failed to load tree</li>';}}")
	b.WriteString("async function loadChanges(){changesEl.innerHTML='<li>Loading changes...</li>';try{")
	b.WriteString("const res=await fetch('/api/v2/projects/'+encodeURIComponent(getProjectID())+'/changes');")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const changes=Array.isArray(data.changes)?data.changes:[];")
	b.WriteString("if(changes.length===0){changesEl.innerHTML='<li class=\"muted\">No recent changes</li>';return;}")
	b.WriteString("changesEl.innerHTML='';changes.forEach((c)=>{")
	b.WriteString("const li=document.createElement('li');")
	b.WriteString("const kind=(c.kind||c.status||'modified').toLowerCase();")
	b.WriteString("const badgeCls=kind==='added'?'added':kind==='deleted'?'deleted':'modified';")
	b.WriteString("li.innerHTML='<span class=\"badge '+badgeCls+'\">'+escapeHtml(kind)+'</span><span>'+escapeHtml(c.path||c.name)+'</span>';")
	b.WriteString("changesEl.appendChild(li);});")
	b.WriteString("}catch(err){changesEl.innerHTML='<li class=\"muted\">Failed to load changes</li>';}}")
	b.WriteString("async function loadAll(){statusEl.textContent='Loading...';")
	b.WriteString("await Promise.all([loadTree(),loadChanges()]);statusEl.textContent='Loaded';}")
	b.WriteString("refreshBtn.addEventListener('click',loadAll);")
	b.WriteString("projectEl.addEventListener('change',loadAll);")
	b.WriteString("loadAll();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func healthDashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/health" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Health Dashboard"))
	b.WriteString("<style>")
	b.WriteString(".status-bar{display:flex;gap:4px;height:24px;border-radius:6px;overflow:hidden;margin-bottom:14px;}")
	b.WriteString(".status-bar div{display:flex;align-items:center;justify-content:center;font-size:10px;font-weight:600;color:#fff;}")
	b.WriteString(".bar-pending{background:#858585;} .bar-progress{background:#0078d4;} .bar-done{background:#89d185;} .bar-failed{background:#f14c4c;} .bar-blocked{background:#f87171;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Health Dashboard</h1>")
	b.WriteString("<div class=\"meta\"><span id=\"health-status\" class=\"muted\">Loading...</span>")
	b.WriteString("<button class=\"btn\" id=\"health-refresh\">Refresh</button></div>")
	b.WriteString("<div class=\"grid\" id=\"health-stats\"></div>")
	b.WriteString("<h2>Task Distribution</h2>")
	b.WriteString("<div class=\"status-bar\" id=\"task-bar\"></div>")
	b.WriteString("<h2>System Info</h2>")
	b.WriteString("<table><tbody id=\"sys-info\"><tr><td>Loading...</td></tr></tbody></table>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const statsEl=document.getElementById('health-stats');")
	b.WriteString("const barEl=document.getElementById('task-bar');")
	b.WriteString("const sysEl=document.getElementById('sys-info');")
	b.WriteString("const statusEl=document.getElementById('health-status');")
	b.WriteString("const refreshBtn=document.getElementById('health-refresh');")
	b.WriteString("function makeStat(label,value,cls){return '<div class=\"stat '+(cls||'')+'\"><div class=\"label\">'+label+'</div><div class=\"value\">'+value+'</div></div>';}")
	b.WriteString("async function loadHealth(){statusEl.textContent='Loading...';try{")
	b.WriteString("const [hRes,mRes,vRes]=await Promise.all([fetch('/api/v2/health'),fetch('/api/v2/metrics'),fetch('/api/v2/version')]);")
	b.WriteString("const health=await hRes.json();const metrics=await mRes.json();const version=await vRes.json();")
	b.WriteString("const t=metrics.totals||{};const ts=metrics.tasks_by_status||{};")
	b.WriteString("const total=t.tasks||0;")
	b.WriteString("statsEl.innerHTML=makeStat('Status',health.ok?'OK':'DOWN',health.ok?'ok':'error')")
	b.WriteString("+makeStat('Tasks',t.tasks||0)+makeStat('Agents',t.agents||0)+makeStat('Runs',t.runs||0)")
	b.WriteString("+makeStat('Events',t.events||0)+makeStat('Projects',t.projects||0)")
	b.WriteString("+makeStat('Active Leases',metrics.active_leases||0)")
	b.WriteString("+makeStat('Circuit',metrics.automation_circuit_state||'-',metrics.automation_circuit_state==='closed'?'ok':'warn');")
	b.WriteString("barEl.innerHTML='';if(total>0){")
	b.WriteString("[['queued','bar-pending',ts.queued],['running','bar-progress',ts.running],['succeeded','bar-done',ts.succeeded],['failed','bar-failed',ts.failed],['review','bar-blocked',ts.needs_review]]")
	b.WriteString(".forEach(([label,cls,count])=>{if(count>0){const pct=Math.max(2,Math.round(count/total*100));")
	b.WriteString("barEl.innerHTML+='<div class=\"'+cls+'\" style=\"flex:'+pct+'\">'+label+' '+count+'</div>';}});}")
	b.WriteString("sysEl.innerHTML='';const rows=[['Version',version.version||'-'],['Schema',metrics.schema_version||'-'],['API v1',version.api&&version.api.v1?'Yes':'No'],['API v2',version.api&&version.api.v2?'Yes':'No']];")
	b.WriteString("rows.forEach(([k,v])=>{sysEl.innerHTML+='<tr><td>'+k+'</td><td>'+v+'</td></tr>';});")
	b.WriteString("statusEl.textContent='Updated '+new Date().toLocaleTimeString();")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load health data';}}")
	b.WriteString("refreshBtn.addEventListener('click',loadHealth);loadHealth();")
	b.WriteString("setInterval(loadHealth,15000);")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func repoGraphHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/graphs/repo" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Repo Graph"))
	b.WriteString("<style>")
	b.WriteString(".graph-canvas{position:relative;min-height:400px;background:#1e1e1e;border:1px solid #3c3c3c;border-radius:8px;overflow:auto;}")
	b.WriteString("svg.repo-dag{width:100%;height:100%;position:absolute;top:0;left:0;pointer-events:none;}")
	b.WriteString("svg.repo-dag line{stroke:#505050;stroke-width:1;}")
	b.WriteString(".rnode{position:absolute;background:#252526;border:1px solid #505050;border-radius:6px;padding:6px 8px;font-size:10px;z-index:1;cursor:pointer;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;}")
	b.WriteString(".rnode:hover{box-shadow:0 0 0 2px #0078d4;z-index:10;}")
	b.WriteString(".rnode.dir{border-color:#0078d4;color:#0078d4;}")
	b.WriteString(".rnode.file{border-color:#858585;color:#ccc;}")
	b.WriteString(".stats-row{display:flex;gap:16px;margin:12px 0;font-size:12px;}")
	b.WriteString(".stats-row .stat{color:#858585;}")
	b.WriteString(".stats-row .stat strong{color:#ccc;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Repo Entity Graph</h1>")
	b.WriteString("<p>File and directory relationships from project repo files</p>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<label class=\"field\">Project<input class=\"input\" id=\"rg-project\" value=\"proj_default\"></label>")
	b.WriteString("<button class=\"btn\" id=\"rg-refresh\">Refresh</button>")
	b.WriteString("<span id=\"rg-status\" class=\"muted\">Loading...</span></div>")
	b.WriteString("<div class=\"stats-row\" id=\"rg-stats\"></div>")
	b.WriteString("<div class=\"graph-canvas\" id=\"rg-canvas\"><svg class=\"repo-dag\" id=\"rg-svg\"></svg></div>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const canvasEl=document.getElementById('rg-canvas');")
	b.WriteString("const svgEl=document.getElementById('rg-svg');")
	b.WriteString("const statusEl=document.getElementById('rg-status');")
	b.WriteString("const statsEl=document.getElementById('rg-stats');")
	b.WriteString("const projectEl=document.getElementById('rg-project');")
	b.WriteString("const refreshBtn=document.getElementById('rg-refresh');")
	b.WriteString(`const escapeHtml=(v)=>String(v??'').replace(/[&<>"']/g,(c)=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)
	b.WriteString("function buildTree(files){const root={name:'/',children:{},isDir:true};")
	b.WriteString("files.forEach(f=>{const parts=(f.path||f.name||'').split('/').filter(Boolean);")
	b.WriteString("let cur=root;parts.forEach((p,i)=>{if(!cur.children[p])cur.children[p]={name:p,children:{},isDir:i<parts.length-1};")
	b.WriteString("cur=cur.children[p];});cur.lang=f.language||'';cur.size=f.size||0;});return root;}")
	b.WriteString("function flatten(node,depth,nodes,edges,parentIdx){")
	b.WriteString("const idx=nodes.length;nodes.push({name:node.name,isDir:node.isDir,depth,lang:node.lang||'',size:node.size||0});")
	b.WriteString("if(parentIdx!==null)edges.push({from:parentIdx,to:idx});")
	b.WriteString("const kids=Object.values(node.children).sort((a,b)=>(b.isDir?1:0)-(a.isDir?1:0)||a.name.localeCompare(b.name));")
	b.WriteString("kids.forEach(c=>flatten(c,depth+1,nodes,edges,idx));}")
	b.WriteString("function renderRepoGraph(files){")
	b.WriteString("const tree=buildTree(files);const nodes=[];const edges=[];flatten(tree,0,nodes,edges,null);")
	b.WriteString("const dirs=nodes.filter(n=>n.isDir).length;const fileCount=nodes.length-dirs;")
	b.WriteString("const langs=new Set(nodes.filter(n=>n.lang).map(n=>n.lang));")
	b.WriteString("statsEl.innerHTML='<span class=\"stat\">Files: <strong>'+fileCount+'</strong></span>'")
	b.WriteString("+'<span class=\"stat\">Dirs: <strong>'+dirs+'</strong></span>'")
	b.WriteString("+'<span class=\"stat\">Languages: <strong>'+langs.size+'</strong></span>';")
	b.WriteString("canvasEl.querySelectorAll('.rnode').forEach(el=>el.remove());svgEl.innerHTML='';")
	b.WriteString("const nodeW=120;const nodeH=28;const padX=20;const padY=8;const indentX=24;")
	b.WriteString("const positions=[];nodes.forEach((n,i)=>{const x=padX+n.depth*indentX;const y=padY+i*(nodeH+padY);positions.push({x,y});")
	b.WriteString("const div=document.createElement('div');div.className='rnode '+(n.isDir?'dir':'file');")
	b.WriteString("div.style.left=x+'px';div.style.top=y+'px';")
	b.WriteString("div.textContent=n.isDir?'\\uD83D\\uDCC1 '+n.name:n.name+(n.lang?' ('+n.lang+')':'');")
	b.WriteString("canvasEl.appendChild(div);});")
	b.WriteString("const totalH=nodes.length*(nodeH+padY)+padY;const totalW=Math.max(...positions.map(p=>p.x))+nodeW+padX*2;")
	b.WriteString("canvasEl.style.height=totalH+'px';canvasEl.style.width=totalW+'px';canvasEl.style.minHeight=totalH+'px';")
	b.WriteString("svgEl.setAttribute('width',totalW);svgEl.setAttribute('height',totalH);svgEl.setAttribute('viewBox','0 0 '+totalW+' '+totalH);")
	b.WriteString("const ns='http://www.w3.org/2000/svg';")
	b.WriteString("edges.forEach(e=>{const from=positions[e.from];const to=positions[e.to];if(!from||!to)return;")
	b.WriteString("const line=document.createElementNS(ns,'line');")
	b.WriteString("line.setAttribute('x1',from.x+6);line.setAttribute('y1',from.y+nodeH);")
	b.WriteString("line.setAttribute('x2',to.x+6);line.setAttribute('y2',to.y);svgEl.appendChild(line);});}")
	b.WriteString("async function loadRepoGraph(){statusEl.textContent='Loading...';try{")
	b.WriteString("const pid=String(projectEl.value||'proj_default').trim();")
	b.WriteString("const res=await fetch('/api/v2/projects/'+encodeURIComponent(pid)+'/files?limit=500');")
	b.WriteString("if(!res.ok)throw new Error('http '+res.status);")
	b.WriteString("const data=await res.json();const files=Array.isArray(data.files)?data.files:[];")
	b.WriteString("statusEl.textContent=files.length+' files';renderRepoGraph(files);")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load repo files';}}")
	b.WriteString("refreshBtn.addEventListener('click',loadRepoGraph);")
	b.WriteString("projectEl.addEventListener('change',loadRepoGraph);")
	b.WriteString("loadRepoGraph();")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func diffViewerHandler(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/diffs/")
	if trimmed == "" || trimmed == "/" || strings.Contains(trimmed, "/") {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString(subPageHead("Diff Viewer"))
	b.WriteString("<style>")
	b.WriteString(".diff-container{font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;font-size:12px;line-height:1.6;overflow-x:auto;}")
	b.WriteString(".diff-line{display:flex;white-space:pre;}")
	b.WriteString(".diff-line .ln{min-width:40px;text-align:right;padding:0 8px;color:#858585;user-select:none;cursor:pointer;}")
	b.WriteString(".diff-line .ln:hover{background:#3c3c3c;color:#0078d4;}")
	b.WriteString(".diff-line .content{flex:1;padding:0 8px;}")
	b.WriteString(".diff-add{background:rgba(22,163,74,0.15);}")
	b.WriteString(".diff-add .content{color:#89d185;}")
	b.WriteString(".diff-del{background:rgba(220,38,38,0.15);}")
	b.WriteString(".diff-del .content{color:#f14c4c;}")
	b.WriteString(".diff-hunk{background:rgba(0,120,212,0.1);color:#0078d4;font-style:italic;}")
	b.WriteString(".diff-ctx .content{color:#858585;}")
	b.WriteString(".split{display:grid;grid-template-columns:1fr 1fr;gap:0;}")
	b.WriteString(".split .pane{overflow-x:auto;border:1px solid #3c3c3c;}")
	b.WriteString(".split .pane:first-child{border-right:none;}")
	b.WriteString(".file-list{list-style:none;padding:0;margin:0 0 12px;display:flex;flex-wrap:wrap;gap:6px;font-size:11px;}")
	b.WriteString(".file-list li{background:#1e1e1e;border:1px solid #3c3c3c;border-radius:6px;padding:3px 8px;cursor:pointer;}")
	b.WriteString(".file-list li:hover{background:#3c3c3c;}")
	b.WriteString(".comment-row{background:#252526;border-left:3px solid #0078d4;padding:6px 12px;margin:2px 0;font-size:12px;}")
	b.WriteString(".comment-row .cm-author{color:#0078d4;font-weight:600;margin-right:6px;} .comment-row .cm-body{color:#ccc;} .comment-row .cm-time{color:#858585;font-size:10px;margin-left:8px;}")
	b.WriteString(".comment-form{display:flex;gap:6px;padding:4px 12px;background:#252526;border-left:3px solid #505050;margin:2px 0;}")
	b.WriteString(".comment-form input{flex:1;background:#3c3c3c;border:1px solid #505050;border-radius:4px;color:#ccc;padding:4px 8px;font-size:12px;}")
	b.WriteString(".comment-form button{background:#505050;border:1px solid #606060;color:#ccc;border-radius:4px;padding:4px 10px;cursor:pointer;font-size:11px;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Diff Viewer</h1>")
	b.WriteString("<p>Artifact: <span class=\"muted\" id=\"diff-artifact-id\"></span></p>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("<button class=\"btn active\" id=\"btn-unified\" type=\"button\">Unified</button>")
	b.WriteString("<button class=\"btn\" id=\"btn-split\" type=\"button\">Split</button>")
	b.WriteString("<span id=\"diff-status\" class=\"muted\">Loading...</span></div>")
	b.WriteString("<ul class=\"file-list\" id=\"diff-files\"></ul>")
	b.WriteString("<div class=\"diff-container\" id=\"diff-content\">Loading...</div>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const artifactId='" + html.EscapeString(trimmed) + "';")
	b.WriteString("document.getElementById('diff-artifact-id').textContent=artifactId;")
	b.WriteString("const contentEl=document.getElementById('diff-content');")
	b.WriteString("const statusEl=document.getElementById('diff-status');")
	b.WriteString("const filesEl=document.getElementById('diff-files');")
	b.WriteString("const btnUnified=document.getElementById('btn-unified');")
	b.WriteString("const btnSplit=document.getElementById('btn-split');")
	b.WriteString(`const escapeHtml=(v)=>String(v??'').replace(/[&<>"']/g,(c)=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));`)
	b.WriteString("let diffText='';let viewMode='unified';")
	b.WriteString("function parseDiffFiles(text){const files=[];const re=/^diff --git a\\/(.+?) b\\//gm;let m;while((m=re.exec(text))!==null)files.push(m[1]);")
	b.WriteString("if(files.length===0){const hdr=/^--- a\\/(.+)$/gm;while((m=hdr.exec(text))!==null)files.push(m[1]);}return files;}")
	b.WriteString("function renderUnified(text){const lines=text.split('\\n');let html='';let ln=0;")
	b.WriteString("lines.forEach((line)=>{ln++;let cls='diff-ctx';")
	b.WriteString("if(line.startsWith('+')){cls='diff-add';}else if(line.startsWith('-')){cls='diff-del';}else if(line.startsWith('@@')){cls='diff-hunk';}")
	b.WriteString("html+='<div class=\"diff-line '+cls+'\"><span class=\"ln\">'+ln+'</span><span class=\"content\">'+escapeHtml(line)+'</span></div>';});")
	b.WriteString("return html;}")
	b.WriteString("function renderSplit(text){const lines=text.split('\\n');let left='';let right='';let lln=0;let rln=0;")
	b.WriteString("lines.forEach((line)=>{")
	b.WriteString("if(line.startsWith('-')){lln++;left+='<div class=\"diff-line diff-del\"><span class=\"ln\">'+lln+'</span><span class=\"content\">'+escapeHtml(line.slice(1))+'</span></div>';")
	b.WriteString("right+='<div class=\"diff-line\"><span class=\"ln\"></span><span class=\"content\"></span></div>';}")
	b.WriteString("else if(line.startsWith('+')){rln++;right+='<div class=\"diff-line diff-add\"><span class=\"ln\">'+rln+'</span><span class=\"content\">'+escapeHtml(line.slice(1))+'</span></div>';")
	b.WriteString("left+='<div class=\"diff-line\"><span class=\"ln\"></span><span class=\"content\"></span></div>';}")
	b.WriteString("else if(line.startsWith('@@')){const h='<div class=\"diff-line diff-hunk\"><span class=\"ln\"></span><span class=\"content\">'+escapeHtml(line)+'</span></div>';left+=h;right+=h;}")
	b.WriteString("else{lln++;rln++;const row='<div class=\"diff-line diff-ctx\"><span class=\"ln\">'+(lln)+'</span><span class=\"content\">'+escapeHtml(line)+'</span></div>';left+=row;right+=row;}});")
	b.WriteString("return '<div class=\"split\"><div class=\"pane\">'+left+'</div><div class=\"pane\">'+right+'</div></div>';}")
	b.WriteString("function renderDiff(){if(!diffText){contentEl.innerHTML='<span class=\"muted\">No diff content</span>';return;}")
	b.WriteString("contentEl.innerHTML=viewMode==='split'?renderSplit(diffText):renderUnified(diffText);}")
	b.WriteString("async function loadDiff(){statusEl.textContent='Loading...';contentEl.innerHTML='Loading...';try{")
	b.WriteString("const res=await fetch('/api/v2/artifacts/'+encodeURIComponent(artifactId)+'/content');")
	b.WriteString("if(res.ok){diffText=await res.text();}else{")
	b.WriteString("const res2=await fetch('/api/v2/artifacts/'+encodeURIComponent(artifactId));")
	b.WriteString("if(!res2.ok)throw new Error('http '+res2.status);const data=await res2.json();diffText=data.content||data.body||JSON.stringify(data,null,2);}")
	b.WriteString("const files=parseDiffFiles(diffText);filesEl.innerHTML='';")
	b.WriteString("files.forEach((f)=>{const li=document.createElement('li');li.textContent=f;filesEl.appendChild(li);});")
	b.WriteString("statusEl.textContent=files.length+' file(s)';renderDiff();")
	b.WriteString("}catch(err){statusEl.textContent='Failed to load diff';contentEl.innerHTML='<span class=\"muted\">Could not load artifact</span>';}}")
	b.WriteString("btnUnified.addEventListener('click',()=>{viewMode='unified';btnUnified.classList.add('active');btnSplit.classList.remove('active');renderDiff();});")
	b.WriteString("btnSplit.addEventListener('click',()=>{viewMode='split';btnSplit.classList.add('active');btnUnified.classList.remove('active');renderDiff();});")
	// Line comment functionality
	b.WriteString("let comments=[];")
	b.WriteString("async function loadComments(){try{const res=await fetch('/api/v2/artifacts/'+encodeURIComponent(artifactId)+'/comments');if(res.ok){const data=await res.json();comments=data.comments||[];}}catch(e){}}")
	b.WriteString("async function postComment(lineNumber,body){try{const res=await fetch('/api/v2/artifacts/'+encodeURIComponent(artifactId)+'/comments',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({line_number:lineNumber,body:body,author:'reviewer'})});if(res.ok){await loadComments();renderDiff();}}catch(e){}}")
	b.WriteString("function renderCommentsForLine(container,ln){const lc=comments.filter(c=>c.line_number===ln);lc.forEach(c=>{const div=document.createElement('div');div.className='comment-row';div.innerHTML='<span class=\"cm-author\">'+escapeHtml(c.author||'anon')+'</span><span class=\"cm-body\">'+escapeHtml(c.body)+'</span><span class=\"cm-time\">'+escapeHtml(c.created_at||'')+'</span>';container.appendChild(div);});}")
	b.WriteString("contentEl.addEventListener('click',(e)=>{const lnEl=e.target.closest('.ln');if(!lnEl)return;const ln=parseInt(lnEl.textContent,10);if(isNaN(ln))return;")
	b.WriteString("const existing=lnEl.parentElement.nextElementSibling;if(existing&&existing.classList.contains('comment-form')){existing.remove();return;}")
	b.WriteString("const form=document.createElement('div');form.className='comment-form';form.innerHTML='<input placeholder=\"Add comment...\" autofocus><button>Post</button>';")
	b.WriteString("lnEl.parentElement.after(form);const inp=form.querySelector('input');inp.focus();form.querySelector('button').addEventListener('click',()=>{const val=inp.value.trim();if(val)postComment(ln,val);});inp.addEventListener('keydown',(ev)=>{if(ev.key==='Enter'){const val=inp.value.trim();if(val)postComment(ln,val);}if(ev.key==='Escape')form.remove();});});")
	// Enhance renderDiff to include inline comments
	b.WriteString("const origRenderDiff=renderDiff;")
	b.WriteString("renderDiff=function(){origRenderDiff();document.querySelectorAll('.diff-line').forEach(el=>{const lnEl=el.querySelector('.ln');if(!lnEl)return;const ln=parseInt(lnEl.textContent,10);if(isNaN(ln))return;renderCommentsForLine(el.parentElement||contentEl,ln);});};")
	b.WriteString("loadComments().then(()=>{loadDiff();});")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	fmt.Fprint(w, b.String())
}

func runsPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/runs/")
	if trimmed == "" || trimmed == "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var b strings.Builder
		b.WriteString(subPageHead("Runs"))
		b.WriteString("<style>")
		b.WriteString("label{font-size:12px;color:#b0b0b0;margin-bottom:6px;display:block;}")
		b.WriteString(".row{display:flex;flex-wrap:wrap;gap:10px;align-items:center;}")
		b.WriteString("input{flex:1;min-width:220px;}")
		b.WriteString(".link{font-size:12px;color:#0078d4;text-decoration:none;border:1px dashed #505050;padding:8px 12px;border-radius:4px;}")
		b.WriteString(".link.disabled{color:#858585;border-color:#3c3c3c;pointer-events:none;}")
		b.WriteString("</style>")
		b.WriteString("<div class=\"card\">")
		b.WriteString("<h1>Runs</h1>")
		b.WriteString("<p>Enter a run ID to open the run viewer.</p>")
		b.WriteString("<form id=\"run-lookup\" autocomplete=\"off\">")
		b.WriteString("<label for=\"run-id\">Run ID</label>")
		b.WriteString("<div class=\"row\">")
		b.WriteString("<input class=\"input\" id=\"run-id\" name=\"runId\" placeholder=\"run_abc123\" required>")
		b.WriteString("<button class=\"btn\" type=\"submit\">Open run</button>")
		b.WriteString("<a class=\"link disabled\" id=\"run-link\" href=\"/runs\">Open /runs/{runId}</a>")
		b.WriteString("</div>")
		b.WriteString("</form>")
		b.WriteString("<p class=\"muted\">Route: /runs/{runId} | Data: GET /api/v2/runs/{runId}</p>")
		b.WriteString("</div>")
		b.WriteString("<script>")
		b.WriteString("const form=document.getElementById('run-lookup');")
		b.WriteString("const input=document.getElementById('run-id');")
		b.WriteString("const link=document.getElementById('run-link');")
		b.WriteString("function updateLink(){const value=input.value.trim();")
		b.WriteString("if(value){const href='/runs/'+encodeURIComponent(value);")
		b.WriteString("link.href=href;link.textContent='Open '+href;link.classList.remove('disabled');}")
		b.WriteString("else{link.href='/runs';link.textContent='Open /runs/{runId}';link.classList.add('disabled');}}")
		b.WriteString("form.addEventListener('submit',(e)=>{e.preventDefault();")
		b.WriteString("const value=input.value.trim();if(!value){input.focus();return;}")
		b.WriteString("window.location.href='/runs/'+encodeURIComponent(value);});")
		b.WriteString("input.addEventListener('input',updateLink);updateLink();")
		b.WriteString("</script>")
		b.WriteString(subPageFoot())
		fmt.Fprint(w, b.String())
		return
	}
	if strings.Contains(trimmed, "/") {
		http.NotFound(w, r)
		return
	}
	page := runViewerPage(trimmed)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, page)
}

func contextPacksPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/context-packs/")
	if trimmed == "" || trimmed == "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, contextPacksCreatePage())
		return
	}
	if strings.Contains(trimmed, "/") {
		http.NotFound(w, r)
		return
	}
	page := contextPackDetailPage(trimmed)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, page)
}

func contextPacksCreatePage() string {
	var b strings.Builder
	b.WriteString(subPageHead("Context Packs"))
	b.WriteString("<style>")
	b.WriteString("label{font-size:12px;color:#858585;margin-bottom:6px;display:block;}")
	b.WriteString(".row{display:flex;flex-wrap:wrap;gap:10px;align-items:center;}")
	b.WriteString(".meta{margin-top:8px;color:#858585;font-size:12px;}")
	b.WriteString(".status{margin-top:10px;color:#ccc;font-size:12px;}")
	b.WriteString("</style>")
	b.WriteString("<div class=\"card\">")
	b.WriteString("<h1>Context Packs</h1>")
	b.WriteString("<p>Create a minimal context pack for a task and view the JSON response.</p>")
	b.WriteString("<form id=\"context-pack-form\" autocomplete=\"off\">")
	b.WriteString("<label for=\"cp-project\">Project ID</label>")
	b.WriteString("<div class=\"row\">")
	b.WriteString("<input id=\"cp-project\" name=\"project\" value=\"proj_default\">")
	b.WriteString("</div>")
	b.WriteString("<label for=\"task-id\">Task ID</label>")
	b.WriteString("<div class=\"row\">")
	b.WriteString("<input id=\"task-id\" name=\"taskId\" placeholder=\"task_abc123\" required>")
	b.WriteString("</div>")
	b.WriteString("<label for=\"query\">Query (optional)</label>")
	b.WriteString("<div class=\"row\">")
	b.WriteString("<input id=\"query\" name=\"query\" placeholder=\"focus on migrations\">")
	b.WriteString("<button class=\"btn\" type=\"submit\">Create pack</button>")
	b.WriteString("</div>")
	b.WriteString("</form>")
	b.WriteString("<div class=\"meta\" id=\"cp-endpoint\">POST /api/v2/projects/proj_default/context-packs</div>")
	b.WriteString("<div class=\"status\" id=\"context-pack-status\">Status: idle</div>")
	b.WriteString("<pre id=\"context-pack-output\">{}</pre>")
	b.WriteString("</div>")
	b.WriteString("<script>")
	b.WriteString("const form=document.getElementById('context-pack-form');")
	b.WriteString("const taskInput=document.getElementById('task-id');")
	b.WriteString("const queryInput=document.getElementById('query');")
	b.WriteString("const projectInput=document.getElementById('cp-project');")
	b.WriteString("const statusEl=document.getElementById('context-pack-status');")
	b.WriteString("const outputEl=document.getElementById('context-pack-output');")
	b.WriteString("const endpointEl=document.getElementById('cp-endpoint');")
	b.WriteString("form.addEventListener('submit',async(e)=>{e.preventDefault();")
	b.WriteString("const taskId=taskInput.value.trim();")
	b.WriteString("if(!taskId){taskInput.focus();return;}")
	b.WriteString("const pid=String(projectInput.value||'proj_default').trim();")
	b.WriteString("endpointEl.textContent='POST /api/v2/projects/'+pid+'/context-packs';")
	b.WriteString("const payload={task_id:taskId};")
	b.WriteString("const query=queryInput.value.trim();")
	b.WriteString("if(query){payload.query=query;}")
	b.WriteString("statusEl.textContent='Status: sending...';outputEl.textContent='';")
	b.WriteString("try{const resp=await fetch('/api/v2/projects/'+encodeURIComponent(pid)+'/context-packs',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});")
	b.WriteString("const text=await resp.text();")
	b.WriteString("let data=text;try{data=JSON.parse(text);}catch(e){}")
	b.WriteString("outputEl.textContent=typeof data==='string'?data:JSON.stringify(data,null,2);")
	b.WriteString("statusEl.textContent='Status: '+resp.status;}")
	b.WriteString("catch(err){statusEl.textContent='Status: error';outputEl.textContent=String(err);}});")
	b.WriteString("</script>")
	b.WriteString(subPageFoot())
	return b.String()
}

const htmlTemplate = `
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Cocopilot - Kanban</title>
        <script defer src="/static/alpine-collapse.min.js"></script>
        <script defer src="/static/alpine.min.js"></script>
        <style>
            * {
                box-sizing: border-box;
                margin: 0;
                padding: 0;
            }

            :root {
                --vscode-bg: #1e1e1e;
                --vscode-sidebar: #252526;
                --vscode-input-bg: #3c3c3c;
                --vscode-border: #3c3c3c;
                --vscode-text: #cccccc;
                --vscode-text-muted: #858585;
                --vscode-accent: #0078d4;
                --vscode-accent-hover: #1c8ae8;
                --vscode-success: #89d185;
                --vscode-warning: #cca700;
                --vscode-info: #3794ff;
            }

            body {
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                font-size: 13px;
                background: var(--vscode-bg);
                color: var(--vscode-text);
                height: 100vh;
                display: flex;
                flex-direction: column;
                overflow: hidden;
            }

            /* Header */
            .header {
                background: var(--vscode-sidebar);
                border-bottom: 1px solid var(--vscode-border);
                padding: 12px 20px;
                display: flex;
                align-items: center;
                gap: 12px;
                flex-shrink: 0;
            }

            .header-icon { color: var(--vscode-accent); }
            .header-title { font-size: 14px; font-weight: 600; }

            .header-nav {
                display: flex;
                align-items: center;
                gap: 10px;
                font-size: 12px;
            }

            .header-nav a {
                color: var(--vscode-text-muted);
                text-decoration: none;
                padding: 4px 6px;
                border-radius: 4px;
                transition: all 0.15s;
            }

            .header-nav a:hover {
                color: var(--vscode-text);
                background: var(--vscode-input-bg);
            }

            .header-actions {
                margin-left: auto;
                display: flex;
                align-items: center;
                gap: 12px;
            }

            .header-btn {
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                color: var(--vscode-text);
                padding: 6px 12px;
                font-size: 12px;
                cursor: pointer;
                border-radius: 4px;
                display: flex;
                align-items: center;
                gap: 6px;
                transition: all 0.15s;
            }

            .header-btn:hover {
                background: var(--vscode-accent);
                border-color: var(--vscode-accent);
            }

            .workdir-input {
                display: flex;
                align-items: center;
                gap: 8px;
            }

            .workdir-input label {
                font-size: 12px;
                color: var(--vscode-text-muted);
            }

            .workdir-input input {
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 4px;
                color: var(--vscode-text);
                padding: 6px 10px;
                font-size: 12px;
                width: 250px;
                outline: none;
            }

            .workdir-input input:focus {
                border-color: var(--vscode-accent);
            }

            /* Kanban Board */
            .kanban-board {
                flex: 1;
                display: grid;
                grid-template-columns: repeat(3, 1fr);
                gap: 16px;
                padding: 20px;
                overflow: hidden;
                min-height: 0;
            }

            .kanban-column {
                background: var(--vscode-sidebar);
                border-radius: 8px;
                display: flex;
                flex-direction: column;
                overflow: hidden;
                border: 1px solid var(--vscode-border);
                transition: border-color 0.2s;
                min-height: 0;
            }

            .kanban-column.drag-over {
                border-color: var(--vscode-accent);
                box-shadow: 0 0 0 2px rgba(0, 120, 212, 0.3);
            }

            .column-header {
                padding: 12px 16px;
                border-bottom: 1px solid var(--vscode-border);
                display: flex;
                align-items: center;
                gap: 8px;
                flex-shrink: 0;
            }

            .column-title { font-weight: 600; font-size: 13px; }

            .column-count {
                background: var(--vscode-input-bg);
                padding: 2px 8px;
                border-radius: 10px;
                font-size: 11px;
                color: var(--vscode-text-muted);
                transition: all 0.3s;
            }

            .column-icon { width: 16px; height: 16px; }
            .col-todo .column-icon { color: var(--vscode-warning); }
            .col-progress .column-icon { color: var(--vscode-info); }
            .col-done .column-icon { color: var(--vscode-success); }

            .column-body {
                flex: 1;
                overflow-y: auto;
                padding: 12px;
                display: flex;
                flex-direction: column;
                gap: 10px;
                min-height: 0;
            }

            .column-body::-webkit-scrollbar { width: 8px; }
            .column-body::-webkit-scrollbar-track { background: transparent; }
            .column-body::-webkit-scrollbar-thumb { background: var(--vscode-border); border-radius: 4px; }

            /* Task Cards */
            .task-card {
                background: var(--vscode-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                cursor: grab;
                transition: all 0.2s ease;
                overflow: hidden;
                flex-shrink: 0;
            }

            .task-card:hover {
                border-color: var(--vscode-accent);
                box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
            }

            .task-card.dragging {
                opacity: 0.5;
                cursor: grabbing;
                transform: scale(0.98);
            }

            .card-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
                padding: 10px 12px;
                cursor: pointer;
                user-select: none;
            }

            .card-header:hover { background: rgba(255,255,255,0.03); }

            .card-left {
                display: flex;
                align-items: center;
                gap: 8px;
            }

            .card-id {
                font-weight: 600;
                color: var(--vscode-accent);
                font-size: 12px;
            }

            .card-preview {
                font-size: 11px;
                color: var(--vscode-text-muted);
                max-width: 150px;
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
            }

            .card-right {
                display: flex;
                align-items: center;
                gap: 8px;
            }

            .card-time {
                font-size: 10px;
                color: var(--vscode-text-muted);
            }

            .delete-btn {
                background: transparent;
                border: none;
                color: var(--vscode-text-muted);
                cursor: pointer;
                padding: 2px;
                border-radius: 4px;
                display: flex;
                align-items: center;
                justify-content: center;
                transition: all 0.15s;
            }

            .delete-btn:hover {
                background: rgba(255, 0, 0, 0.2);
                color: #ff6b6b;
            }

            .expand-icon {
                transition: transform 0.2s;
                color: var(--vscode-text-muted);
            }

            .expand-icon.rotated { transform: rotate(180deg); }

            .card-body {
                padding: 0 12px 12px 12px;
                overflow: hidden;
            }

            .card-instructions {
                font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
                font-size: 11px;
                line-height: 1.5;
                white-space: pre-wrap;
                word-wrap: break-word;
                color: var(--vscode-text);
                margin: 0;
                padding: 10px;
                background: rgba(0,0,0,0.2);
                border-radius: 4px;
                max-height: 300px;
                overflow-y: auto;
            }

            .card-output {
                margin-top: 10px;
                padding: 10px;
                background: rgba(137, 209, 133, 0.1);
                border-radius: 4px;
                border-left: 3px solid var(--vscode-success);
            }

            .card-output-label {
                font-size: 10px;
                color: var(--vscode-success);
                text-transform: uppercase;
                letter-spacing: 0.5px;
                margin-bottom: 6px;
            }

            .card-output-text {
                font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
                font-size: 11px;
                line-height: 1.5;
                white-space: pre-wrap;
                word-wrap: break-word;
                color: var(--vscode-text);
                max-height: 200px;
                overflow-y: auto;
            }

            /* Empty state */
            .empty-column {
                display: flex;
                align-items: center;
                justify-content: center;
                height: 80px;
                color: var(--vscode-text-muted);
                font-style: italic;
                font-size: 12px;
            }

            /* Modal */
            .modal-overlay {
                position: fixed;
                top: 0; left: 0; right: 0; bottom: 0;
                background: rgba(0, 0, 0, 0.6);
                z-index: 1000;
                display: flex;
                align-items: center;
                justify-content: center;
            }

            .modal {
                background: var(--vscode-sidebar);
                border: 1px solid var(--vscode-border);
                border-radius: 8px;
                max-width: 500px;
                width: 90%;
                box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
            }

            .modal-header {
                display: flex;
                align-items: center;
                justify-content: space-between;
                padding: 16px;
                border-bottom: 1px solid var(--vscode-border);
            }

            .modal-title { font-size: 14px; font-weight: 600; }

            .modal-close {
                background: transparent;
                border: none;
                color: var(--vscode-text-muted);
                cursor: pointer;
                padding: 4px;
                border-radius: 4px;
            }

            .modal-close:hover {
                background: var(--vscode-input-bg);
                color: var(--vscode-text);
            }

            .modal-body { padding: 16px; }

            .modal-body textarea {
                width: 100%;
                min-height: 120px;
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                color: var(--vscode-text);
                font-family: inherit;
                font-size: 13px;
                padding: 12px;
                resize: vertical;
                outline: none;
            }

            .modal-body textarea:focus { border-color: var(--vscode-accent); }

            .modal-footer {
                padding: 12px 16px;
                border-top: 1px solid var(--vscode-border);
                display: flex;
                justify-content: flex-end;
                gap: 8px;
            }

            .btn {
                padding: 8px 16px;
                border-radius: 4px;
                font-size: 12px;
                cursor: pointer;
                border: 1px solid var(--vscode-border);
                transition: all 0.15s;
            }

            .btn-primary {
                background: var(--vscode-accent);
                border-color: var(--vscode-accent);
                color: #fff;
            }

            .btn-primary:hover { background: var(--vscode-accent-hover); }
            .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

            .btn-secondary {
                background: var(--vscode-input-bg);
                color: var(--vscode-text);
            }

            .btn-secondary:hover { background: #4a4a4a; }

            /* Code block for instructions */
            .code-block {
                background: var(--vscode-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                overflow: hidden;
            }

            .code-header {
                display: flex;
                align-items: center;
                justify-content: space-between;
                padding: 8px 12px;
                background: var(--vscode-input-bg);
                border-bottom: 1px solid var(--vscode-border);
            }

            .code-label {
                font-size: 11px;
                color: var(--vscode-text-muted);
                text-transform: uppercase;
                letter-spacing: 0.5px;
            }

            .copy-btn {
                background: transparent;
                border: 1px solid var(--vscode-border);
                color: var(--vscode-text);
                padding: 4px 10px;
                font-size: 11px;
                cursor: pointer;
                border-radius: 4px;
                display: flex;
                align-items: center;
                gap: 4px;
                transition: all 0.15s;
            }

            .copy-btn:hover {
                background: var(--vscode-accent);
                border-color: var(--vscode-accent);
            }

            .copy-btn.copied {
                background: var(--vscode-success);
                border-color: var(--vscode-success);
                color: #1e1e1e;
            }

            .code-content {
                padding: 16px;
            }

            .code-content pre {
                font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
                font-size: 13px;
                line-height: 1.6;
                color: var(--vscode-text);
                margin: 0;
                white-space: pre-wrap;
                word-wrap: break-word;
            }

            .modal.modal-wide {
                max-width: 600px;
            }

            /* Animations */
            .fade-enter-active, .fade-leave-active {
                transition: opacity 0.2s, transform 0.2s;
            }
            .fade-enter-from, .fade-leave-to {
                opacity: 0;
                transform: translateY(-10px);
            }

            [x-cloak] { display: none !important; }

            /* Task hierarchy - indentation for child tasks */
            .task-card[data-depth="1"] { margin-left: 20px; border-left: 3px solid var(--vscode-accent); }
            .task-card[data-depth="2"] { margin-left: 40px; border-left: 3px solid var(--vscode-info); }
            .task-card[data-depth="3"] { margin-left: 60px; border-left: 3px solid var(--vscode-warning); }

            .parent-badge {
                display: inline-flex;
                align-items: center;
                gap: 4px;
                font-size: 10px;
                color: var(--vscode-text-muted);
                background: var(--vscode-input-bg);
                padding: 2px 6px;
                border-radius: 4px;
            }

            .parent-badge svg {
                width: 10px;
                height: 10px;
            }

            /* Parent task select in modal */
            .modal-body select {
                width: 100%;
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                color: var(--vscode-text);
                font-family: inherit;
                font-size: 13px;
                padding: 10px 12px;
                outline: none;
                cursor: pointer;
            }

            .modal-body select:focus { border-color: var(--vscode-accent); }

            .modal-body label {
                display: block;
                font-size: 12px;
                color: var(--vscode-text-muted);
                margin-bottom: 6px;
            }

            .form-group {
                margin-bottom: 12px;
            }

            /* Agent Status Styles */
            .status-badge {
                border-radius: 10px;
                font-size: 10px;
                font-weight: 500;
                padding: 2px 6px;
                text-transform: uppercase;
            }

            .status-badge.status-online {
                background: var(--vscode-charts-green);
                color: white;
            }

            .status-badge.status-offline {
                background: var(--vscode-charts-red);
                color: white;
            }

            .status-badge.status-busy {
                background: var(--vscode-charts-orange);
                color: white;
            }

            .status-badge.status-idle {
                background: var(--vscode-charts-blue);
                color: white;
            }

            .agent-item:last-child {
                border-bottom: none;
            }

            .agent-status [x-cloak] { display: none !important; }

            /* Footer */
            .footer {
                background: var(--vscode-sidebar);
                border-top: 1px solid var(--vscode-border);
                padding: 10px 20px;
                text-align: center;
                font-size: 12px;
                color: var(--vscode-text-muted);
                flex-shrink: 0;
            }

            .footer a {
                color: var(--vscode-accent);
                text-decoration: none;
            }

            .footer a:hover {
                text-decoration: underline;
            }
        </style>
    </head>
    <body x-data="kanbanApp()" x-init="init()">
        <!-- Create Task Modal -->
        <div class="modal-overlay"
             x-show="showModal"
             x-cloak
             x-transition:enter="fade-enter-active"
             x-transition:leave="fade-leave-active"
             @click.self="showModal = false"
             @keydown.escape.window="showModal = false">
            <div class="modal" @click.stop>
                <div class="modal-header">
                    <span class="modal-title">Create New Task</span>
                    <button class="modal-close" @click="showModal = false">
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                            <path fill-rule="evenodd" clip-rule="evenodd" d="M8 8.707l3.646 3.647.708-.707L8.707 8l3.647-3.646-.707-.708L8 7.293 4.354 3.646l-.707.708L7.293 8l-3.646 3.646.707.708L8 8.707z"/>
                        </svg>
                    </button>
                </div>
                <form @submit.prevent="createTask">
                    <div class="modal-body">
                        <div class="form-group">
                            <label>Parent Task (optional)</label>
                            <select x-model="newTaskParentId">
                                <option value="">No parent (standalone task)</option>
                                <template x-for="task in tasks" :key="task.id">
                                    <option :value="task.id" x-text="'#' + task.id + ': ' + task.instructions.substring(0, 50) + (task.instructions.length > 50 ? '...' : '')"></option>
                                </template>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Instructions</label>
                            <textarea
                                x-model="newTaskInstructions"
                                placeholder="Enter task instructions..."
                                required
                                x-ref="taskInput"></textarea>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" @click="showModal = false">Cancel</button>
                        <button type="submit" class="btn btn-primary" :disabled="!newTaskInstructions.trim()">Create Task</button>
                    </div>
                </form>
            </div>
        </div>

        <!-- Instructions Modal -->
        <div class="modal-overlay"
             x-show="showInstructionsModal"
             x-cloak
             x-transition:enter="fade-enter-active"
             x-transition:leave="fade-leave-active"
             @click.self="showInstructionsModal = false"
             @keydown.escape.window="showInstructionsModal = false">
            <div class="modal modal-wide" @click.stop>
                <div class="modal-header">
                    <span class="modal-title">Initial Copilot Instructions</span>
                    <button class="modal-close" @click="showInstructionsModal = false">
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                            <path fill-rule="evenodd" clip-rule="evenodd" d="M8 8.707l3.646 3.647.708-.707L8.707 8l3.647-3.646-.707-.708L8 7.293 4.354 3.646l-.707.708L7.293 8l-3.646 3.646.707.708L8 8.707z"/>
                        </svg>
                    </button>
                </div>
                <div class="modal-body">
                    <div class="code-block">
                        <div class="code-header">
                            <span class="code-label">Paste this to Copilot</span>
                            <button class="copy-btn" :class="{ 'copied': copiedInstructions }" @click="copyInstructions">
                                <svg x-show="!copiedInstructions" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                    <path fill-rule="evenodd" clip-rule="evenodd" d="M4 4l1-1h5.414L14 6.586V14l-1 1H5l-1-1V4zm9 3l-3-3H5v10h8V7zM3 1L2 2v10l1 1V2h6.414l-1-1H3z"/>
                                </svg>
                                <svg x-show="copiedInstructions" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                    <path fill-rule="evenodd" clip-rule="evenodd" d="M14.431 3.323l-8.47 10-.79-.036-3.35-4.77.818-.574 2.978 4.24 8.051-9.506.763.646z"/>
                                </svg>
                                <span x-text="copiedInstructions ? 'Copied!' : 'Copy'"></span>
                            </button>
                        </div>
                        <div class="code-content">
                            <pre x-ref="instructionsText">
                            See http://127.0.0.1:8080/instructions use curl to get the instructions and proceed with it. Do not stop when there are no new tasks!
                            </pre>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="header">
            <svg class="header-icon" width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
                <path d="M8.5 1a6.5 6.5 0 1 1 0 13 6.5 6.5 0 0 1 0-13zm0 1a5.5 5.5 0 1 0 0 11 5.5 5.5 0 0 0 0-11zm-2 3.5a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm0 2a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm0 2a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5z"/>
            </svg>
            <span class="header-title">Cocopilot</span>
            <nav class="header-nav" aria-label="Primary">
                <a href="/agents">Agents</a>
                <a href="/runs">Runs</a>
                <a href="/memory">Memory</a>
                <a href="/context-packs">Context Packs</a>
                <a href="/graphs/tasks">Task DAG</a>
                <a href="/graphs/repo">Repo Graph</a>
                <a href="/repo">Repo</a>
                <a href="/audit">Audit</a>
                <a href="/health">Health</a>
            </nav>
            <div class="header-actions">
                <div class="workdir-input">
                    <label for="project">Project:</label>
                    <select id="project" x-model="currentProject" @change="onProjectChange" style="background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); border-radius: 4px; color: var(--vscode-text); padding: 6px 10px; font-size: 12px; min-width: 150px; cursor: pointer;">
                        <template x-for="project in projects" :key="project.id">
                            <option :value="project.id" x-text="project.name"></option>
                        </template>
                    </select>
                </div>
                <div class="workdir-input">
                    <label>Agents:</label>
                    <div class="agent-status" x-data="{ showAgentDetails: false }" @click="showAgentDetails = !showAgentDetails" style="position: relative; cursor: pointer;">
                        <div style="display: flex; align-items: center; gap: 6px; background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); border-radius: 4px; padding: 6px 10px; font-size: 12px; min-width: 120px;">
                            <span x-text="agents.filter(a => a.status === 'ONLINE').length"></span>
                            <span style="color: var(--vscode-text-muted);">online</span>
                            <span style="color: var(--vscode-text-muted); font-size: 10px;">▼</span>
                        </div>
                        <div x-show="showAgentDetails" x-cloak style="position: absolute; top: 100%; right: 0; background: var(--vscode-dropdown-bg); border: 1px solid var(--vscode-border); border-radius: 4px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); z-index: 1000; min-width: 200px; max-height: 300px; overflow-y: auto;">
                            <template x-for="agent in agents" :key="agent.id">
                                <div class="agent-item" style="padding: 8px 12px; border-bottom: 1px solid var(--vscode-border); display: flex; justify-content: space-between; align-items: center;">
                                    <span x-text="agent.name" style="font-size: 12px; font-weight: 500;"></span>
                                    <span :class="'status-badge status-' + agent.status.toLowerCase()" x-text="agent.status" style="font-size: 10px; padding: 2px 6px; border-radius: 10px; font-weight: 500;"></span>
                                </div>
                            </template>
                            <div x-show="agents.length === 0" style="padding: 12px; text-align: center; color: var(--vscode-text-muted); font-size: 12px;">
                                No agents registered
                            </div>
                        </div>
                    </div>
                </div>
                <div class="workdir-input">
                    <label for="workdir">Workdir:</label>
                    <input type="text" id="workdir" x-model="workdir" @change="updateWorkdir" placeholder="/path/to/workdir">
                </div>
                <button class="header-btn" @click="showInstructionsModal = true">
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M14.5 2H9l-.35.15-.65.64-.65-.64L7 2H1.5l-.5.5v10l.5.5h5.29l.86.85h1.71l.85-.85h5.29l.5-.5v-10l-.5-.5zm-7 10.32l-.18-.17L7 12H2V3h4.79l.74.74-.03 8.58zM14 12H9l-.35.15-.14.13V3.7l.7-.7H14v9z"/>
                    </svg>
                    Initial Instructions
                </button>
                <button class="header-btn" @click="openModal">
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M14 7v1H8v6H7V8H1V7h6V1h1v6h6z"/>
                    </svg>
                    New Task
                </button>
            </div>
        </div>

        <div class="kanban-board">
            <!-- To Do Column -->
            <div class="kanban-column col-todo"
                 :class="{ 'drag-over': dragOverColumn === 'NOT_PICKED' }"
                 @dragover.prevent="dragOverColumn = 'NOT_PICKED'"
                 @dragleave="dragOverColumn = null"
                 @drop="dropTask('NOT_PICKED')">
                <div class="column-header">
                    <svg class="column-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M8 3.5a.5.5 0 0 0-1 0V9a.5.5 0 0 0 .252.434l3.5 2a.5.5 0 0 0 .496-.868L8 8.71V3.5z"/>
                        <path d="M8 16A8 8 0 1 0 8 0a8 8 0 0 0 0 16zm7-8A7 7 0 1 1 1 8a7 7 0 0 1 14 0z"/>
                    </svg>
                    <span class="column-title">To Do</span>
                    <span class="column-count" x-text="todoTasks.length"></span>
                </div>
                <div class="column-body">
                    <template x-for="task in todoTasks" :key="task.id">
                        <div class="task-card"
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag">
                            <div class="card-header" @click="toggleExpand(task.id)">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-preview" x-text="task.instructions.substring(0, 40) + (task.instructions.length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="formatTime(task.created_at)" :title="task.created_at"></span>
                                    <button class="delete-btn" @click.stop="deleteTask(task.id)" title="Delete task">
                                        <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                                            <path fill-rule="evenodd" d="M5.75 3V1.5h4.5V3h3.75v1.5H2V3h3.75zm-.5 5.25a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zm4 0a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zM3.5 5v9.5h9V5h-9z"/>
                                        </svg>
                                    </button>
                                    <svg class="expand-icon" :class="{ 'rotated': expandedTasks.includes(task.id) }" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                        <path fill-rule="evenodd" d="M1.646 4.646a.5.5 0 0 1 .708 0L8 10.293l5.646-5.647a.5.5 0 0 1 .708.708l-6 6a.5.5 0 0 1-.708 0l-6-6a.5.5 0 0 1 0-.708z"/>
                                    </svg>
                                </div>
                            </div>
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="todoTasks.length === 0">No tasks</div>
                </div>
            </div>

            <!-- In Progress Column -->
            <div class="kanban-column col-progress"
                 :class="{ 'drag-over': dragOverColumn === 'IN_PROGRESS' }"
                 @dragover.prevent="dragOverColumn = 'IN_PROGRESS'"
                 @dragleave="dragOverColumn = null"
                 @drop="dropTask('IN_PROGRESS')">
                <div class="column-header">
                    <svg class="column-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M8 0a8 8 0 1 0 8 8A8 8 0 0 0 8 0zm0 14a6 6 0 1 1 6-6 6 6 0 0 1-6 6z" opacity=".3"/>
                        <path d="M8 0v2a6 6 0 0 1 6 6h2a8 8 0 0 0-8-8z"/>
                    </svg>
                    <span class="column-title">In Progress</span>
                    <span class="column-count" x-text="progressTasks.length"></span>
                </div>
                <div class="column-body">
                    <template x-for="task in progressTasks" :key="task.id">
                        <div class="task-card"
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag">
                            <div class="card-header" @click="toggleExpand(task.id)">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-preview" x-text="task.instructions.substring(0, 40) + (task.instructions.length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="formatTime(task.created_at)" :title="task.created_at"></span>
                                    <button class="delete-btn" @click.stop="deleteTask(task.id)" title="Delete task">
                                        <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                                            <path fill-rule="evenodd" d="M5.75 3V1.5h4.5V3h3.75v1.5H2V3h3.75zm-.5 5.25a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zm4 0a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zM3.5 5v9.5h9V5h-9z"/>
                                        </svg>
                                    </button>
                                    <svg class="expand-icon" :class="{ 'rotated': expandedTasks.includes(task.id) }" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                        <path fill-rule="evenodd" d="M1.646 4.646a.5.5 0 0 1 .708 0L8 10.293l5.646-5.647a.5.5 0 0 1 .708.708l-6 6a.5.5 0 0 1-.708 0l-6-6a.5.5 0 0 1 0-.708z"/>
                                    </svg>
                                </div>
                            </div>
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="progressTasks.length === 0">No tasks</div>
                </div>
            </div>

            <!-- Done Column -->
            <div class="kanban-column col-done"
                 :class="{ 'drag-over': dragOverColumn === 'COMPLETE' }"
                 @dragover.prevent="dragOverColumn = 'COMPLETE'"
                 @dragleave="dragOverColumn = null"
                 @drop="dropTask('COMPLETE')">
                <div class="column-header">
                    <svg class="column-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zm-3.97-3.03a.75.75 0 0 0-1.08.022L7.477 9.417 5.384 7.323a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-.01-1.05z"/>
                    </svg>
                    <span class="column-title">Done</span>
                    <span class="column-count" x-text="doneTasks.length"></span>
                </div>
                <div class="column-body">
                    <template x-for="task in doneTasks" :key="task.id">
                        <div class="task-card"
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag">
                            <div class="card-header" @click="toggleExpand(task.id)">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-preview" x-text="task.instructions.substring(0, 40) + (task.instructions.length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="formatTime(task.created_at)" :title="task.created_at"></span>
                                    <button class="delete-btn" @click.stop="deleteTask(task.id)" title="Delete task">
                                        <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                                            <path fill-rule="evenodd" d="M5.75 3V1.5h4.5V3h3.75v1.5H2V3h3.75zm-.5 5.25a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zm4 0a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zM3.5 5v9.5h9V5h-9z"/>
                                        </svg>
                                    </button>
                                    <svg class="expand-icon" :class="{ 'rotated': expandedTasks.includes(task.id) }" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                        <path fill-rule="evenodd" d="M1.646 4.646a.5.5 0 0 1 .708 0L8 10.293l5.646-5.647a.5.5 0 0 1 .708.708l-6 6a.5.5 0 0 1-.708 0l-6-6a.5.5 0 0 1 0-.708z"/>
                                    </svg>
                                </div>
                            </div>
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                                <div class="card-output" x-show="task.output">
                                    <div class="card-output-label">Output</div>
                                    <pre class="card-output-text" x-text="task.output"></pre>
                                </div>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="doneTasks.length === 0">No tasks</div>
                </div>
            </div>
        </div>

        <footer class="footer">
            Created by <a href="https://dganev.com" target="_blank" rel="noopener">syl</a>
        </footer>

        <script>
            function kanbanApp() {
                return {
                    tasks: [],
                    expandedTasks: [],
                    showModal: false,
                    showInstructionsModal: false,
                    copiedInstructions: false,
                    newTaskInstructions: '',
                    newTaskParentId: '',
                    draggingId: null,
                    draggingTask: null,
                    dragOverColumn: null,
                    eventSource: null,
                    workdir: '/tmp',
                    projects: [],
                    currentProject: 'proj_default',
                    agents: [],

                    // Compute task depth based on parent chain
                    getTaskDepth(task) {
                        let depth = 0;
                        let currentParentId = task.parent_task_id;
                        while (currentParentId && depth < 3) {
                            const parent = this.tasks.find(t => t.id === currentParentId);
                            if (!parent) break;
                            depth++;
                            currentParentId = parent.parent_task_id;
                        }
                        return depth;
                    },

                    // Organize tasks hierarchically within each status column
                    getOrganizedTasks(status) {
                        const statusTasks = this.tasks.filter(t => t.status === status);
                        const rootTasks = statusTasks.filter(t => !t.parent_task_id);
                        const childrenMap = {};

                        statusTasks.forEach(t => {
                            if (t.parent_task_id) {
                                if (!childrenMap[t.parent_task_id]) childrenMap[t.parent_task_id] = [];
                                childrenMap[t.parent_task_id].push(t);
                            }
                        });

                        const result = [];
                        const addWithChildren = (task, depth = 0) => {
                            result.push({ ...task, depth });
                            (childrenMap[task.id] || []).forEach(child => addWithChildren(child, depth + 1));
                        };

                        rootTasks.forEach(t => addWithChildren(t));

                        // Also add orphaned children (whose parent is in a different status)
                        statusTasks.forEach(t => {
                            if (t.parent_task_id && !result.find(r => r.id === t.id)) {
                                result.push({ ...t, depth: this.getTaskDepth(t) });
                            }
                        });

                        return result;
                    },

                    get todoTasks() {
                        return this.getOrganizedTasks('NOT_PICKED');
                    },

                    get progressTasks() {
                        return this.getOrganizedTasks('IN_PROGRESS');
                    },

                    get doneTasks() {
                        return this.getOrganizedTasks('COMPLETE');
                    },

                    init() {
                        this.connectSSE();
                        this.fetchWorkdir();
                        this.fetchProjects();
                        this.fetchAgents();
                        
                        // Refresh agents status every 30 seconds
                        setInterval(() => {
                            this.fetchAgents();
                        }, 30000);
                    },

                    async fetchProjects() {
                        try {
                            const response = await fetch('/api/v2/projects');
                            const data = await response.json();
                            this.projects = data.projects || [];
                            if (this.projects.length > 0 && this.projects.find(p => p.id === 'proj_default')) {
                                this.currentProject = 'proj_default';
                            } else if (this.projects.length > 0) {
                                this.currentProject = this.projects[0].id;
                            }
                        } catch (error) {
                            console.error('Failed to fetch projects:', error);
                        }
                    },

                    async fetchAgents() {
                        try {
                            const response = await fetch('/api/v2/agents');
                            const data = await response.json();
                            this.agents = data.agents || [];
                        } catch (error) {
                            console.error('Failed to fetch agents:', error);
                            this.agents = [];
                        }
                    },

                    onProjectChange() {
                        // Reconnect SSE with new project filter
                        this.connectSSE();
                    },

                    formatTime(iso) {
                        if (!iso) return '';
                        const d = new Date(iso);
                        const now = new Date();
                        const diff = now - d;
                        if (diff < 60000) return 'just now';
                        if (diff < 3600000) return Math.floor(diff / 60000) + 'm ago';
                        if (diff < 86400000) return Math.floor(diff / 3600000) + 'h ago';
                        if (diff < 604800000) return Math.floor(diff / 86400000) + 'd ago';
                        return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
                    },

                    async fetchWorkdir() {
                        try {
                            const response = await fetch('/api/workdir');
                            const data = await response.json();
                            this.workdir = data.workdir;
                        } catch (error) {
                            console.error('Failed to fetch workdir:', error);
                        }
                    },

                    async updateWorkdir() {
                        const formData = new FormData();
                        formData.append('workdir', this.workdir);
                        await fetch('/set-workdir', {
                            method: 'POST',
                            body: formData
                        });
                    },

                    connectSSE() {
                        if (this.eventSource) {
                            this.eventSource.close();
                        }
                        const url = this.currentProject ? '/events?project_id=' + encodeURIComponent(this.currentProject) : '/events';
                        this.eventSource = new EventSource(url);

                        // SSE sends named "tasks" events
                        this.eventSource.addEventListener('tasks', (event) => {
                            try {
                                this.tasks = JSON.parse(event.data);
                            } catch (e) {
                                console.error('Failed to parse SSE task data:', e);
                            }
                        });

                        this.eventSource.onerror = () => {
                            this.eventSource.close();
                            // Reconnect after 3 seconds
                            setTimeout(() => this.connectSSE(), 3000);
                        };
                    },

                    openModal() {
                        this.showModal = true;
                        this.$nextTick(() => this.$refs.taskInput.focus());
                    },

                    copyInstructions() {
                        const text = this.$refs.instructionsText.textContent;
                        navigator.clipboard.writeText(text).then(() => {
                            this.copiedInstructions = true;
                            setTimeout(() => {
                                this.copiedInstructions = false;
                            }, 2000);
                        });
                    },

                    async createTask() {
                        if (!this.newTaskInstructions.trim()) return;

                        const formData = new FormData();
                        formData.append('instructions', this.newTaskInstructions);
                        formData.append('project_id', this.currentProject);
                        if (this.newTaskParentId) {
                            formData.append('parent_task_id', this.newTaskParentId);
                        }

                        try {
                            const response = await fetch('/create', {
                                method: 'POST',
                                body: formData
                            });
                            if (!response.ok) {
                                const text = await response.text();
                                alert('Failed to create task: ' + (text || response.statusText));
                                return;
                            }
                        } catch (error) {
                            alert('Failed to create task: ' + error.message);
                            return;
                        }

                        this.newTaskInstructions = '';
                        this.newTaskParentId = '';
                        this.showModal = false;
                    },

                    toggleExpand(taskId) {
                        const index = this.expandedTasks.indexOf(taskId);
                        if (index === -1) {
                            this.expandedTasks.push(taskId);
                        } else {
                            this.expandedTasks.splice(index, 1);
                        }
                    },

                    startDrag(task) {
                        this.draggingId = task.id;
                        this.draggingTask = task;
                    },

                    endDrag() {
                        this.draggingId = null;
                        this.draggingTask = null;
                        this.dragOverColumn = null;
                    },

                    async dropTask(newStatus) {
                        if (!this.draggingTask || this.draggingTask.status === newStatus) {
                            this.endDrag();
                            return;
                        }

                        const formData = new FormData();
                        formData.append('task_id', this.draggingTask.id);
                        formData.append('status', newStatus);

                        const originalStatus = this.draggingTask.status;
                        // Optimistic update
                        this.draggingTask.status = newStatus;

                        try {
                            const response = await fetch('/update-status', {
                                method: 'POST',
                                body: formData
                            });
                            if (!response.ok) {
                                this.draggingTask.status = originalStatus;
                            }
                        } catch (error) {
                            this.draggingTask.status = originalStatus;
                        }

                        this.endDrag();
                    },

                    async deleteTask(taskId) {
                        if (!confirm('Are you sure you want to delete this task?')) {
                            return;
                        }

                        const formData = new FormData();
                        formData.append('task_id', taskId);

                        try {
                            const response = await fetch('/delete', {
                                method: 'POST',
                                body: formData
                            });
                            if (!response.ok) {
                                const text = await response.text();
                                alert('Failed to delete task: ' + (text || response.statusText));
                            }
                        } catch (error) {
                            alert('Failed to delete task: ' + error.message);
                        }
                    }
                };
            }
        </script>
    </body>
</html>
`
