package test

import (
	"context"
	"strconv"
	"sync"
	"time"

	log "github.com/freundallein/scheduler/backend/chassis/logging"

	"math/rand"

	"github.com/freundallein/scheduler/backend/chassis/protocol"
	"github.com/freundallein/scheduler/backend/chassis/queue"
	"github.com/jackc/pgx/v4"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// Config ...
type Config struct {
	QueueDst   queue.Client
	StorageDSN string
	Workers    int
}

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func worker(ctx context.Context, cfg *Config, workerID int, group *sync.WaitGroup) {
	cli := cfg.QueueDst
	conn, err := pgx.Connect(context.Background(), cfg.StorageDSN)
	if err != nil {
		log.WithFields(log.Fields{
			"event": "ctx_cancel",
		}).Fatal(err)
	}
	defer conn.Close(context.Background())

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
			time.Sleep(time.Millisecond * 10)
			rand.Seed(time.Now().UnixNano())
			var insertedID int
			query := `insert into t_object(data) values ($1) returning id`
			err := conn.QueryRow(context.Background(), query, map[string]string{"random": randSeq(10)}).Scan(&insertedID)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "ctx_canceled",
					"worker": workerID,
				}).Error(err)
				continue
			}

			message := protocol.Request{
				Method: "submit:export",
				Params: map[string]string{"objectID": strconv.Itoa(insertedID)},
			}
			jsonMsg, err := message.JSON()
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "serialize_failed",
					"worker": workerID,
				}).Error(err)
				continue
			}
			err = cli.SendMessage(jsonMsg)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "send_message_failed",
					"worker": workerID,
				}).Error(err)
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
