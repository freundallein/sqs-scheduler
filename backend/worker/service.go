package worker

import (
	"context"
	"sync"

	log "github.com/freundallein/scheduler/backend/chassis/logging"
	"github.com/freundallein/scheduler/backend/chassis/monkey"

	"github.com/freundallein/scheduler/backend/chassis/protocol"
	"github.com/freundallein/scheduler/backend/chassis/queue"
	"github.com/freundallein/scheduler/backend/chassis/storage"
)

// Config ...
type Config struct {
	QueueSrc   queue.Client
	QueueDst   queue.Client
	StorageDSN string
	Workers    int
}

func worker(ctx context.Context, cfg *Config, workerID int, group *sync.WaitGroup) {
	cliSrc := cfg.QueueSrc
	cliDst := cfg.QueueDst
	var handlers = map[storage.Action]func(*protocol.Request) *protocol.Response{
		storage.EXPORT: HandleExport(cfg.StorageDSN, workerID),
		storage.DUMMY:  HandleDummy,
	}
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
			msg, err := cliSrc.ReceiveMessage()
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "receive_failed",
					"worker": workerID,
				}).Error(err)
				continue
			}
			request := &protocol.Request{}
			err = request.FromJSON(msg.Body)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "receive_broken_message",
					"worker": workerID,
				}).Error(err)
				continue
			}
			var action storage.Action
			switch request.Method {
			case string(storage.EXPORT):
				action = storage.EXPORT
			default:
				action = storage.DUMMY
			}
			log.WithFields(log.Fields{
				"event":    "receive_message",
				"worker":   workerID,
				"action":   action,
				"taskID":   request.ID,
				"objectID": request.Params["objectID"],
			}).Info(request)
			handler, ok := handlers[action]
			if !ok {
				log.WithFields(log.Fields{
					"event":  "handler_not_found",
					"worker": workerID,
					"taskID": request.ID,
				}).Error(action)
				continue
			}
			response := handler(request)

			jsonMsg, err := response.JSON()
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "response_serialize_failed",
					"worker": workerID,
					"taskID": request.ID,
				}).Error(err)
				continue
			}
			err = cliDst.SendMessage(jsonMsg)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "result_send_failed",
					"worker": workerID,
					"taskID": request.ID,
				}).Error(err)
				continue
			}
			err = cliSrc.Acknowledge(msg)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "ack_message_failed",
					"worker": workerID,
					"taskID": request.ID,
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
