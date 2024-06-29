package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmakaron/compman/internal/app/compman/config"
	httpsrv "github.com/jmakaron/compman/internal/pkg/http"
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
	var cfg *config.Config
	cfg, err = config.ParseConfig(*cfgPath)
	if err != nil {
		log.Error(fmt.Sprintf("failed to parse config, %+v", err))
		return 1
	}

	tmpHandler := func(w http.ResponseWriter, r *http.Request) error {
		ids := httpsrv.GetIdList(r)
		log.Debug(fmt.Sprintf("logging request: %s %s for %+v", r.URL, r.Method, ids))
		w.WriteHeader(http.StatusOK)
		return nil
	}

	srv := &httpsrv.HTTPService{}
	rspec := httpsrv.RouterSpec{
		"company/": {http.MethodGet: tmpHandler},
		"company/{id1}": {
			http.MethodGet:    tmpHandler,
			http.MethodPost:   tmpHandler,
			http.MethodPut:    tmpHandler,
			http.MethodDelete: tmpHandler,
		},
	}

	log.Info("initializing")
	if err = srv.Init(cfg.HttpCfg, rspec, log); err != nil {
		log.Error(fmt.Sprintf("failed to initialize http service component, %+v", err))
		return 1
	}
	log.Info("initialized")
	log.Info("starting")
	if err = srv.Start(); err != nil {
		log.Error(fmt.Sprintf("failed to start http service component, %+v", err))
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
	if err = srv.Stop(); err != nil {
		log.Error(fmt.Sprintf("failed to stop http service component, %+v", err))
		return 1
	}

	log.Info("stopped")
	return 0
}
