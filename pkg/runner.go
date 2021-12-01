package pkg

import (
	"log"
	"sync"
)

func RunForever(watchers []*Watcher) {
	wg := sync.WaitGroup{}

	for _, watcher := range watchers {
		wg.Add(1)
		go func(w *Watcher) {
			defer wg.Done()
			if err := w.Work(); err != nil {
				log.Printf("watcher returned with error: %v", err)
			}
		}(watcher)
	}

	wg.Wait()
}
