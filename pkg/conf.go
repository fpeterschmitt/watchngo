package pkg

import (
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/go-ini/ini"
)

// Executor names.
const (
	ExecutorUnixShell = "unixshell"
	ExecutorStdout    = "stdout"
	ExecutorRaw       = "raw"
)

type ExecutorProvider func(name, commandTemplate string) (Executor, error)

func ExecutorFromName(name, commandTemplate string) (Executor, error) {
	switch name {
	case ExecutorRaw:
		return NewExecutorRaw(os.Stdout, commandTemplate), nil
	case ExecutorStdout:
		return NewExecutorPrintPath(os.Stdout), nil
	case ExecutorUnixShell:
		return NewExecutorUnixShell(os.Stdout, commandTemplate), nil
	default:
		return nil, fmt.Errorf("conf: unknown executor type %s", name)
	}
}

func WatcherFromConf(section *ini.Section, logger *log.Logger, debug bool, defExecutorName string, prov ExecutorProvider) (*Watcher, error) {
	name := section.Name()

	match := section.Key("match").String()
	if match == "" {
		return nil, fmt.Errorf("conf: missing required 'match' key")
	}

	command := section.Key("command").MustString("")
	if command == "" {
		return nil, fmt.Errorf("conf: missing required 'command' key")
	}

	filter := regexp.MustCompile(section.Key("filter").MustString(".*"))

	executor, err := prov(section.Key("executor").MustString(defExecutorName), command)
	if err != nil {
		return nil, err
	}

	debug = section.Key("debug").MustBool(debug)

	finder := LocalFinder{Match: match}

	notifier := NewFSNotifyNotifier()

	var wLogger Logger
	wLogger = InfoLogger{Logger: logger}
	if debug {
		wLogger = DebugLogger{Logger: wLogger}
	}

	w, err := NewWatcher(
		name,
		finder,
		filter,
		notifier,
		executor,
		wLogger,
	)

	return w, err
}

func BuildCfgFrom(name, match, filter, command, executor string, debug bool) *ini.File {
	cfg := ini.Empty()
	section, err := cfg.NewSection(name)
	if err != nil {
		panic(err)
	}

	section.NewKey("match", match)
	section.NewKey("command", command)

	if filter != "" {
		section.NewKey("filter", filter)
	}

	if debug {
		section.NewBooleanKey("debug")
	}

	if executor != "" {
		section.NewKey("executor", executor)
	}

	return cfg
}

// WatchersFromConf returns watchers from a configuration file provided at location "path".
func WatchersFromConf(cfg *ini.File, logger *log.Logger, prov ExecutorProvider) ([]*Watcher, error) {
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
		if w, err := WatcherFromConf(section, logger, debug, defExecutorName, prov); err != nil {
			return nil, err
		} else {
			watchers = append(watchers, w)
		}
	}

	return watchers, nil
}
