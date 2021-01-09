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
Exports sql database records from `t_object` to `t_exported_object`.  
You can write your own worker's handler and define `action` in all scheduler's packages

## Installation
- install `aws cli` and set `.credentials` for SQS
- create SQS with terraform - ```make terraform```
- build docker images with ```make dockerbuild```
- start Postgres database with docker-compose
- run `init.sql` queries in Postgres database to create all tables
- start scheduler locally with ```make up```
- stop scheduler with ```make down```

## Typical workflow
- start inserting random records to t_object and enqueue "export" tasks to SQS with ```make test``` command
- submitter pulls tasks from SQS and persists them in PG storage as `SCHEDULED` tasks
- scheduler acquires next task (set `ACQUIRED` state) and enqueues it to SQS
- worker pulls acquired task, does export from t_object to t_exported_object and sends results to SQS
- resulter pulls results and persists them in PG storage, changing `ACQUIRED` state to `SUCCESS`/`ERROR`
- Each task has 10 attempts, then it forced to `CRITICAL_ERROR` and processing of that task stops.
- supervisor fixes `ACQUIRED` state to `ERROR` if `ACQUIRED` is longer than `staleTimeout` seconds
- all operation should be idempotent and retryable (and they are)


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
- [x] supervisor's db cleaner
- [ ] task priority
- [ ] multistage tasks
- [ ] rabbitmq/kafka integration
- [ ] http api for enqueue and state polling
