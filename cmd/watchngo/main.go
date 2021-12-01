package main

import (
	"log"
	"os"

	"github.com/Leryan/watchngo/pkg"
	"github.com/go-ini/ini"

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

	var watchers []*pkg.Watcher

	if *flagCommand != "" && *flagMatch != "" {
		executor, err := pkg.ExecutorFromName(*flagExecutor)
		if err != nil {
			log.Fatal(err)
		}
		w, err := pkg.NewWatcher(
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

		watchers = append(watchers, w)
	} else {
		cfg, err := ini.Load(*flagCfg)
		if err != nil {
			log.Fatalf("conf: from path: %s: %v", *flagCfg, err)
		}

		watchers, err = pkg.WatchersFromConf(cfg, logger, pkg.ExecutorFromName)
		if err != nil {
			log.Fatalf("error: WatchersFromConf: %v", err)
		}
	}

	pkg.RunForever(watchers)
}
