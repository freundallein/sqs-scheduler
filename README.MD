## scheduler

Simple distributed task scheduler.

## Technlogies
- Golang
- Postgresql
- Amazon SQS
- JSON-RPC 2.0

## Design
![alt text](https://raw.githubusercontent.com/freundallein/scheduler/master/design/scheduler.png "Scheduler")

## Example worker
Exports database records from t_object to t_exported_object.  
You can write your own worker's handler and define `action` in all scheduler's packages

## Typical workflow
- create SQS with terraform - ```make terraform```
- deploy scheduler locally with ```make up```
- start inserting random records to t_object and enqueue "export" tasks to SQS with ```make test``` command
- submitter pulls tasks from SQS and persist them in PG storage as `SCHEDULED` tasks
- scheduler acquires next task (set `ACQUIRED` state) and enqueue it to SQS
- worker pulls acquired task, do export from t_object to t_exported_object and send results to SQS
- resulter pulls results and persist them in PG storage, changing `ACQUIRED` state to `SUCCESS`/`ERROR`
- Each task has 10 attempts, then it forced to `CRITICAL_ERROR` and processing of that task stops
- supervisor fix `ACQUIRED` state to `ERROR` if `ACQUIRED` is longer than `staleTimeout` seconds
- all operation should be idempotent and retryable (and they are)
- stop scheduler with ```make down```

Features:  
- [x] multiworkers per instance  
- [x] graceful shutdown  
- [x] logfmt  
- [x] unified configuration
- [x] random errors
- [x] supervising
- [x] crit error on max atempts
- [x] exponential delay on error -> delayed_dt
- [x] index sql table -> delayed_dt
- [x] containerization
- [x] monitoring (prometheus)

- [ ] task priority
- [ ] multistage tasks
- [ ] rabbitmq/kafka integration
- [ ] http api for enqueue and state polling
