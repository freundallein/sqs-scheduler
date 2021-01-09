package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/freundallein/scheduler/backend/chassis/logging"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/freundallein/scheduler/backend/chassis/config"
	"github.com/freundallein/scheduler/backend/chassis/storage"
	"github.com/freundallein/scheduler/backend/supervisor"
)

func main() {
	appCfg, err := config.Read()

	if err != nil {
		log.WithFields(log.Fields{
			"event": "config_read_failed",
		}).Fatal(err)
	}
	log.Init("supervisor", appCfg)
	log.WithFields(log.Fields{
		"event": "init_service",
	}).Info("service initialized")
	repoCfg := storage.Config{
		DSN: appCfg.Storage.DSN,
	}
	repo, err := storage.InitPGRepository(repoCfg)
	if err != nil {
		log.WithFields(log.Fields{
			"event": "init_storage_failed",
		}).Fatal(err)
	}
	cfg := &supervisor.Config{
		Repository:      repo,
		Workers:         appCfg.Supervisor.Workers,
		StaleTimeout:    appCfg.Supervisor.StaleTimeout,
		RepairBatchSize: appCfg.Supervisor.RepairBatchSize,
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var group sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	supervisor.Run(ctx, cfg, &group)
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":2112",
		Handler: router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen: %s\n", err)
		}
	}()
	<-done
	log.WithFields(log.Fields{
		"event": "ctx_cancel",
	}).Info("received syscall")
	cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server Shutdown Failed:%+v", err)
	}
	group.Wait()
}
