package pkg

import (
	"fmt"
	"sync"
	"time"
)

// Watcher ...
type Watcher struct {
	Name       string
	Finder     Finder
	Filter     Filter
	Logger     Logger
	Executor   Executor
	FSWatcher  Notifier
	eLock      sync.RWMutex
	eventQueue chan NotificationEvent
}

func (w *Watcher) exec(event NotificationEvent, eventFile string) {
	w.Logger.Log("running command on watcher \"%s\"", w.Name)
	err := w.Executor.Exec(event, eventFile)

	if err == nil {
		w.Logger.Log("finished running command on watcher \"%s\"", w.Name)
	} else {
		w.Logger.Log("finished running command on watcher \"%s\" with error: %v", w.Name, err)
	}
}

func (w *Watcher) handleFSEvent(event NotificationEvent, eventFile string) bool {
	w.Logger.Debug("event: %v", event)

	if eventFile == "" {
		return false
	}

	if !w.Filter.MatchString(eventFile) {
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
						w.exec(event, event.Path)
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
	res, err := w.Finder.Find()
	if err != nil {
		return err
	}

	for _, location := range res.Locations {
		w.Logger.Debug("add location %s", location)
		if err := w.FSWatcher.Add(location); err != nil {
			return err
		}
	}

	go w.eventQueueConsumer()
	defer w.FSWatcher.Close()
	defer func() { close(w.eventQueue) }()

	w.Logger.Log("running watcher \"%s\"", w.Name)

	events := w.FSWatcher.Events()

	for {
		event := <-events

		w.Logger.Debug("pre-filtering event: %v", event)

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

func NewWatcher(name string, finder Finder, filter Filter, notifier Notifier, executor Executor, logger Logger) (*Watcher, error) {
	if logger == nil {
		return nil, fmt.Errorf("new watcher: logger cannot be nil")
	}

	if executor == nil {
		return nil, fmt.Errorf("new watcher: executor cannot be nil")
	}

	if notifier == nil {
		return nil, fmt.Errorf("new watcher: notifier cannot be nil")
	}

	if finder == nil {
		return nil, fmt.Errorf("new watcher: finder cannot be nil")
	}

	if filter == nil {
		return nil, fmt.Errorf("new watcher: filter cannot be nil")
	}

	watcher := &Watcher{
		Name:       name,
		FSWatcher:  notifier,
		Logger:     logger,
		Executor:   executor,
		Filter:     filter,
		Finder:     finder,
		eventQueue: make(chan NotificationEvent),
	}

	return watcher, nil
}
