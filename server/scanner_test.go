package server

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"main.go", "go"},
		{"app.js", "javascript"},
		{"index.ts", "typescript"},
		{"script.py", "python"},
		{"lib.rs", "rust"},
		{"Main.java", "java"},
		{"app.rb", "ruby"},
		{"util.c", "c"},
		{"util.cpp", "cpp"},
		{"util.h", "c"},
		{"util.hpp", "cpp"},
		{"Program.cs", "cs"},
		{"app.swift", "swift"},
		{"app.kt", "kotlin"},
		{"README.md", "markdown"},
		{"data.json", "json"},
		{"config.yaml", "yaml"},
		{"config.yml", "yaml"},
		{"config.toml", "toml"},
		{"doc.xml", "xml"},
		{"index.html", "html"},
		{"style.css", "css"},
		{"style.scss", "scss"},
		{"query.sql", "sql"},
		{"run.sh", "shell"},
		{"run.bash", "shell"},
		{"run.zsh", "shell"},
		{"run.fish", "shell"},
		{"script.ps1", "powershell"},
		{"schema.proto", "protobuf"},
		{"schema.graphql", "graphql"},
		{"notes.txt", "text"},
		{"settings.cfg", "config"},
		{"settings.ini", "config"},
		{".env", "config"},
		// Special filenames
		{"Dockerfile", "dockerfile"},
		{"Makefile", "makefile"},
		{"Jenkinsfile", "groovy"},
		// Dockerfile extension
		{"app.dockerfile", "dockerfile"},
		// Unknown
		{"archive.tar.gz", ""},
		{"random.xyz", ""},
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			got := detectLanguage(tc.filename)
			if got != tc.want {
				t.Errorf("detectLanguage(%q) = %q, want %q", tc.filename, got, tc.want)
			}
		})
	}
}

func TestComputeContentHash(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello world\n")
	fpath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := computeContentHash(fpath, 64*1024)
	if err != nil {
		t.Fatal(err)
	}

	// Compute expected hash.
	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])

	if hash != expected {
		t.Errorf("hash = %q, want %q", hash, expected)
	}
}

func TestComputeContentHashLargeFile(t *testing.T) {
	dir := t.TempDir()
	// Create a file larger than 64KB.
	data := make([]byte, 128*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	fpath := filepath.Join(dir, "large.bin")
	if err := os.WriteFile(fpath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Hash should only cover first 64KB.
	hash, err := computeContentHash(fpath, 64*1024)
	if err != nil {
		t.Fatal(err)
	}

	h := sha256.Sum256(data[:64*1024])
	expected := hex.EncodeToString(h[:])
	if hash != expected {
		t.Errorf("hash = %q, want %q", hash, expected)
	}
}

func TestComputeContentHashMissingFile(t *testing.T) {
	_, err := computeContentHash("/nonexistent/path/file.txt", 64*1024)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseGitignore(t *testing.T) {
	dir := t.TempDir()
	gitignore := `# comment
*.log
build/
!important.log
`
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		t.Fatal(err)
	}

	isIgnored, err := parseGitignore(dir)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path  string
		isDir bool
		want  bool
	}{
		{"app.log", false, true},
		{"sub/debug.log", false, true},
		{"important.log", false, false}, // negated
		{"build", true, true},           // directory-only pattern
		{"build", false, false},         // file named build: not matched (dir-only)
		{"main.go", false, false},
		{"src/main.go", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := isIgnored(tc.path, tc.isDir)
			if got != tc.want {
				t.Errorf("isIgnored(%q, %v) = %v, want %v", tc.path, tc.isDir, got, tc.want)
			}
		})
	}
}

func TestParseGitignoreNoFile(t *testing.T) {
	dir := t.TempDir()
	isIgnored, err := parseGitignore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if isIgnored("anything.go", false) {
		t.Error("expected no ignores when .gitignore is missing")
	}
}

