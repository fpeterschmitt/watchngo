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

func FindRecursive(path string, matches []string) ([]string, error) {
	wr := walkRec{
		matches: matches,
	}

	err := filepath.Walk(path, wr.walkRecursive)
	if err != nil {
		return matches, fmt.Errorf("walk: %s: %v", path, err)
	}

	return wr.matches, nil
}

func FindGlob(pattern string, matches []string) ([]string, error) {
	nMatches, err := filepath.Glob(pattern)

	if err != nil {
		return matches, fmt.Errorf("glob: %s: %v", pattern, err)
	}

	matches = append(matches, nMatches...)

	return matches, nil
}
