package pkg

import (
	"log"
	"sync"
)

func RunForever(watchers []*Watcher) {
	working := 0
	wg := sync.WaitGroup{}

	for _, watcher := range watchers {
		if err := watcher.Find(); err != nil {
			log.Printf("error: watcher.Find: %s: %v", watcher.Name, err)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := watcher.Work(); err != nil {
				log.Printf("watcher returned with error: %v", err)
			}
		}()

		working++
	}

	if working < 1 {
		log.Fatalf("error: no watcher working")
	}

	wg.Wait()
}
