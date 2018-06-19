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

type Watcher struct {
	Name      string
	Command   string
	Match     string
	FSWatcher *fsnotify.Watcher
	executing bool
	eLock     sync.RWMutex
}

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

func (W *Watcher) exec(command string) {
	W.setExecuting(true)

	r, w := io.Pipe()
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Stdout = w
	cmd.Stderr = w

	go func() {
		if err := cmd.Run(); err != nil {
			log.Println(err)
		}
		w.Close()
	}()

	for {
		if cmd.ProcessState != nil {
			if cmd.ProcessState.Exited() {
				break
			}
		}
		b := make([]byte, 1024)
		r.Read(b)
		fmt.Printf("%s", string(b))
	}
	b, _ := ioutil.ReadAll(r)
	fmt.Printf("%s", string(b))

	W.setExecuting(false)
}

func (w *Watcher) Work() {
	log.Printf("running watcher %v", w.Name)
	matchstat, err := os.Stat(w.Match)
	if err != nil {
		log.Fatalf("worker: %s: %v", w.Name, err)
	}

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
				isCreate := fsnotify.Create&event.Op == fsnotify.Create

				if w.getExecuting() {
					log.Printf("already running, ignoring")
					break
				}

				if isWrite || isChmod {
					w.exec(w.Command)

				} else if isRemove {
					time.Sleep(time.Millisecond * 10)
					_, err := os.Stat(event.Name)
					if err == nil {
						w.exec(w.Command)
						w.FSWatcher.Add(event.Name)
					}
				} else if isCreate && matchstat.IsDir() {
					w.exec(w.Command)
				}

			case err := <-w.FSWatcher.Errors:
				log.Printf("error: %v", err)
			}
		}
	}()
}
