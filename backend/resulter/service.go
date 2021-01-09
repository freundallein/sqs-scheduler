package resulter

import (
	"context"
	"strconv"
	"sync"

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
			msg, err := cli.ReceiveMessage()
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "receive_failed",
					"worker": workerID,
				}).Error(err)
				continue
			}
			response := protocol.Response{}
			err = response.FromJSON(msg.Body)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "received_broken_message",
					"worker": workerID,
				}).Info(err)
				continue
			}
			log.WithFields(log.Fields{
				"event":  "receive_result",
				"worker": workerID,
			}).Info("receive results for task")

			taskID, err := strconv.Atoi(response.ID)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "received_broken_task_id",
					"worker": workerID,
					"taskID": response.ID,
				}).Error(err)
				continue
			}
			task := &storage.Task{
				ID:     taskID,
				Result: response.Result,
				Error:  response.Error,
			}
			err = repo.SetTaskResult(task)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "result_error",
					"worker": workerID,
					"taskID": response.ID,
				}).Error(err)
			} else {
				log.WithFields(log.Fields{
					"event":  "result_to_storage",
					"worker": workerID,
					"taskID": response.ID,
				}).Info("save result to storage")
			}
			err = cli.Acknowledge(msg)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "ack_message_failed",
					"worker": workerID,
					"taskID": response.ID,
				}).Error(err)
				continue
			}
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
