package watcher

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Leryan/watchngo/pkg/utils"
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
	Debug      bool
	Logger     *log.Logger
	executor   Executor
	eLock      sync.RWMutex
	filter     *regexp.Regexp
	eventQueue chan fsnotify.Event
}

// Find add files to the watcher. Currently only one file with it's exact
// path (may be relative) is supported.
func (w *Watcher) Find() error {
	matchstat, err := os.Stat(w.Match)
	wr := utils.NewWalkRec()

	if err == nil && matchstat.IsDir() {
		wr, err = utils.FindRecursive(w.Match, wr)
		if err != nil {
			return fmt.Errorf("find: %v", err)
		}

		if w.Debug {
			w.Logger.Printf("watcher %s found recursive directories", w.Name)
		}

	} else if err == nil && !matchstat.IsDir() {
		wr.Matches = append(wr.Matches, w.Match)

		if w.Debug {
			w.Logger.Printf("watcher %s use single file", w.Name)
		}

	} else if err != nil {
		wr.Matches, err = utils.FindGlob(w.Match, wr.Matches)

		if err != nil {
			return fmt.Errorf("glob: %v", err)
		} else if len(wr.Matches) == 0 {
			return fmt.Errorf("empty glob: %s", w.Match)
		}

		if w.Debug {
			w.Logger.Printf("watcher %s use glob match", w.Name)
		}

	} else {
		return fmt.Errorf("bad conf: %s: %v", w.Match, err)
	}

	if w.Filter != "" {
		rfilter, err := regexp.Compile(w.Filter)

		if err != nil {
			return fmt.Errorf("filter: %s: %v", w.Filter, rfilter)
		}

		w.filter = rfilter
	}

	for _, match := range wr.Matches {
		if w.Debug {
			w.Logger.Printf("add match: %s", match)
		}
		err := w.FSWatcher.Add(match)
		if err != nil {
			return fmt.Errorf("on match: %s: %v", match, err)
		}
	}
	return nil
}

func (w *Watcher) exec(command string, output io.Writer) {
	w.Logger.Printf("running command on watcher %s", w.Name)
	err := w.executor.Exec(command)

	if err == nil {
		w.Logger.Printf("finished running command on watcher %s", w.Name)
	} else {
		w.Logger.Printf("finished running command on watcher %s with error: %v", w.Name, err)
	}
}

func (w *Watcher) handleFSEvent(event fsnotify.Event, executed bool) bool {
	eventFile := path.Clean(event.Name)

	command := strings.Replace(w.Command, "%match", w.Match, -1)
	command = strings.Replace(command, "%filter", w.Filter, -1)
	command = strings.Replace(command, "%event.file", eventFile, -1)
	command = strings.Replace(command, "%event.op", event.Op.String(), -1)

	if w.Debug {
		w.Logger.Printf("event: %v", event)
		w.Logger.Printf("command: %v", command)
	}

	if w.filter != nil && !w.filter.MatchString(eventFile) {
		return false
	}

	if eventFile == "" {
		return false
	}

	isWrite := fsnotify.Write&event.Op == fsnotify.Write
	isRemove := fsnotify.Remove&event.Op == fsnotify.Remove
	isChmod := fsnotify.Chmod&event.Op == fsnotify.Chmod
	//isCreate := fsnotify.Create&event.Op == fsnotify.Create
	isRename := fsnotify.Rename&event.Op == fsnotify.Rename

	eventFileStat, err := os.Stat(eventFile)
	if err != nil && !isRemove && !isRename {
		w.Logger.Printf("worker: %s: %v", eventFile, err)
		return false
	}

	isFile := false
	isDir := false

	if eventFileStat != nil {
		isFile = !eventFileStat.IsDir()
		isDir = eventFileStat.IsDir()
	}

	if w.executor.Running() {
		if w.Debug {
			w.Logger.Printf("already running, ignoring")
		}
		return false
	}

	mustExec := false
	if (isWrite || isChmod) && isFile {
		mustExec = true
	} else if isRemove || isRename {
		w.FSWatcher.Remove(eventFile)

		retries := 0
		for retries < 10 {
			_, err := os.Stat(eventFile)

			if err == nil {
				w.FSWatcher.Add(eventFile)
				mustExec = true
				break
			} else if w.Debug {
				w.Logger.Printf("re-add attempt: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
			retries++
		}

		if retries == 10 {
			w.Logger.Printf("cannot re-add file: %s", eventFile)
		}
	} else if isDir {
		mustExec = true
	}

	if mustExec && !executed {
		w.exec(command, os.Stdout)
	}

	return mustExec
}

func (w *Watcher) eventQueueConsumer() {
	timerInterval := time.Millisecond * 500
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
				if w.Debug {
					w.Logger.Printf("sending %d events", len(events))
					w.Logger.Printf("events: %v", events)
				}
				executed := false
				for _, event := range events {
					executed = executed || w.handleFSEvent(event, executed)
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

	w.Logger.Printf("running watcher %v", w.Name)

	for {
		select {
		case event := <-w.FSWatcher.Events:
			w.eventQueue <- event

		case err := <-w.FSWatcher.Errors:
			w.Logger.Printf("error: %v, watcher %s stopped", err, w.Name)
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
		return nil, fmt.Errorf("logger cannot be nil")
	}

	if executor == nil {
		return nil, fmt.Errorf("executor cannot be nil")
	}

	return &Watcher{
		Name:      name,
		Filter:    filter,
		Command:   command,
		FSWatcher: fswatcher,
		Match:     match,
		Logger:    logger,
		Debug:     debug,
		executor:  executor,
	}, nil
}
