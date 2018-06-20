package watcher

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
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
	WithShell  bool
	Debug      bool
	executing  bool
	eLock      sync.RWMutex
	filter     *regexp.Regexp
	eventQueue chan fsnotify.Event
}

// Find add files to the watcher. Currently only one file with it's exact
// path (may be relative) is supported.
func (w *Watcher) Find() error {
	matchstat, err := os.Stat(w.Match)
	matches := make([]string, 0)

	if err == nil && matchstat.IsDir() {
		matches, err = utils.FindRecursive(w.Match, matches)
		if err != nil {
			return fmt.Errorf("find: %v", err)
		}

		if w.Debug {
			log.Printf("watcher %s found recursive directories", w.Name)
		}

	} else if err == nil && !matchstat.IsDir() {
		matches = append(matches, w.Match)

		if w.Debug {
			log.Printf("watcher %s use single file", w.Name)
		}

	} else if err != nil {
		matches, err = utils.FindGlob(w.Match, matches)

		if err != nil {
			return fmt.Errorf("glob: %v", err)
		} else if len(matches) == 0 {
			return fmt.Errorf("empty glob: %s", w.Match)
		}

		if w.Debug {
			log.Printf("watcher %s use glob match", w.Name)
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

	for _, match := range matches {
		if w.Debug {
			log.Printf("add match: %s", match)
		}
		err := w.FSWatcher.Add(match)
		if err != nil {
			return fmt.Errorf("on match: %s: %v", match, err)
		}
	}
	return nil
}

func (w *Watcher) setExecuting(executing bool) {
	w.eLock.Lock()
	defer w.eLock.Unlock()
	w.executing = executing
}

func (w *Watcher) getExecuting() bool {
	w.eLock.Lock()
	defer w.eLock.Unlock()
	return w.executing
}

func (w *Watcher) exec(command string) {
	w.setExecuting(true)
	defer w.setExecuting(false)

	rp, wp := io.Pipe()
	var cmd *exec.Cmd

	if w.WithShell {
		cmd = exec.Command("/bin/sh", "-c", command)
	} else {
		cmd = exec.Command(command)
	}
	cmd.Stdout = wp
	cmd.Stderr = wp

	log.Printf("running command for watcher %s", w.Name)

	execFinished := make(chan bool, 1)

	go func() {
		if err := cmd.Run(); err != nil {
			log.Printf("watcher %s: %v", w.Name, err)
		}
		wp.Close()
		execFinished <- true
	}()

	reader := bufio.NewReader(rp)

	for {
		b, err := reader.ReadBytes('\n')

		if len(b) > 0 {
			fmt.Printf("%s", string(b))
		}

		if err != nil {
			break
		}
	}

	if reader.Buffered() > 0 {
		b, _ := ioutil.ReadAll(reader)
		fmt.Printf("%s", string(b))
	}

	<-execFinished
	log.Printf("finished command for watcher %s", w.Name)
}

func (w *Watcher) handleFSEvent(event fsnotify.Event, executed bool) bool {
	eventFile := path.Clean(event.Name)

	command := strings.Replace(w.Command, "%match", w.Match, -1)
	command = strings.Replace(command, "%filter", w.Filter, -1)
	command = strings.Replace(command, "%event.file", eventFile, -1)
	command = strings.Replace(command, "%event.op", event.Op.String(), -1)

	if w.Debug {
		log.Printf("event: %v", event)
		log.Printf("command: %v", command)
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
		log.Printf("worker: %s: %v", eventFile, err)
		return false
	}

	isFile := false
	isDir := false

	if eventFileStat != nil {
		isFile = !eventFileStat.IsDir()
		isDir = eventFileStat.IsDir()
	}

	if w.getExecuting() {
		if w.Debug {
			log.Printf("already running, ignoring")
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
			}

			time.Sleep(time.Millisecond * 100)
			retries++
		}

		if retries == 10 {
			log.Printf("cannot re-add file: %s", eventFile)
		}
	} else if isDir {
		mustExec = true
	}

	if mustExec && !executed {
		w.exec(command)
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
					log.Printf("sending %d events", len(events))
					log.Printf("events: %v", events)
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
	w.setExecuting(false)

	w.eventQueue = make(chan fsnotify.Event)
	go w.eventQueueConsumer()

	log.Printf("running watcher %v", w.Name)

	for {
		select {
		case event := <-w.FSWatcher.Events:
			w.eventQueue <- event

		case err := <-w.FSWatcher.Errors:
			log.Printf("error: %v, watcher %s stopped", err, w.Name)
			w.FSWatcher.Close()
			return err
		}
	}
}
