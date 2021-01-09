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
	"github.com/freundallein/scheduler/backend/worker"
)

func main() {
	appCfg, err := config.Read()

	if err != nil {
		log.WithFields(log.Fields{
			"event": "config_read_failed",
		}).Fatal(err)
	}
	log.Init("worker", appCfg)
	log.WithFields(log.Fields{
		"event": "init_service",
	}).Info("service initialized")
	// Inbound queue
	queueSrcCfg := queue.Config{
		Name:    appCfg.Worker.Queuesrc.Name,
		URL:     appCfg.Worker.Queuesrc.URL,
		Retries: appCfg.Worker.Queuesrc.Retries,

		//AWS specific
		Region:             appCfg.AWS.Region,
		CredentialsFile:    appCfg.AWS.CredentialsFile,
		CredentialsProfile: appCfg.AWS.CredentialsProfile,
	}
	queueSrcClient := queue.InitAWSQueue(queueSrcCfg)
	// Results queue
	queueDstCfg := queue.Config{
		Name:    appCfg.Worker.Queuedst.Name,
		URL:     appCfg.Worker.Queuedst.URL,
		Retries: appCfg.Worker.Queuedst.Retries,

		//AWS specific
		Region:             appCfg.AWS.Region,
		CredentialsFile:    appCfg.AWS.CredentialsFile,
		CredentialsProfile: appCfg.AWS.CredentialsProfile,
	}
	queueDstClient := queue.InitAWSQueue(queueDstCfg)
	cfg := &worker.Config{
		QueueSrc:   queueSrcClient,
		QueueDst:   queueDstClient,
		StorageDSN: appCfg.Storage.DSN,
		Workers:    appCfg.Worker.Workers,
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var group sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	worker.Run(ctx, cfg, &group)
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
