package main

import (
	"log"

	"github.com/Leryan/watchngo/pkg/conf"

	"flag"

	"github.com/fsnotify/fsnotify"
)

func main() {
	flagCfg := flag.String("conf", "watchngo.ini", "configuration file path")
	flag.Parse()

	watchers, err := conf.FromPath(*flagCfg)

	if err != nil {
		log.Fatalf("error: %v", err)
	}

	forever := make(chan bool, 1)

	for _, watcher := range watchers {
		fswatcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatalf("error: %v", err)
		}

		watcher.FSWatcher = fswatcher

		if err = watcher.Find(); err != nil {
			log.Fatalf("error: %v", err)
		}

		if err = watcher.Work(); err != nil {
			log.Fatalf("error: %v", err)
		}
	}

	<-forever
}
