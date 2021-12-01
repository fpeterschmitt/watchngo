//go:generate mockgen -source=finder.go -destination=mock_finder_test.go -package=pkg_test Finder

package pkg

import (
	"fmt"
	"os"
)

type FinderResults struct {
	Files []string
}

type Finder interface {
	Find() (*FinderResults, error)
}

type LocalFinder struct {
	Match string
}

func (l LocalFinder) Find() (*FinderResults, error) {
	matchstat, err := os.Stat(l.Match)
	var fr FinderResults

	if err == nil && matchstat.IsDir() {
		fr.Files, _, err = FindRecursive(l.Match)
		if err != nil {
			return nil, fmt.Errorf("find: %w", err)
		}
	} else if err == nil && !matchstat.IsDir() {
		fr.Files = append(fr.Files, l.Match)
	} else if err != nil {
		fr.Files, err = FindGlob(l.Match, fr.Files)

		if err != nil {
			return nil, fmt.Errorf("glob: %w", err)
		} else if len(fr.Files) == 0 {
			return nil, fmt.Errorf("empty glob: %s", l.Match)
		}
	} else {
		return nil, fmt.Errorf("bad conf: %s: %w", l.Match, err)
	}

	return &fr, nil
}
