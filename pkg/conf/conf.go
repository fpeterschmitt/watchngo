package conf

import (
	"fmt"

	"github.com/Leryan/watchngo/pkg/watcher"
	"github.com/fsnotify/fsnotify"

	"github.com/go-ini/ini"
)

// WatchersFromPath returns configuration from file at path
func WatchersFromPath(path string) ([]*watcher.Watcher, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("conf: from path: %s: %v", path, err)
	}

	// we only have the DEFAULT section
	if len(cfg.Sections()) == 1 {
		return nil, fmt.Errorf("conf: no configuration")
	}

	watchers := make([]*watcher.Watcher, 0)

	defaultSection := cfg.Section(ini.DEFAULT_SECTION)

	debug := false
	if defaultSection.HasKey("debug") {
		debug, err = defaultSection.Key("debug").Bool()
		if err != nil {
			return nil, fmt.Errorf("conf: debug is not a bool: %v", err)
		}
	}

	// exclude the DEFAULT section, which comes first
	for _, section := range cfg.Sections()[1:] {
		watcher := watcher.Watcher{
			Name:  section.Name(),
			Debug: debug,
		}

		fswatcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil, fmt.Errorf("fsnotify: %v", err)
		}

		watcher.FSWatcher = fswatcher

		if section.HasKey("match") {
			watcher.Match = section.Key("match").String()
		} else {
			return nil, fmt.Errorf("conf: missing required match key")
		}

		if section.HasKey("command") {
			watcher.Command = section.Key("command").String()
		} else {
			return nil, fmt.Errorf("conf: missing required command key: %v", err)
		}

		if section.HasKey("filter") {
			watcher.Filter = section.Key("filter").String()
		}

		if section.HasKey("debug") {
			debug, err := section.Key("debug").Bool()
			if err != nil {
				return nil, fmt.Errorf("conf: debug is not a bool: %v", err)
			}
			watcher.Debug = debug
		}

		watchers = append(watchers, &watcher)
	}

	return watchers, nil
}
