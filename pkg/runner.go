package pkg

import (
	"log"
)

// Run all watchers
func Run(watchers []*Watcher) {
	forever := make(chan bool, 1)
	working := 0

	for _, watcher := range watchers {
		if err := watcher.Find(); err != nil {
			log.Printf("error: watcher.Find: %s: %v", watcher.Name, err)
			continue
		}

		go watcher.Work()

		working++
	}

	if working > 0 {
		<-forever
	} else {
		log.Fatalf("error: no watcher working")
	}
}
