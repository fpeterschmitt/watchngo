package main

import (
	"log"

	"github.com/Leryan/watchngo/pkg/conf"

	"flag"
)

func main() {
	flagCfg := flag.String("conf", "watchngo.ini", "configuration file path")
	flag.Parse()

	watchers, err := conf.WatchersFromPath(*flagCfg)

	if err != nil {
		log.Fatalf("error: WatchersFromPath: %v", err)
	}

	forever := make(chan bool, 1)
	working := 0

	for _, watcher := range watchers {
		if err = watcher.Find(); err != nil {
			log.Printf("error: watcher.Find: %s: %v", watcher.Name, err)
			continue
		}

		if err = watcher.Work(); err != nil {
			log.Printf("error: watcher.Work: %s: %v", watcher.Name, err)
			continue
		}

		working++
	}

	if working > 0 {
		<-forever
	} else {
		log.Fatalf("error: no watcher working")
	}
}
