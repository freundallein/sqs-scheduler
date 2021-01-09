package worker

import (
	"context"

	log "github.com/freundallein/scheduler/backend/chassis/logging"

	"github.com/freundallein/scheduler/backend/chassis/monkey"
	"github.com/freundallein/scheduler/backend/chassis/protocol"
	"github.com/freundallein/scheduler/backend/chassis/storage"
	"github.com/jackc/pgx/v4"
)

// HandleDummy - ...
func HandleDummy(request *protocol.Request) *protocol.Response {
	log.Info("processing_object: id=%s attempt=%s", request.Params["objectID"], request.Params["attempt"])
	response := &protocol.Response{
		ID: request.ID,
	}
	err := monkey.RandomizeError(nil)
	if err != nil {
		response.Error = map[string]string{"code": "5050", "message": "random error", "attempt": request.Params["attempt"]}
	} else {
		response.Result = map[string]string{"result": "success", "attempt": request.Params["attempt"]}
	}
	return response
}

// HandleExport - ...
func HandleExport(storageDSN string, workerID int) func(request *protocol.Request) *protocol.Response {
	return func(request *protocol.Request) *protocol.Response {
		conn, err := pgx.Connect(context.Background(), storageDSN)
		if err != nil {
			log.WithFields(log.Fields{
				"event":  "storage_conn_failed",
				"worker": workerID,
			}).Error(err)
		}
		defer conn.Close(context.Background())

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
				"attempt":  request.Params["attempt"],
			}).Error(err)
			response.Error = map[string]string{"code": "1", "message": err.Error(), "attempt": request.Params["attempt"]}
			return response
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
				"attempt":  request.Params["attempt"],
			}).Error(err)
			response.Error = map[string]string{"code": "2", "message": err.Error(), "attempt": request.Params["attempt"]}
			return response
		}
		response.Result = map[string]string{"result": "success", "attempt": request.Params["attempt"]}
		log.WithFields(log.Fields{
			"event":    "object_processed",
			"worker":   workerID,
			"taskID":   request.ID,
			"objectID": request.Params["objectID"],
			"attempt":  request.Params["attempt"],
		}).Info("successfully export object")
		return response
	}
}
