package submitter

import (
	"context"
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
			msg, err := cli.ReceiveMessage()
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "receive_failed",
					"worker": workerID,
				}).Error(err)
				continue
			}
			request := protocol.Request{}
			err = request.FromJSON(msg.Body)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "received_broken_message",
					"worker": workerID,
				}).Error(err)
				continue
			}
			if _, ok := request.Params["objectID"]; !ok {
				log.WithFields(log.Fields{
					"event":  "unsupported_message",
					"worker": workerID,
				}).Error("no objectID supported")
				continue
			}
			var action storage.Action
			switch request.Method {
			case "submit:export":
				action = storage.EXPORT
			default:
				action = storage.DUMMY
			}
			log.WithFields(log.Fields{
				"event":    "receive_message",
				"worker":   workerID,
				"action":   action,
				"objectID": request.Params["objectID"],
			}).Info(request)
			task := &storage.Task{
				Action:    action,
				Payload:   request.Params,
				CreatedDt: time.Now(),
				UpdatedDt: time.Now(),
				State:     storage.SCHEDULED,
				Result:    map[string]string{},
				Attempts:  0,
			}
			err = repo.Enqueue(task)
			err = monkey.RandomizeError(err)
			if err != nil {
				if err.Error() != "duplicated task" {
					log.WithFields(log.Fields{
						"event":    "submit_failed",
						"worker":   workerID,
						"action":   action,
						"objectID": request.Params["objectID"],
					}).Error(err)
					continue
				}
				log.WithFields(log.Fields{
					"event":    "duplicated_task",
					"worker":   workerID,
					"action":   action,
					"objectID": request.Params["objectID"],
				}).Warn("receive duplicated task")
			} else {
				log.WithFields(log.Fields{
					"event":    "submit_to_db",
					"worker":   workerID,
					"action":   action,
					"objectID": request.Params["objectID"],
				}).Info("submit task to storage")
			}

			err = cli.Acknowledge(msg)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":    "ack_message_failed",
					"worker":   workerID,
					"action":   action,
					"objectID": request.Params["objectID"],
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
