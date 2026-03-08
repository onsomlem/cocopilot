// File scanning — thin wrapper over internal/scanner.
package server

import "github.com/onsomlem/cocopilot/internal/scanner"

func detectLanguage(filename string) string {
	return scanner.DetectLanguage(filename)
}

func computeContentHash(fpath string, maxBytes int64) (string, error) {
	return scanner.ComputeContentHash(fpath, maxBytes)
}

func parseGitignore(workdir string) (func(path string, isDir bool) bool, error) {
	return scanner.ParseGitignore(workdir)
}

func ScanProjectFiles(projectID, workdir string) ([]RepoFile, error) {
	return scanner.ScanProjectFiles(projectID, workdir)
}
