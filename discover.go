package main

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// findRepos walks root and returns the parent directory of every .git
// directory, including nested repositories. maxDepth limits how deep a
// repository may sit relative to root (0 = root itself only, -1 = unlimited).
func findRepos(root string, maxDepth int) ([]string, error) {
	var repos []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == ".git" && path != root {
			repos = append(repos, filepath.Dir(path))
			return filepath.SkipDir
		}
		if maxDepth >= 0 && path != root {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			depth := strings.Count(rel, string(filepath.Separator)) + 1
			if depth > maxDepth {
				return filepath.SkipDir
			}
		}
		return nil
	})
	return repos, err
}
