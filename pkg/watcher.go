package pkg

import (
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher ...
type Watcher struct {
	Name       string
	Command    string
	Match      string
	Filter     string
	FSWatcher  *fsnotify.Watcher
	Shell      string
	Logger     Logger
	executor   Executor
	eLock      sync.RWMutex
	filter     *regexp.Regexp
	eventQueue chan fsnotify.Event
}

// Find add files to the watcher. Currently only one file with it's exact
// path (may be relative) is supported.
func (w *Watcher) Find() error {
	matchstat, err := os.Stat(w.Match)
	wr := NewWalkRec()

	if err == nil && matchstat.IsDir() {
		wr, err = FindRecursive(w.Match, wr)
		if err != nil {
			return fmt.Errorf("find: %w", err)
		}

		w.Logger.Debug("watcher %s found recursive directories", w.Name)

	} else if err == nil && !matchstat.IsDir() {
		wr.Matches = append(wr.Matches, w.Match)

		w.Logger.Debug("watcher %s use single file", w.Name)

	} else if err != nil {
		wr.Matches, err = FindGlob(w.Match, wr.Matches)

		if err != nil {
			return fmt.Errorf("glob: %w", err)
		} else if len(wr.Matches) == 0 {
			return fmt.Errorf("empty glob: %s", w.Match)
		}

		w.Logger.Debug("watcher %s use glob match", w.Name)

	} else {
		return fmt.Errorf("bad conf: %s: %w", w.Match, err)
	}

	if w.Filter != "" {
		rfilter, err := regexp.Compile(w.Filter)

		if err != nil {
			return fmt.Errorf("filter: %s: %v", w.Filter, rfilter)
		}

		w.filter = rfilter
	}

	for _, match := range wr.Matches {
		w.Logger.Debug("add match: %s", match)
		if err := w.FSWatcher.Add(match); err != nil {
			return fmt.Errorf("on match: %s: %w", match, err)
		}
	}
	return nil
}

func (w *Watcher) exec(command string) {
	w.Logger.Log("running command on watcher \"%s\"", w.Name)
	err := w.executor.Exec(command)

	if err == nil {
		w.Logger.Log("finished running command on watcher \"%s\"", w.Name)
	} else {
		w.Logger.Log("finished running command on watcher \"%s\" with error: %v", w.Name, err)
	}
}

func (w *Watcher) makeCommand(event fsnotify.Event, eventFile string) string {
	command := strings.Replace(w.Command, "%match", w.Match, -1)
	command = strings.Replace(command, "%filter", w.Filter, -1)
	command = strings.Replace(command, "%event.file", eventFile, -1)
	command = strings.Replace(command, "%event.op", event.Op.String(), -1)
	return command
}

func (w *Watcher) handleFSEvent(event fsnotify.Event, eventFile string) bool {
	w.Logger.Debug("event: %v", event)

	if w.filter != nil && !w.filter.MatchString(eventFile) {
		return false
	}

	if eventFile == "" {
		return false
	}

	isWrite := fsnotify.Write&event.Op == fsnotify.Write
	isRemove := fsnotify.Remove&event.Op == fsnotify.Remove
	isChmod := fsnotify.Chmod&event.Op == fsnotify.Chmod
	isCreate := fsnotify.Create&event.Op == fsnotify.Create
	isRename := fsnotify.Rename&event.Op == fsnotify.Rename

	eventFileStat, err := os.Stat(eventFile)
	if err != nil && !isRemove && !isRename {
		w.Logger.Log("worker: %s: %v", eventFile, err)
		return false
	}

	isFile := false
	isDir := false

	if eventFileStat != nil {
		isFile = !eventFileStat.IsDir()
		isDir = eventFileStat.IsDir()
	}

	if w.executor.Running() {
		w.Logger.Debug("already running, ignoring")
		return false
	}

	mustExec := false
	if (isWrite || isChmod || isCreate) && isFile {
		mustExec = true
	} else if isRemove || isRename {
		_ = w.FSWatcher.Remove(eventFile)
		mustExec = true
	} else if isDir {
		mustExec = true
	}

	return mustExec
}

func (w *Watcher) eventQueueConsumer() {
	timerInterval := time.Millisecond * 250
	timer := time.NewTimer(timerInterval)
	evtDate := time.Now()
	events := make([]fsnotify.Event, 0)

	for {
		select {
		case event := <-w.eventQueue:
			events = append(events, event)
			evtDate = time.Now()
		case <-timer.C:
			if time.Now().Sub(evtDate) > timerInterval && len(events) > 0 {
				w.Logger.Debug("sending %d events", len(events))
				w.Logger.Debug("events: %v", events)
				executed := false
				for _, event := range events {
					eventFile := path.Clean(event.Name)
					if w.handleFSEvent(event, eventFile) && !executed {
						w.exec(w.makeCommand(event, eventFile))
						executed = true
					}
				}
				events = make([]fsnotify.Event, 0)
				evtDate = time.Now()
			}
			timer.Reset(timerInterval)
		}
	}
}

// Work fires the watcher and run commands when an event is received.
func (w *Watcher) Work() error {
	w.eventQueue = make(chan fsnotify.Event)
	go w.eventQueueConsumer()

	w.Logger.Log("running watcher \"%s\"", w.Name)

	for {
		select {
		case event := <-w.FSWatcher.Events:
			w.eventQueue <- event

		case err := <-w.FSWatcher.Errors:
			w.Logger.Log("watcher \"%s\" stopped: %v", w.Name, err)
			w.FSWatcher.Close()
			return err
		}
	}
}

// NewWatcher requires logger and executor to not be nil.
// param name is purely cosmetic, for logs.
// param match is a file, directory or "glob" path (shell-like).
// param command is a single command to run through executor.
// param executor is an instance of Executor that is not required to honor the given command, like for the Print executor.
// param debug shows more details when running.
// param logger must log to stderr when using executor Print.
func NewWatcher(name, match, filter, command string, executor Executor, debug bool, logger *log.Logger) (*Watcher, error) {
	fswatcher, err := fsnotify.NewWatcher()

	if err != nil {
		return nil, err
	}

	if logger == nil {
		return nil, fmt.Errorf("new watcher: logger cannot be nil")
	}

	if executor == nil {
		return nil, fmt.Errorf("new watcher: executor cannot be nil")
	}

	var wLogger Logger
	wLogger = InfoLogger{Logger: logger}
	if debug {
		wLogger = DebugLogger{Logger: wLogger}
	}

	return &Watcher{
		Name:      name,
		Filter:    filter,
		Command:   command,
		FSWatcher: fswatcher,
		Match:     match,
		Logger:    wLogger,
		executor:  executor,
	}, nil
}
