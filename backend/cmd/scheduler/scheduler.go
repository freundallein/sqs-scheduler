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
	"github.com/freundallein/scheduler/backend/chassis/queue"
	"github.com/freundallein/scheduler/backend/chassis/storage"
	"github.com/freundallein/scheduler/backend/scheduler"
)

func main() {
	appCfg, err := config.Read()

	if err != nil {
		log.WithFields(log.Fields{
			"event": "config_read_failed",
		}).Fatal(err)
	}
	log.Init("scheduler", appCfg)
	log.WithFields(log.Fields{
		"event": "init_service",
	}).Info("service initialized")
	queueCfg := queue.Config{
		Name:    appCfg.Scheduler.Queuedst.Name,
		URL:     appCfg.Scheduler.Queuedst.URL,
		Retries: appCfg.Scheduler.Queuedst.Retries,

		//AWS specific
		Region:             appCfg.AWS.Region,
		CredentialsFile:    appCfg.AWS.CredentialsFile,
		CredentialsProfile: appCfg.AWS.CredentialsProfile,
	}
	queueClient := queue.InitAWSQueue(queueCfg)
	repoCfg := storage.Config{
		DSN: appCfg.Storage.DSN,
	}
	repo, err := storage.InitPGRepository(repoCfg)
	if err != nil {
		log.WithFields(log.Fields{
			"event": "init_storage_failed",
		}).Fatal(err)
	}
	cfg := &scheduler.Config{
		Queue:      queueClient,
		Repository: repo,
		Workers:    appCfg.Scheduler.Workers,
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var group sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	scheduler.Run(ctx, cfg, &group)

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
