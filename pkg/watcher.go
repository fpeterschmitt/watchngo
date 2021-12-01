package pkg

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Watcher ...
type Watcher struct {
	Name       string
	Command    string
	Match      string
	Filter     string
	Logger     Logger
	Executor   Executor
	FSWatcher  Notifier
	eLock      sync.RWMutex
	filter     *regexp.Regexp
	eventQueue chan NotificationEvent
}

// find add files to the watcher. Currently only one file with it's exact
// path (may be relative) is supported.
func (w *Watcher) find() error {
	matchstat, err := os.Stat(w.Match)
	var matches []string

	if err == nil && matchstat.IsDir() {
		matches, _, err = FindRecursive(w.Match)
		if err != nil {
			return fmt.Errorf("find: %w", err)
		}

		w.Logger.Debug("watcher %s found recursive directories", w.Name)

	} else if err == nil && !matchstat.IsDir() {
		matches = append(matches, w.Match)

		w.Logger.Debug("watcher %s use single file", w.Name)

	} else if err != nil {
		matches, err = FindGlob(w.Match, matches)

		if err != nil {
			return fmt.Errorf("glob: %w", err)
		} else if len(matches) == 0 {
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

	for _, match := range matches {
		w.Logger.Debug("add match: %s", match)
		if err := w.FSWatcher.Add(match); err != nil {
			return fmt.Errorf("on match: %s: %w", match, err)
		}
	}
	return nil
}

func (w *Watcher) exec(command string) {
	w.Logger.Log("running command on watcher \"%s\"", w.Name)
	err := w.Executor.Exec(command)

	if err == nil {
		w.Logger.Log("finished running command on watcher \"%s\"", w.Name)
	} else {
		w.Logger.Log("finished running command on watcher \"%s\" with error: %v", w.Name, err)
	}
}

func (w *Watcher) makeCommand(event NotificationEvent, eventFile string) string {
	command := strings.Replace(w.Command, "%match", w.Match, -1)
	command = strings.Replace(command, "%filter", w.Filter, -1)
	command = strings.Replace(command, "%event.file", eventFile, -1)
	command = strings.Replace(command, "%event.op", event.Notification.String(), -1)
	return command
}

func (w *Watcher) handleFSEvent(event NotificationEvent, eventFile string) bool {
	w.Logger.Debug("event: %v", event)

	if w.filter != nil && !w.filter.MatchString(eventFile) {
		return false
	}

	if eventFile == "" {
		return false
	}

	isWrite := NotificationWrite&event.Notification == NotificationWrite
	isRemove := NotificationRemove&event.Notification == NotificationRemove
	isChmod := NotificationChmod&event.Notification == NotificationChmod
	isCreate := NotificationCreate&event.Notification == NotificationCreate
	isRename := NotificationRename&event.Notification == NotificationRename

	if event.Error != nil && !isRemove && !isRename {
		w.Logger.Log("worker: %s: %v", eventFile, event.Error)
		return false
	}

	isFile := event.FileType == FileTypeFile
	isDir := event.FileType == FileTypeDir

	if w.Executor.Running() {
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
	events := make([]NotificationEvent, 0)

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
					if w.handleFSEvent(event, event.Path) && !executed {
						w.exec(w.makeCommand(event, event.Path))
						executed = true
					}
				}
				events = make([]NotificationEvent, 0)
				evtDate = time.Now()
			}
			timer.Reset(timerInterval)
		}
	}
}

// Work fires the watcher and run commands when an event is received.
func (w *Watcher) Work() error {
	defer w.FSWatcher.Close()
	defer func() { close(w.eventQueue) }()
	go w.eventQueueConsumer()

	w.Logger.Log("running watcher \"%s\"", w.Name)

	events := w.FSWatcher.Events()

	for {
		event := <-events

		if event.Notification&NotificationError == NotificationError {
			if event.Path == "" {
				w.Logger.Log("watcher \"%s\" stopped: %v -> closed: %v", w.Name, event.Error, w.FSWatcher.Close())
				return event.Error
			}
		} else {
			w.eventQueue <- event
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
func NewWatcher(name, match, filter, command string, executor Executor, notifier Notifier, debug bool, logger *log.Logger) (*Watcher, error) {
	if logger == nil {
		return nil, fmt.Errorf("new watcher: logger cannot be nil")
	}

	if executor == nil {
		return nil, fmt.Errorf("new watcher: executor cannot be nil")
	}

	if notifier == nil {
		return nil, fmt.Errorf("new watcher: notifier cannot be nil")
	}

	var wLogger Logger
	wLogger = InfoLogger{Logger: logger}
	if debug {
		wLogger = DebugLogger{Logger: wLogger}
	}

	watcher := &Watcher{
		Name:       name,
		Filter:     filter,
		Command:    command,
		FSWatcher:  notifier,
		Match:      match,
		Logger:     wLogger,
		Executor:   executor,
		eventQueue: make(chan NotificationEvent),
	}

	return watcher, watcher.find()
}
