// Package scanner provides project file scanning, language detection, and gitignore parsing.
package scanner

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

// DefaultFileScanMaxSize is the default max file size for content hashing (1 MB).
const DefaultFileScanMaxSize int64 = 1048576

// DefaultIgnoreDirs contains directories that are always skipped during scanning.
var DefaultIgnoreDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".DS_Store":    true,
	"__pycache__":  true,
	".venv":        true,
	"vendor":       true,
}

// ExtensionLanguageMap maps file extensions to language names.
var ExtensionLanguageMap = map[string]string{
	".go":         "go",
	".js":         "javascript",
	".ts":         "typescript",
	".py":         "python",
	".rs":         "rust",
	".java":       "java",
	".rb":         "ruby",
	".c":          "c",
	".cpp":        "cpp",
	".h":          "c",
	".hpp":        "cpp",
	".cs":         "cs",
	".swift":      "swift",
	".kt":         "kotlin",
	".md":         "markdown",
	".json":       "json",
	".yaml":       "yaml",
	".yml":        "yaml",
	".toml":       "toml",
	".xml":        "xml",
	".html":       "html",
	".css":        "css",
	".scss":       "scss",
	".sql":        "sql",
	".sh":         "shell",
	".bash":       "shell",
	".zsh":        "shell",
	".fish":       "shell",
	".ps1":        "powershell",
	".dockerfile": "dockerfile",
	".proto":      "protobuf",
	".graphql":    "graphql",
	".txt":        "text",
	".cfg":        "config",
	".ini":        "config",
	".env":        "config",
}

// SpecialFilenameMap maps exact filenames (case-sensitive) to languages.
var SpecialFilenameMap = map[string]string{
	"Dockerfile":  "dockerfile",
	"Makefile":    "makefile",
	"Jenkinsfile": "groovy",
}

// DetectLanguage returns the language name for the given filename, or "" if unknown.
func DetectLanguage(filename string) string {
	base := filepath.Base(filename)
	if lang, ok := SpecialFilenameMap[base]; ok {
		return lang
	}
	ext := strings.ToLower(filepath.Ext(base))
	if lang, ok := ExtensionLanguageMap[ext]; ok {
		return lang
	}
	return ""
}

// ComputeContentHash reads up to maxBytes from the file and returns the SHA-256 hex digest.
func ComputeContentHash(fpath string, maxBytes int64) (string, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, io.LimitReader(f, maxBytes)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// FileScanMaxSize returns the configured max size for content hashing.
func FileScanMaxSize() int64 {
	if v := os.Getenv("COCO_FILE_SCAN_MAX_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return DefaultFileScanMaxSize
}

// gitignorePattern represents a single parsed .gitignore pattern.
type gitignorePattern struct {
	pattern  string
	negation bool
	dirOnly  bool
}

// ParseGitignore reads the .gitignore at workdir root and returns a matcher function.
// The matcher returns true if the given relative path should be ignored.
func ParseGitignore(workdir string) (func(path string, isDir bool) bool, error) {
	gitignorePath := filepath.Join(workdir, ".gitignore")
	f, err := os.Open(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return func(string, bool) bool { return false }, nil
		}
		return nil, err
	}
	defer f.Close()

	var patterns []gitignorePattern
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var p gitignorePattern
		if strings.HasPrefix(line, "!") {
			p.negation = true
			line = line[1:]
		}
		if strings.HasSuffix(line, "/") {
			p.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		p.pattern = line
		patterns = append(patterns, p)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	return func(relPath string, isDir bool) bool {
		ignored := false
		for _, p := range patterns {
			if p.dirOnly && !isDir {
				continue
			}
			if MatchGitignore(p.pattern, relPath) {
				if p.negation {
					ignored = false
				} else {
					ignored = true
				}
			}
		}
		return ignored
	}, nil
}

// MatchGitignore checks whether relPath matches a gitignore glob pattern.
// If the pattern contains no slash, it matches against the basename.
// Otherwise it matches against the full relative path.
func MatchGitignore(pattern, relPath string) bool {
	if !strings.Contains(pattern, "/") {
		base := filepath.Base(relPath)
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		for _, part := range parts {
			if matched, _ := filepath.Match(pattern, part); matched {
				return true
			}
		}
		return false
	}
	pattern = strings.TrimPrefix(pattern, "/")
	if matched, _ := filepath.Match(pattern, filepath.ToSlash(relPath)); matched {
		return true
	}
	return false
}

// ScanProjectFiles walks workdir and returns a RepoFile for each file found.
func ScanProjectFiles(projectID, workdir string) ([]models.RepoFile, error) {
	maxSize := FileScanMaxSize()

	isIgnored, err := ParseGitignore(workdir)
	if err != nil {
		return nil, err
	}

	var files []models.RepoFile

	err = filepath.Walk(workdir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(workdir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if rel == "." {
			return nil
		}

		name := info.Name()

		if info.IsDir() {
			if DefaultIgnoreDirs[name] {
				return filepath.SkipDir
			}
			if isIgnored(rel, true) {
				return filepath.SkipDir
			}
			return nil
		}

		if DefaultIgnoreDirs[name] {
			return nil
		}

		if isIgnored(rel, false) {
			return nil
		}

		now := models.NowISO()
		size := info.Size()
		modTime := info.ModTime().UTC().Format(time.RFC3339)
		lang := DetectLanguage(name)

		rf := models.RepoFile{
			ID:           "rf_" + uuid.New().String(),
			ProjectID:    projectID,
			Path:         rel,
			SizeBytes:    &size,
			LastModified: &modTime,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if lang != "" {
			rf.Language = &lang
		}

		if size <= maxSize {
			hash, hashErr := ComputeContentHash(path, 64*1024)
			if hashErr == nil {
				rf.ContentHash = &hash
			}
		}

		files = append(files, rf)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}
