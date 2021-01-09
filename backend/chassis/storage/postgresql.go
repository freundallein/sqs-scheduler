package storage

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4/pgxpool"
)

// PGRepository - ...
type PGRepository struct {
	pool *pgxpool.Pool
}

// InitPGRepository - ...
func InitPGRepository(cfg Config) (TaskRepository, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.ConnectConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}
	return &PGRepository{
		pool: pool,
	}, nil
}

// Enqueue - ...
func (repo *PGRepository) Enqueue(task *Task) error {
	query := `insert into t_scheduler(action, payload, state) values ($1, $2, $3)`
	_, err := repo.pool.Exec(context.Background(), query, task.Action, task.Payload, "SCHEDULED")
	//ERROR: duplicate key value violates unique constraint "scheduler_object_index" (SQLSTATE 23505)
	if err != nil {
		if err.Error() == `ERROR: duplicate key value violates unique constraint "scheduler_object_index" (SQLSTATE 23505)` {
			return errors.New("duplicated task")
		}
		return err
	}
	return nil
}

// SelectTask - ...
func (repo *PGRepository) SelectTask() (*Task, error) {
	var task Task
	query := `
	with task as (
        select id, action, payload, state, attempts 
		from t_scheduler where 
			state in ('SCHEDULED', 'ERROR')
			and delayed_dt < localtimestamp
	    limit 1 for update skip locked
	) update t_scheduler
	set 
		state = 'ACQUIRED', 
		updated_dt = localtimestamp, 
		delayed_dt = null, 
		attempts = t_scheduler.attempts +1
	from task
	where t_scheduler.id = task.id
	returning t_scheduler.id, t_scheduler.action, t_scheduler.payload, t_scheduler.state, t_scheduler.attempts;
	`
	err := repo.pool.QueryRow(context.Background(), query).Scan(
		&task.ID,
		&task.Action,
		&task.Payload,
		&task.State,
		&task.Attempts,
	)
	if err != nil {
		return nil, err
	}
	task.Payload["attempt"] = strconv.Itoa(task.Attempts) // Versioning
	return &task, nil
}

// SetTaskResult - ...
func (repo *PGRepository) SetTaskResult(task *Task) error {
	var tag pgconn.CommandTag
	var err error
	if len(task.Error) == 0 {
		attempt := task.Result["attempt"]
		// delete(task.Result, "attempt")
		query := `
		update t_scheduler
		set 
		  state = 'SUCCESS', 
		  result = $3,
		  error = '{}',
		  updated_dt = localtimestamp, 
		  delayed_dt = null
		where id = $1 and state = 'ACQUIRED' and attempts = $2;
		`
		tag, err = repo.pool.Exec(context.Background(), query, task.ID, attempt, task.Result)
	} else {
		attempt := task.Error["attempt"]
		query := `
		update t_scheduler
		set 
		  state = CASE WHEN attempts < $1 THEN 'ERROR' ELSE 'CRITICAL_ERROR' END,
		  error = $2,
		  updated_dt = localtimestamp, 
		  delayed_dt = CASE WHEN attempts < $1 THEN localtimestamp + concat(5 * attempts, ' seconds')::INTERVAL ELSE null END
		where id = $3 and state = 'ACQUIRED' and attempts = $4;
		`
		tag, err = repo.pool.Exec(context.Background(), query, TaskMaxRetries, task.Error, task.ID, attempt)
	}

	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("zero rows affected")
	}
	return nil
}

// RepairStaleTasks ...
func (repo *PGRepository) RepairStaleTasks(timeout int, batchSize int) (int, error) {
	query := `
	with tasks as (
        select id, attempts 
	    from t_scheduler where state = 'ACQUIRED' and updated_dt < localtimestamp - concat($1::int, ' seconds')::INTERVAL
	    limit $2 for update skip locked
	) update t_scheduler
	set 
	  state = CASE WHEN t_scheduler.attempts + 1 < $3 THEN 'ERROR' ELSE 'CRITICAL_ERROR' END,
	  updated_dt = localtimestamp, 
	  delayed_dt = CASE WHEN t_scheduler.attempts + 1 < $3 THEN localtimestamp + concat(5 * (t_scheduler.attempts +1), ' seconds')::INTERVAL ELSE null END,
	  attempts = t_scheduler.attempts +1, 
	  error = '{"code": "0", "message": "stale task"}'
	from tasks
	where t_scheduler.id = tasks.id;
	`
	cmdTag, err := repo.pool.Exec(context.Background(), query, timeout, batchSize, TaskMaxRetries)
	if err != nil {
		return 0, err
	}
	return int(cmdTag.RowsAffected()), nil
}

// CleanOldTasks ...
func (repo *PGRepository) CleanOldTasks(expiration int) (int, error) {
	query := `
	delete from t_scheduler 
	where 
		state = 'SUCCESS' and 
		updated_dt < localtimestamp - concat($1::int, ' seconds')::INTERVAL;
	`
	cmdTag, err := repo.pool.Exec(context.Background(), query, expiration)
	if err != nil {
		return 0, err
	}
	return int(cmdTag.RowsAffected()), nil
}
