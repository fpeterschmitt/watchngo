package main

import (
	"github.com/Leryan/watchngo/pkg/runner"
	"log"
	"os"

	"github.com/Leryan/watchngo/pkg/conf"
	"github.com/Leryan/watchngo/pkg/watcher"

	"flag"
)

func main() {
	flagCfg := flag.String("conf", "watchngo.ini", "configuration file path")

	flagMatch := flag.String("match", "", "file or directory to watch")
	flagFilter := flag.String("filter", "", "filter as a regex supported by golang")
	flagCommand := flag.String("command", "", "command to run. see configuration example for supported variables")
	flagExecutor := flag.String("executor", "unixshell", "executors: unixshell, raw, stdout")
	flagDebug := flag.Bool("debug", false, "debug")

	flag.Parse()

	logger := log.New(os.Stderr, "", log.LstdFlags)

	log.SetOutput(os.Stderr)

	if *flagCommand != "" && *flagMatch != "" {
		executor, err := conf.ExecutorFrom(*flagExecutor)
		if err != nil {
			log.Fatal(err)
		}
		w, err := watcher.NewWatcher(
			"on the fly",
			*flagMatch,
			*flagFilter,
			*flagCommand,
			executor,
			*flagDebug,
			logger,
		)

		if err != nil {
			log.Fatalf("error: on the fly: %v", err)
		}

		runner.Run([]*watcher.Watcher{w})
	} else {

		watchers, err := conf.WatchersFromPath(*flagCfg, logger)
		if err != nil {
			log.Fatalf("error: WatchersFromPath: %v", err)
		}

		runner.Run(watchers)
	}
}
