package main

import (
	"log"

	"github.com/Leryan/watchngo/pkg/conf"
	"github.com/Leryan/watchngo/pkg/watcher"
	"github.com/fsnotify/fsnotify"

	"flag"
)

func run(watchers []*watcher.Watcher) {
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

func main() {
	flagCfg := flag.String("conf", "watchngo.ini", "configuration file path")

	flagMatch := flag.String("match", "", "match")
	flagFilter := flag.String("filter", "", "filter")
	flagCommand := flag.String("command", "", "command")
	flagDebug := flag.Bool("debug", false, "debug")

	flag.Parse()

	if *flagCommand != "" && *flagMatch != "" {
		fswatcher, err := fsnotify.NewWatcher()

		if err != nil {
			log.Fatalf("error: on the fly watcher: %v", err)
		}

		run([]*watcher.Watcher{&watcher.Watcher{
			Name: "on the fly",
			//Command:   strconv.Quote(*flagCommand),
			Command:   *flagCommand,
			Match:     *flagMatch,
			Filter:    *flagFilter,
			FSWatcher: fswatcher,
			Debug:     *flagDebug,
			WithShell: true,
		}})
	} else {

		watchers, err := conf.WatchersFromPath(*flagCfg)
		if err != nil {
			log.Fatalf("error: WatchersFromPath: %v", err)
		}

		run(watchers)

	}
}
