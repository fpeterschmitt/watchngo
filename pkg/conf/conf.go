package conf

import (
	"fmt"

	"github.com/Leryan/watchngo/pkg/watcher"

	"github.com/go-ini/ini"
)

// FromPath returns configuration from file at path
func FromPath(path string) ([]*watcher.Watcher, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("conf: from path: %s: %v", path, err)
	}

	// we only have the DEFAULT section
	if len(cfg.Sections()) == 1 {
		return nil, fmt.Errorf("conf: no configuration")
	}

	watchers := make([]*watcher.Watcher, 0)
	// exclude the DEFAULT section, which comes first
	for _, section := range cfg.Sections()[1:] {
		wName := section.Name()

		// match
		iMatch, err := section.GetKey("match")
		if err != nil {
			return nil, fmt.Errorf("conf: match: %v", err)
		}
		wMatch := iMatch.String()

		// command
		iCommand, err := section.GetKey("command")
		if err != nil {
			return nil, fmt.Errorf("conf: command: %v", err)
		}
		wCommand := iCommand.String()

		// filter
		iFilter, err := section.GetKey("filter")
		wFilter := ""
		if err == nil {
			wFilter = iFilter.String()
		}

		watcher := watcher.Watcher{
			Name:    wName,
			Command: wCommand,
			Match:   wMatch,
			Filter:  wFilter,
		}
		fmt.Println(watcher.Filter)
		watchers = append(watchers, &watcher)
	}

	return watchers, nil
}
