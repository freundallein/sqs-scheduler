package supervisor

import (
	"context"
	"sync"
	"time"

	log "github.com/freundallein/scheduler/backend/chassis/logging"

	"github.com/freundallein/scheduler/backend/chassis/monkey"
	"github.com/freundallein/scheduler/backend/chassis/storage"
)

// Config ...
type Config struct {
	Repository      storage.TaskRepository
	Workers         int
	StaleTimeout    int
	RepairBatchSize int
}

func worker(ctx context.Context, cfg *Config, workerID int, group *sync.WaitGroup) {
	repo := cfg.Repository

	for {
		select {
		case <-ctx.Done():
			log.WithFields(log.Fields{
				"event":  "ctx_canceled",
				"worker": workerID,
			}).Info("exit goroutine")
			group.Done()
			return
		case <-time.After(time.Second * 5):
			repaired, err := repo.RepairStaleTasks(cfg.StaleTimeout, cfg.RepairBatchSize)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "stale_task_repair_failed",
					"worker": workerID,
				}).Error(err)
			}
			log.WithFields(log.Fields{
				"event":  "stale_task_repair",
				"worker": workerID,
			}).Info("select and repair stale tasks:", repaired)
		}
	}
}

// Run ...
func Run(ctx context.Context, cfg *Config, group *sync.WaitGroup) {
	log.WithFields(log.Fields{
		"event": "start_service",
	}).Info("starting ", cfg.Workers, " workers")
	for wrk := 1; wrk <= cfg.Workers; wrk++ {
		group.Add(1)
		go worker(ctx, cfg, wrk, group)
	}
}
