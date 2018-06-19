package watcher

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Leryan/watchngo/pkg/utils"
	"github.com/fsnotify/fsnotify"
)

// Watcher ...
type Watcher struct {
	Name      string
	Command   string
	Match     string
	Filter    string
	FSWatcher *fsnotify.Watcher
	Shell     string
	WithShell bool
	Debug     bool
	executing bool
	eLock     sync.RWMutex
	filter    *regexp.Regexp
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

	go func() {
		if err := cmd.Run(); err != nil {
			log.Printf("watcher %s: %v", w.Name, err)
		}
		wp.Close()
	}()

	for {
		b := make([]byte, 1024)
		n, err := rp.Read(b)

		if n > 0 {
			fmt.Printf("%s", string(b))
		}

		if err != nil {
			break
		}
	}

	b, _ := ioutil.ReadAll(rp)
	fmt.Printf("%s", string(b))

	log.Printf("finished command for watcher %s", w.Name)
}

func (w *Watcher) handleFSEvent(event fsnotify.Event) {
	command := strings.Replace(w.Command, "%match", w.Match, -1)
	command = strings.Replace(command, "%filter", w.Filter, -1)
	command = strings.Replace(command, "%event.file", event.Name, -1)
	command = strings.Replace(command, "%event.op", event.Op.String(), -1)

	if w.Debug {
		log.Printf("event: %v", event)
		log.Printf("command: %v", command)
	}

	if w.filter != nil && !w.filter.MatchString(event.Name) {
		return
	}

	matchstat, err := os.Stat(event.Name)
	if err != nil {
		log.Printf("worker: %s: %v", event.Name, err)
		return
	}

	// just to be very explicit
	isFile := !matchstat.IsDir()
	isDir := matchstat.IsDir()

	isWrite := fsnotify.Write&event.Op == fsnotify.Write
	isRemove := fsnotify.Remove&event.Op == fsnotify.Remove
	isChmod := fsnotify.Chmod&event.Op == fsnotify.Chmod
	//isCreate := fsnotify.Create&event.Op == fsnotify.Create
	isRename := fsnotify.Rename&event.Op == fsnotify.Rename

	if w.getExecuting() {
		if w.Debug {
			log.Printf("already running, ignoring")
		}
		return
	}

	if (isWrite || isChmod) && isFile {
		go w.exec(command)

	} else if (isRemove || isRename) && isFile {
		// FIXIT: ...
		time.Sleep(time.Millisecond * 100)

		_, err := os.Stat(event.Name)
		if err == nil {
			w.FSWatcher.Add(event.Name)
			go w.exec(command)
		} else {
			log.Fatalf("cannot re-add file: %s", event.Name)
		}

	} else if isDir {
		go w.exec(command)
	}

	time.Sleep(time.Millisecond * 10)

}

// Work fires the watcher and run commands when an event is received.
func (w *Watcher) Work() error {
	w.setExecuting(false)

	go func() {
		for {
			select {
			case event := <-w.FSWatcher.Events:
				w.handleFSEvent(event)

			case err := <-w.FSWatcher.Errors:
				log.Printf("error: %v", err)
			}
		}
	}()

	log.Printf("running watcher %v", w.Name)

	return nil
}
