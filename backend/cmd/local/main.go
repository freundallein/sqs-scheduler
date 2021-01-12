package main

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	log "github.com/freundallein/scheduler/backend/chassis/logging"
	"github.com/freundallein/scheduler/backend/chassis/protocol"
	"github.com/jackc/pgx/v4"

	"github.com/freundallein/scheduler/backend/chassis/config"
	"github.com/freundallein/scheduler/backend/chassis/monkey"
	"github.com/freundallein/scheduler/backend/chassis/storage"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
func insert(workerID int, inbound chan *protocol.Request) {
	conn, err := pgx.Connect(context.Background(), "postgresql://scheduler:scheduler@pg-db:5433/scheduler")
	if err != nil {
		log.WithFields(log.Fields{
			"event":  "ctx_cancel",
			"module": "inserter",
		}).Fatal(err)
	}
	defer conn.Close(context.Background())
	for {
		time.Sleep(time.Millisecond * 900)
		rand.Seed(time.Now().UnixNano())
		var insertedID int
		query := `insert into t_object(data) values ($1) returning id`
		err := conn.QueryRow(context.Background(), query, map[string]string{"random": randSeq(10)}).Scan(&insertedID)
		if err != nil {
			log.WithFields(log.Fields{
				"event":  "ctx_canceled",
				"module": "inserter",
				"worker": workerID,
			}).Error(err)
			continue
		}
		message := protocol.Request{
			Method: "submit:export",
			Params: map[string]string{"objectID": strconv.Itoa(insertedID)},
		}
		inbound <- &message
	}
}
func process(workerID int, outbound chan *protocol.Request, results chan *protocol.Response) {
	conn, err := pgx.Connect(context.Background(), "postgresql://scheduler:scheduler@pg-db:5433/scheduler")
	if err != nil {
		log.WithFields(log.Fields{
			"event":  "storage_conn_failed",
			"worker": workerID,
		}).Error(err)
	}
	defer conn.Close(context.Background())
	for {
		select {
		case request := <-outbound:
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
				"module":   "processor",
				"objectID": request.Params["objectID"],
			}).Debug(request)
			response := &protocol.Response{
				ID: request.ID,
			}
			var object storage.Object
			query := `select id, data from t_object where id=$1`
			err = conn.QueryRow(context.Background(), query, request.Params["objectID"]).Scan(&object.ID, &object.Data)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":    "select_object_failed",
					"worker":   workerID,
					"taskID":   request.ID,
					"objectID": request.Params["objectID"],
					"module":   "processor",
					"attempt":  request.Params["attempt"],
				}).Error(err)
				response.Error = map[string]string{"code": "1", "message": err.Error(), "attempt": request.Params["attempt"]}
				results <- response
				continue
			}
			var returnedID int
			query = `insert into t_exported_object(id, data) values ($1, $2) returning id`
			err = conn.QueryRow(context.Background(), query, object.ID, object.Data).Scan(&returnedID)
			err = monkey.RandomizeError(err)
			if err != nil && err.Error() != `ERROR: duplicate key value violates unique constraint "t_exported_object_pkey" (SQLSTATE 23505)` {
				log.WithFields(log.Fields{
					"event":    "insert_object_failed",
					"worker":   workerID,
					"taskID":   request.ID,
					"objectID": request.Params["objectID"],
					"module":   "processor",
					"attempt":  request.Params["attempt"],
				}).Error(err)
				response.Error = map[string]string{"code": "2", "message": err.Error(), "attempt": request.Params["attempt"]}
				results <- response
				continue
			}
			response.Result = map[string]string{"result": "success", "attempt": request.Params["attempt"]}
			log.WithFields(log.Fields{
				"event":    "object_processed",
				"worker":   workerID,
				"taskID":   request.ID,
				"module":   "processor",
				"objectID": request.Params["objectID"],
				"attempt":  request.Params["attempt"],
			}).Debug("successfully export object")
			results <- response
		}
	}
}
func main() {
	appCfg, err := config.Read()

	if err != nil {
		log.WithFields(log.Fields{
			"event": "config_read_failed",
		}).Fatal(err)
	}
	log.Init("local", appCfg)
	log.WithFields(log.Fields{
		"event": "init_service",
	}).Debug("service initialized")

	repoCfg := storage.Config{
		DSN: appCfg.Storage.DSN + "?pool_max_conns=5000",
	}
	repo, err := storage.InitPGRepository(repoCfg)
	if err != nil {
		log.WithFields(log.Fields{
			"event": "init_storage_failed",
		}).Fatal(err)
	}
	inbound := make(chan *protocol.Request, 120000)
	outbound := make(chan *protocol.Request, 120000)
	results := make(chan *protocol.Response, 120000)
	go supervisor(repo, 1)

	for i := 0; i < 400; i++ {
		// go insert(i, inbound)
	}
	for i := 0; i < 300; i++ {
		// go submit(repo, i, inbound)
	}

	for i := 0; i < 150; i++ {
		go schedule(repo, i, outbound)
	}
	for i := 0; i < 400; i++ {
		go process(i, outbound, results)
	}
	for i := 0; i < 150; i++ {
		go result(repo, i, results)
	}
	for {
		select {
		case <-time.After(time.Second * 5):
			log.Info("-----------------------")
			log.Info("INBOUND ", len(inbound))
			log.Info("OUTBOUND ", len(outbound))
			log.Info("RESULTS ", len(results))
		}
	}
}

