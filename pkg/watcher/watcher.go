package watcher

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher ...
type Watcher struct {
	Name      string
	Command   string
	Match     string
	FSWatcher *fsnotify.Watcher
	executing bool
	eLock     sync.RWMutex
}

// Find add files to the watcher. Currently only one file with it's exact
// path (may be relative) is supported.
func (w *Watcher) Find() error {
	err := w.FSWatcher.Add(w.Match)
	if err != nil {
		return fmt.Errorf("on match: %s: %v", w.Match, err)
	}
	return nil
}

func (w *Watcher) setExecuting(executing bool) {
	w.eLock.Lock()
	defer w.eLock.Unlock()
	w.executing = executing
}

func (w *Watcher) getExecuting() bool {
	w.eLock.RLock()
	defer w.eLock.RUnlock()
	return w.executing
}

func (w *Watcher) exec(command string) {
	w.setExecuting(true)

	rp, wp := io.Pipe()
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Stdout = wp
	cmd.Stderr = wp

	go func() {
		if err := cmd.Run(); err != nil {
			log.Println(err)
		}
		wp.Close()
	}()

	for {
		if cmd.ProcessState != nil {
			if cmd.ProcessState.Exited() {
				break
			}
		}
		b := make([]byte, 1024)
		rp.Read(b)
		fmt.Printf("%s", string(b))
	}
	b, _ := ioutil.ReadAll(rp)
	fmt.Printf("%s", string(b))

	w.setExecuting(false)
}

// Work fires the watcher and run commands when an event is received.
func (w *Watcher) Work() error {
	log.Printf("running watcher %v", w.Name)
	matchstat, err := os.Stat(w.Match)
	if err != nil {
		return fmt.Errorf("worker: %s: %v", w.Name, err)
	}

	// just to be very explicit
	isFile := !matchstat.IsDir()
	isDir := matchstat.IsDir()

	w.setExecuting(false)

	go func() {
		for {
			select {
			case event := <-w.FSWatcher.Events:
				log.Printf("event: %v", event)
				log.Printf("command: %v", w.Command)

				isWrite := fsnotify.Write&event.Op == fsnotify.Write
				isRemove := fsnotify.Remove&event.Op == fsnotify.Remove
				isChmod := fsnotify.Chmod&event.Op == fsnotify.Chmod
				//isCreate := fsnotify.Create&event.Op == fsnotify.Create
				isRename := fsnotify.Rename&event.Op == fsnotify.Rename

				if w.getExecuting() {
					log.Printf("already running, ignoring")
					break
				}

				if (isWrite || isChmod) && isFile {
					w.exec(w.Command)

				} else if (isRemove || isRename) && isFile {
					// FIXIT: ...
					time.Sleep(time.Millisecond * 10)

					_, err := os.Stat(event.Name)
					if err == nil {
						w.exec(w.Command)
						w.FSWatcher.Add(event.Name)
					}

				} else if isDir {
					w.exec(w.Command)
				}

			case err := <-w.FSWatcher.Errors:
				log.Printf("error: %v", err)
			}
		}
	}()

	return nil
}
