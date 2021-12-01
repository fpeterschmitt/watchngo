package pkg

import (
	"fmt"
	"log"
	"os"

	"github.com/go-ini/ini"
)

// Executor names.
const (
	ExecutorUnixShell = "unixshell"
	ExecutorStdout    = "stdout"
	ExecutorRaw       = "raw"
)

func ExecutorFromName(name string) (Executor, error) {
	switch name {
	case ExecutorRaw:
		return NewExecutorRaw(os.Stdout), nil
	case ExecutorStdout:
		return NewExecutorPrintPath(os.Stdout), nil
	case ExecutorUnixShell:
		return NewExecutorUnixShell(os.Stdout), nil
	default:
		return nil, fmt.Errorf("conf: unknown executor type %s", name)
	}
}

func WatcherFromConf(section *ini.Section, logger *log.Logger, debug bool, defExecutorName string) (*Watcher, error) {
	name := section.Name()

	match := section.Key("match").String()
	if match == "" {
		return nil, fmt.Errorf("conf: missing required 'match' key")
	}

	command := section.Key("command").MustString("")
	if command == "" {
		return nil, fmt.Errorf("conf: missing required 'command' key")
	}

	filter := section.Key("filter").MustString("")
	executor, err := ExecutorFromName(section.Key("executor").MustString(defExecutorName))
	if err != nil {
		return nil, err
	}

	debug = section.Key("debug").MustBool(debug)

	w, err := NewWatcher(
		name,
		match,
		filter,
		command,
		executor,
		debug,
		logger,
	)

	return w, err
}

// WatchersFromConf returns watchers from a configuration file provided at location "path".
func WatchersFromConf(cfg *ini.File, logger *log.Logger) ([]*Watcher, error) {
	// we only have the DEFAULT section
	if len(cfg.Sections()) == 1 {
		return nil, fmt.Errorf("conf: no configuration")
	}

	defaultSection := cfg.Section(ini.DefaultSection)
	debug := defaultSection.Key("debug").MustBool(false)
	defExecutorName := ExecutorUnixShell
	if defaultSection.HasKey("executor") {
		defExecutorName = defaultSection.Key("executor").String()
	}

	watchers := make([]*Watcher, 0)
	// exclude the DEFAULT section, which comes first
	for _, section := range cfg.Sections()[1:] {
		if w, err := WatcherFromConf(section, logger, debug, defExecutorName); err != nil {
			return nil, err
		} else {
			watchers = append(watchers, w)
		}
	}

	return watchers, nil
}
