package main

import (
	"log"
	"sync"

	"github.com/Leryan/watchngo/pkg/conf"

	"github.com/fsnotify/fsnotify"
)

func main() {
	watchers, err := conf.FromPath("watchngo.ini")
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	wg := sync.WaitGroup{}
	for _, watcher := range watchers {
		fswatcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		watcher.FSWatcher = fswatcher
		err = watcher.Find()
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		watcher.Work()
		wg.Add(1)
	}
	wg.Wait()
}
