package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/freundallein/scheduler/backend/chassis/logging"

	"github.com/freundallein/scheduler/backend/chassis/config"
	"github.com/freundallein/scheduler/backend/chassis/queue"
	"github.com/freundallein/scheduler/backend/test"
)

func main() {
	appCfg, err := config.Read()
	if err != nil {
		log.WithFields(log.Fields{
			"event": "config_read_failed",
		}).Fatal(err)
	}
	log.Init("test", appCfg)
	log.WithFields(log.Fields{
		"event": "init_service",
	}).Info("test service initialized")
	queueCfg := queue.Config{
		// Use submitter's cfg
		Name:    appCfg.Submitter.Queuesrc.Name,
		URL:     appCfg.Submitter.Queuesrc.URL,
		Retries: appCfg.Submitter.Queuesrc.Retries,

		//AWS specific
		Region:             appCfg.AWS.Region,
		CredentialsFile:    appCfg.AWS.CredentialsFile,
		CredentialsProfile: appCfg.AWS.CredentialsProfile,
	}
	queueClient := queue.InitAWSQueue(queueCfg)

	cfg := &test.Config{
		QueueDst:   queueClient,
		StorageDSN: appCfg.Storage.DSN,
		Workers:    appCfg.Submitter.Workers, // Use submitter's cfg
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var group sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	test.Run(ctx, cfg, &group)
	<-done
	log.WithFields(log.Fields{
		"event": "ctx_cancel",
	}).Info("received syscall")
	cancel()
	group.Wait()
}
