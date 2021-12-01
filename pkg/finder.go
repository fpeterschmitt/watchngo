//go:generate mockgen -source=finder.go -destination=mock_finder_test.go -package=pkg_test Finder

package pkg

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type FinderResults struct {
	Locations []string
}

type Finder interface {
	Find() (*FinderResults, error)
}

type walkRec struct {
	Matches []string
	Exclude []string
}

func (w *walkRec) walkRecursive(file string, info os.FileInfo, walkErr error) error {
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
	wr := walkRec{}

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

type LocalFinder struct {
	Match string
}

func (l LocalFinder) Find() (*FinderResults, error) {
	matchstat, err := os.Stat(l.Match)
	var fr FinderResults

	if err == nil && matchstat.IsDir() {
		fr.Locations, _, err = FindRecursive(l.Match)
		if err != nil {
			return nil, fmt.Errorf("find: %w", err)
		}
	} else if err == nil && !matchstat.IsDir() {
		fr.Locations = append(fr.Locations, l.Match)
	} else if err != nil {
		fr.Locations, err = FindGlob(l.Match, fr.Locations)

		if err != nil {
			return nil, fmt.Errorf("glob: %w", err)
		} else if len(fr.Locations) == 0 {
			return nil, fmt.Errorf("empty glob: %s", l.Match)
		}
	} else {
		return nil, fmt.Errorf("bad conf: %s: %w", l.Match, err)
	}

	return &fr, nil
}