func submit(repo storage.TaskRepository, workerID int, inbound chan *protocol.Request) {
	for {
		select {
		case request := <-inbound:
			if _, ok := request.Params["objectID"]; !ok {
				log.WithFields(log.Fields{
					"event":  "unsupported_message",
					"worker": workerID,
					"module": "submitter",
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
				"module":   "submitter",
				"worker":   workerID,
				"action":   action,
				"objectID": request.Params["objectID"],
			}).Debug(request)
			task := &storage.Task{
				Action:    action,
				Payload:   request.Params,
				CreatedDt: time.Now(),
				UpdatedDt: time.Now(),
				State:     storage.SCHEDULED,
				Result:    map[string]string{},
				Attempts:  0,
			}
			err := repo.Enqueue(task)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "submit_failed",
					"module": "submitter",
					"worker": workerID,
				}).Error(err)
			}
		}
	}
}

func schedule(repo storage.TaskRepository, workerID int, outbound chan *protocol.Request) {
	for {
		task, err := repo.SelectTask()
		err = monkey.RandomizeError(err)
		if err != nil {
			if err.Error() == "no rows in result set" {
				log.WithFields(log.Fields{
					"event":  "select_task_failed",
					"module": "scheduler",
					"worker": workerID,
				}).Debug(err)
				time.Sleep(time.Second * 5)
			} else {
				log.WithFields(log.Fields{
					"event":  "select_task_failed",
					"module": "scheduler",
					"worker": workerID,
				}).Error(err)
			}
			continue
		}
		log.WithFields(log.Fields{
			"event":    "task_acquire",
			"module":   "scheduler",
			"worker":   workerID,
			"taskID":   task.ID,
			"action":   task.Action,
			"objectID": task.Payload["objectID"],
		}).Debug("acquire task")
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
		outbound <- &message
	}
}

func result(repo storage.TaskRepository, workerID int, results chan *protocol.Response) {
	for {
		select {
		case response := <-results:
			log.WithFields(log.Fields{
				"event":  "receive_result",
				"module": "resulter",
				"worker": workerID,
			}).Debug("receive results for task")

			taskID, err := strconv.Atoi(response.ID)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "received_broken_task_id",
					"worker": workerID,
					"module": "resulter",
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
					"event":  "set_task_failed",
					"module": "resulter",
					"worker": workerID,
					"taskID": response.ID,
				}).Error(err)
				continue
			}
		}
	}
}

func supervisor(repo storage.TaskRepository, workerID int) {
	for {
		select {
		case <-time.After(time.Second * 1):
			repaired, err := repo.RepairStaleTasks(60, 10)
			err = monkey.RandomizeError(err)
			if err != nil {
				log.WithFields(log.Fields{
					"event":  "stale_task_repair_failed",
					"module": "supervisor",
					"worker": workerID,
				}).Error(err)
			}
			log.WithFields(log.Fields{
				"event":  "stale_task_repair",
				"module": "supervisor",
				"worker": workerID,
			}).Debug("select and repair stale tasks:", repaired)
		}
	}
}
