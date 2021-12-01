package pkg

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

func (w *WalkRec) walkRecursive(file string, info os.FileInfo, walkErr error) error {
	if walkErr != nil {
		if strings.Contains(walkErr.Error(), "permission denied") {
			log.Printf("skipped: %v", walkErr)
			w.Exclude = append(w.Exclude, file)
			return nil
		}
		return fmt.Errorf("walk: %s: %w", file, walkErr)
	}

	if info.IsDir() {
		w.Matches = append(w.Matches, file)
	}

	return nil
}

// FindRecursive looks for directories in the given path, that MUST be
// a directory.
func FindRecursive(path string) (matches []string, excludes []string, err error) {
	wr := WalkRec{}

	if err = filepath.Walk(path, wr.walkRecursive); err != nil {
		return nil, nil, fmt.Errorf("walk: %s: %w", path, err)
	}

	matches = wr.Matches
	excludes = wr.Exclude

	return
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
