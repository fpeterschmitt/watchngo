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

const (
	CfgDebug    = "debug"
	CfgSilent   = "silent"
	CfgMatch    = "match"
	CfgFilter   = "filter"
	CfgCommand  = "command"
	CfgExecutor = "executor"
)

type Cfg struct {
	// available for defaults
	Debug        bool
	ExecutorName string
	Silent       bool
	// NOT available for defaults
	Name string
	// Match defaults to "." if it is empty
	Match string
	// Filter defaults to ".*" if it is empty
	Filter string
	// CommandTemplate is required
	CommandTemplate string
}

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

// BuildIniCfgFrom creates an in-memory ini config to be used with WatcherFromConf or WatchersFromConf.
func BuildIniCfgFrom(cfg Cfg) *ini.File {
	iniCfg := ini.Empty()

	section, err := iniCfg.NewSection(cfg.Name)
	if err != nil {
		panic(err)
	}

	section.NewKey(CfgMatch, cfg.Match)
	section.NewKey(CfgCommand, cfg.CommandTemplate)

	if cfg.Filter != "" {
		section.NewKey(CfgFilter, cfg.Filter)
	}

	if cfg.Debug && !cfg.Silent {
		section.NewKey(CfgDebug, "true")
	}

	if cfg.Silent {
		section.NewKey(CfgSilent, "true")
	}

	if cfg.ExecutorName != "" {
		section.NewKey(CfgExecutor, cfg.ExecutorName)
	}

	return iniCfg
}

func WatcherFromConf(iniCfg *ini.Section, logger *log.Logger, defaults Cfg, prov ExecutorProvider) (*Watcher, error) {
	name := iniCfg.Name()
	match := iniCfg.Key(CfgMatch).MustString(".")
	command := iniCfg.Key(CfgCommand).String()
	if command == "" {
		return nil, fmt.Errorf("conf: missing required 'command' key")
	}
	filter := regexp.MustCompile(iniCfg.Key(CfgFilter).MustString(".*"))
	executor, err := prov(iniCfg.Key(CfgExecutor).MustString(defaults.ExecutorName), command)
	if err != nil {
		return nil, err
	}

	debug := iniCfg.Key(CfgDebug).MustBool(defaults.Debug)
	silent := iniCfg.Key(CfgSilent).MustBool(defaults.Silent)

	finder := LocalFinder{Match: match}

	notifier := NewFSNotifyNotifier()

	var wLogger Logger
	wLogger = InfoLogger{Logger: logger}
	if debug {
		wLogger = DebugLogger{Logger: wLogger}
	}

	if silent {
		wLogger = SilentLogger{}
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

func WatchersFromConf(inicfg *ini.File, logger *log.Logger, prov ExecutorProvider) ([]*Watcher, error) {
	// we only have the DEFAULT section
	if len(inicfg.Sections()) == 1 {
		return nil, fmt.Errorf("conf: no configuration")
	}

	defaultSection := inicfg.Section(ini.DefaultSection)

	// get default values from the default section of the ini file.
	// fallback to hardcoded values for keys missing in that section.
	defaults := Cfg{
		Debug:        defaultSection.Key(CfgDebug).MustBool(false),
		ExecutorName: defaultSection.Key(CfgExecutor).MustString(ExecutorUnixShell),
		Silent:       defaultSection.Key(CfgSilent).MustBool(false),
	}

	watchers := make([]*Watcher, 0)
	// exclude the DEFAULT section, which comes first
	for _, section := range inicfg.Sections()[1:] {
		if w, err := WatcherFromConf(section, logger, defaults, prov); err != nil {
			return nil, err
		} else {
			watchers = append(watchers, w)
		}
	}

	return watchers, nil
}
