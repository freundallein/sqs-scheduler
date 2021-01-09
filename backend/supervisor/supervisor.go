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
	Expiration      int
}

func worker(ctx context.Context, cfg *Config, workerID int, group *sync.WaitGroup) {
	group.Add(1)
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

func dbCleaner(ctx context.Context, cfg *Config, group *sync.WaitGroup) {
	log.WithFields(log.Fields{
		"event": "start_db_cleaner",
	}).Info("starting db cleaner with ", cfg.Expiration, "s expiration time")
	group.Add(1)
	repo := cfg.Repository
	for {
		select {
		case <-ctx.Done():
			log.WithFields(log.Fields{
				"event":  "ctx_canceled",
				"worker": "db_cleaner",
			}).Info("exit goroutine")
			group.Done()
			return
		case <-time.After(time.Second * 5):
			cleaned, err := repo.CleanOldTasks(cfg.Expiration)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "clean_table_failed",
					"worker": "db_cleaner",
				}).Error(err)
			}
			log.WithFields(log.Fields{
				"event":  "clean_table",
				"worker": "db_cleaner",
			}).Info("cleaned rows:", cleaned)
		}
	}
}

// Run ...
func Run(ctx context.Context, cfg *Config, group *sync.WaitGroup) {
	log.WithFields(log.Fields{
		"event": "start_service",
	}).Info("starting ", cfg.Workers, " workers")
	go dbCleaner(ctx, cfg, group)
	for wrk := 1; wrk <= cfg.Workers; wrk++ {
		go worker(ctx, cfg, wrk, group)
	}
}
