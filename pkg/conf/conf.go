package conf

import (
	"fmt"

	"github.com/Leryan/watchngo/pkg/watcher"

	"github.com/go-ini/ini"
)

type Conf struct {
	// watcher -> command
	m map[string]watcher.Watcher
}

// FromPath returns configuration from file at path
func FromPath(path string) ([]watcher.Watcher, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("conf: from path: %s: %v", path, err)
	}

	// we only have the DEFAULT section
	if len(cfg.Sections()) == 1 {
		return nil, fmt.Errorf("conf: no configuration")
	}

	watchers := make([]watcher.Watcher, 0)
	// exclude the DEFAULT section, which comes first
	for _, section := range cfg.Sections()[1:] {
		wName := section.Name()

		// match
		kMatch, err := section.GetKey("match")
		if err != nil {
			return nil, fmt.Errorf("conf: match: %v", err)
		}
		wMatch := kMatch.String()

		// command
		kCommand, err := section.GetKey("command")
		if err != nil {
			return nil, fmt.Errorf("conf: command: %v", err)
		}
		wCommand := kCommand.String()

		watcher := watcher.Watcher{
			Name:    wName,
			Command: wCommand,
			Match:   wMatch,
		}
		watchers = append(watchers, watcher)
	}

	return watchers, nil
}
