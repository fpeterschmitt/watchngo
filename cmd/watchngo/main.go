package main

import (
	"log"
	"os"

	"github.com/Leryan/watchngo/pkg"
	"github.com/go-ini/ini"

	"flag"
)

func main() {
	flagConf := flag.String("conf", "watchngo.ini", "configuration file path")
	flagMatch := flag.String(pkg.CfgMatch, "", "file or directory to watch. defaults to current directory")
	flagFilter := flag.String(pkg.CfgFilter, "", "filter as a regex supported by golang")
	flagCommand := flag.String(pkg.CfgCommand, "", "command to run. see configuration example for supported variables")
	flagExecutor := flag.String(pkg.CfgExecutor, pkg.ExecutorUnixShell, "executors: unixshell, raw, stdout")
	flagDebug := flag.Bool(pkg.CfgDebug, false, "debug")
	flagSilent := flag.Bool(pkg.CfgSilent, false, "silence any output originating from watchngo. overrides -debug.")
	flag.Parse()

	var cfg *ini.File
	if *flagCommand != "" {
		cfg = pkg.BuildIniCfgFrom(pkg.Cfg{
			Name:            "cli",
			Match:           *flagMatch,
			Filter:          *flagFilter,
			CommandTemplate: *flagCommand,
			ExecutorName:    *flagExecutor,
			Debug:           *flagDebug,
			Silent:          *flagSilent,
		})
	} else {
		var err error
		if cfg, err = ini.Load(*flagConf); err != nil {
			log.Fatalf("conf: from path: %s: %v", *flagConf, err)
		}
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)
	log.SetOutput(os.Stderr)

	watchers, err := pkg.WatchersFromConf(cfg, logger, pkg.ExecutorFromName)
	if err != nil {
		log.Fatalf("error: WatchersFromConf: %v", err)
	}

	pkg.RunForever(watchers)
}
