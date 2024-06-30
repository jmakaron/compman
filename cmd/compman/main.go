package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmakaron/compman/internal/app/compman"
	"github.com/jmakaron/compman/internal/app/compman/config"
	"github.com/jmakaron/compman/pkg/logger"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfgPath := flag.String("cfg", "", "json config file path")
	debug := flag.Bool("debug", false, "run in debug mode")
	flag.Parse()
	if flag.Parsed() {
		if cfgPath == nil || len(*cfgPath) == 0 {
			flag.Usage()
			return 1
		}
	}
	log, err := logger.New(*debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger, %+v\n", err)
		return 1
	}
	var cfg *config.AppConfig
	cfg, err = config.ParseConfigFile(*cfgPath)
	if err != nil {
		log.Error(fmt.Sprintf("failed to parse config, %+v", err))
		return 1
	}
	cfg.HttpCfg.Debug = *debug

	c := compman.New(log)
	log.Info("initializing")
	if err = c.Init(cfg); err != nil {
		log.Error(fmt.Sprintf("failed to initialize component, %+v", err))
		return 1
	}
	log.Info("initialized")
	log.Info("starting")
	if err = c.Start(); err != nil {
		log.Error(fmt.Sprintf("failed to start component, %+v", err))
		return 1
	}
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGPIPE)
	log.Info("started")
	for {
		sig := <-signalCh
		if sig == syscall.SIGPIPE {
			continue
		}
		log.Info(fmt.Sprintf("caught signal: %+v", sig))
		if sig == syscall.SIGHUP {
			// should handle cfg reload and continue here
			continue
		}
		log.Info("stopping")
		break
	}
	c.Stop()
	log.Info("stopped")
	return 0
}
