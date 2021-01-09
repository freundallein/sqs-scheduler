package scheduler

import (
	"context"
	"strconv"
	"sync"
	"time"

	log "github.com/freundallein/scheduler/backend/chassis/logging"

	"github.com/freundallein/scheduler/backend/chassis/monkey"
	"github.com/freundallein/scheduler/backend/chassis/protocol"
	"github.com/freundallein/scheduler/backend/chassis/queue"
	"github.com/freundallein/scheduler/backend/chassis/storage"
)

// Config ...
type Config struct {
	Queue      queue.Client
	Repository storage.TaskRepository
	Workers    int
}

func worker(ctx context.Context, cfg *Config, workerID int, group *sync.WaitGroup) {
	cli := cfg.Queue
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
		default:
			task, err := repo.SelectTask()
			err = monkey.RandomizeError(err)
			if err != nil {
				if err.Error() == "no rows in result set" {
					log.WithFields(log.Fields{
						"event":  "select_task_failed",
						"worker": workerID,
					}).Info(err)
					time.Sleep(time.Second * 5)
				} else {
					log.WithFields(log.Fields{
						"event":  "select_task_failed",
						"worker": workerID,
					}).Error(err)
				}
				continue
			}
			log.WithFields(log.Fields{
				"event":    "task_acquire",
				"worker":   workerID,
				"taskID":   task.ID,
				"action":   task.Action,
				"objectID": task.Payload["objectID"],
			}).Info("acquire task")
			var action string
			switch task.Action {
			case storage.EXPORT:
				action = string(storage.EXPORT)
			default:
				action = string(storage.DUMMY)
			}
			message := protocol.Request{
				Method: action,
				Params: task.Payload,
				ID:     strconv.Itoa(task.ID),
			}
			jsonMsg, err := message.JSON()
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "request_serialize_failed",
					"worker": workerID,
				}).Error(err)
				continue
			}
			err = cli.SendMessage(jsonMsg)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "request_send_failed",
					"worker": workerID,
				}).Error(err)
				continue
			}
			log.WithFields(log.Fields{
				"event":    "send_task",
				"worker":   workerID,
				"taskID":   task.ID,
				"action":   task.Action,
				"objectID": task.Payload["objectID"],
			}).Info("send task to workers")
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
