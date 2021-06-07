package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type WalkRec struct {
	Matches []string
	Exclude []string
}

func NewWalkRec() WalkRec {
	return WalkRec{
		Matches: make([]string, 0),
		Exclude: make([]string, 0),
	}
}

func (w *WalkRec) walkRecursive(file string, info os.FileInfo, err error) error {
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			log.Printf("skipped: %v", err)
			w.Exclude = append(w.Exclude, file)
			return nil
		}
		return fmt.Errorf("walk: %s: %w", file, err)
	}

	if info.IsDir() {
		w.Matches = append(w.Matches, file)
	}

	return nil
}

// FindRecursive looks for directories in the given path, that MUST be
// a directory.
func FindRecursive(path string, wr WalkRec) (WalkRec, error) {
	var err error

	if err = filepath.Walk(path, wr.walkRecursive); err != nil {
		return wr, fmt.Errorf("walk: %s: %w", path, err)
	}

	return wr, nil
}

// FindGlob uses the given pattern that must be
// compatible with path/filepath.Glob()
func FindGlob(pattern string, matches []string) ([]string, error) {
	nMatches, err := filepath.Glob(pattern)

	if err != nil {
		return matches, fmt.Errorf("glob: %s: %w", pattern, err)
	}

	matches = append(matches, nMatches...)

	return matches, nil
}