func TestScanProjectFiles(t *testing.T) {
	dir := t.TempDir()

	// Create directory structure.
	dirs := []string{
		"src",
		"src/sub",
		".git",
		"node_modules",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create files.
	writeFile := func(rel, content string) {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("main.go", "package main\n")
	writeFile("src/app.ts", "console.log('hi');\n")
	writeFile("src/sub/util.py", "print('hi')\n")
	writeFile("README.md", "# Hello\n")
	writeFile("Makefile", "all:\n\techo hi\n")
	writeFile(".git/config", "[core]\n") // should be skipped
	writeFile("node_modules/pkg.js", "module.exports = {};\n") // should be skipped

	files, err := ScanProjectFiles("proj_test123", dir)
	if err != nil {
		t.Fatal(err)
	}

	// Collect paths.
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	sort.Strings(paths)

	expected := []string{
		"Makefile",
		"README.md",
		"main.go",
		"src/app.ts",
		"src/sub/util.py",
	}

	if len(paths) != len(expected) {
		t.Fatalf("got %d files %v, want %d files %v", len(paths), paths, len(expected), expected)
	}
	for i, p := range paths {
		if p != expected[i] {
			t.Errorf("path[%d] = %q, want %q", i, p, expected[i])
		}
	}

	// Verify fields on one file.
	for _, f := range files {
		if f.Path == "main.go" {
			if f.ProjectID != "proj_test123" {
				t.Errorf("ProjectID = %q, want %q", f.ProjectID, "proj_test123")
			}
			if f.ID == "" || f.ID[:3] != "rf_" {
				t.Errorf("ID = %q, want rf_ prefix", f.ID)
			}
			if f.Language == nil || *f.Language != "go" {
				t.Errorf("Language = %v, want 'go'", f.Language)
			}
			if f.SizeBytes == nil || *f.SizeBytes != int64(len("package main\n")) {
				t.Errorf("SizeBytes = %v, want %d", f.SizeBytes, len("package main\n"))
			}
			if f.ContentHash == nil || *f.ContentHash == "" {
				t.Error("ContentHash should be set")
			}
			if f.LastModified == nil || *f.LastModified == "" {
				t.Error("LastModified should be set")
			}
			if f.CreatedAt == "" {
				t.Error("CreatedAt should be set")
			}
			if f.UpdatedAt == "" {
				t.Error("UpdatedAt should be set")
			}
		}
		if f.Path == "Makefile" {
			if f.Language == nil || *f.Language != "makefile" {
				t.Errorf("Makefile Language = %v, want 'makefile'", f.Language)
			}
		}
	}
}

func TestScanProjectFilesGitignore(t *testing.T) {
	dir := t.TempDir()

	writeFile := func(rel, content string) {
		parent := filepath.Dir(filepath.Join(dir, rel))
		os.MkdirAll(parent, 0755)
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile(".gitignore", "*.log\nbuild/\n")
	writeFile("main.go", "package main\n")
	writeFile("debug.log", "some log\n")
	writeFile("build/output.bin", "binary\n")

	files, err := ScanProjectFiles("proj_gi", dir)
	if err != nil {
		t.Fatal(err)
	}

	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	sort.Strings(paths)

	// .gitignore itself should appear, debug.log and build/ should not.
	expected := []string{".gitignore", "main.go"}
	if len(paths) != len(expected) {
		t.Fatalf("got %d files %v, want %d files %v", len(paths), paths, len(expected), expected)
	}
	for i, p := range paths {
		if p != expected[i] {
			t.Errorf("path[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

func TestScanMaxSize(t *testing.T) {
	dir := t.TempDir()

	// Set max size to 100 bytes.
	t.Setenv("COCO_FILE_SCAN_MAX_SIZE", "100")

	// Small file: should have hash.
	smallContent := []byte("small file\n")
	if err := os.WriteFile(filepath.Join(dir, "small.txt"), smallContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Large file: over 100 bytes, should have nil hash.
	largeContent := make([]byte, 200)
	for i := range largeContent {
		largeContent[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(dir, "large.txt"), largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	files, err := ScanProjectFiles("proj_maxsize", dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		switch f.Path {
		case "small.txt":
			if f.ContentHash == nil {
				t.Error("small.txt should have content_hash")
			}
			if f.SizeBytes == nil || *f.SizeBytes != int64(len(smallContent)) {
				t.Errorf("small.txt SizeBytes = %v, want %d", f.SizeBytes, len(smallContent))
			}
		case "large.txt":
			if f.ContentHash != nil {
				t.Errorf("large.txt should have nil content_hash, got %q", *f.ContentHash)
			}
			if f.SizeBytes == nil || *f.SizeBytes != int64(len(largeContent)) {
				t.Errorf("large.txt SizeBytes = %v, want %d", f.SizeBytes, len(largeContent))
			}
		default:
			t.Errorf("unexpected file: %s", f.Path)
		}
	}
}
