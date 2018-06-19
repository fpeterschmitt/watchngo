package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

type walkRec struct {
	matches []string
}

func (w *walkRec) walkRecursive(file string, info os.FileInfo, err error) error {
	if err != nil {
		return fmt.Errorf("walk: %s: %v", file, err)
	}

	if info.IsDir() {
		w.matches = append(w.matches, file)
	}

	return nil
}

// FindRecursive looks for directories in the given path, that MUST be
// a directory.
func FindRecursive(path string, matches []string) ([]string, error) {
	var err error

	wr := walkRec{
		matches: matches,
	}

	if err = filepath.Walk(path, wr.walkRecursive); err != nil {
		return matches, fmt.Errorf("walk: %s: %v", path, err)
	}

	return wr.matches, nil
}

// FindGlob uses the given pattern that must be
// compatible with path/filepath.Glob()
func FindGlob(pattern string, matches []string) ([]string, error) {
	nMatches, err := filepath.Glob(pattern)

	if err != nil {
		return matches, fmt.Errorf("glob: %s: %v", pattern, err)
	}

	matches = append(matches, nMatches...)

	return matches, nil
}
