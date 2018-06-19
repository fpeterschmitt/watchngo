package watcher

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
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
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Stdout = wp
	cmd.Stderr = wp

	log.Printf("running command for watcher %s", w.Name)

	go func() {
		if err := cmd.Run(); err != nil {
			log.Println(err)
		}
		wp.Close()
	}()

	for {
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			break
		}

		b := make([]byte, 1024)
		n, _ := rp.Read(b)

		if n > 0 {
			fmt.Printf("%s", string(b))
		}
	}

	b, _ := ioutil.ReadAll(rp)
	fmt.Printf("%s", string(b))

	log.Printf("finished command for watcher %s", w.Name)
}

// Work fires the watcher and run commands when an event is received.
func (w *Watcher) Work() error {
	w.setExecuting(false)

	go func() {
		for {
			select {
			case event := <-w.FSWatcher.Events:
				if w.Debug {
					log.Printf("event: %v", event)
					log.Printf("command: %v", w.Command)
				}

				if w.filter != nil && !w.filter.MatchString(event.Name) {
					break
				}

				matchstat, err := os.Stat(event.Name)
				if err != nil {
					log.Printf("worker: %s: %v", event.Name, err)
					break
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
					break
				}

				if (isWrite || isChmod) && isFile {
					go w.exec(w.Command)

				} else if (isRemove || isRename) && isFile {
					// FIXIT: ...
					time.Sleep(time.Millisecond * 10)

					_, err := os.Stat(event.Name)
					if err == nil {
						go w.exec(w.Command)
						w.FSWatcher.Add(event.Name)
					}

				} else if isDir {
					go w.exec(w.Command)
				}

				time.Sleep(time.Millisecond * 10)

			case err := <-w.FSWatcher.Errors:
				log.Printf("error: %v", err)
			}
		}
	}()

	log.Printf("running watcher %v", w.Name)

	return nil
}
