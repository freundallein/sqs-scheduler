package storage

import (
	"time"
)

// State - submitted record's possible states
type State string

const (
	SCHEDULED      State = "SCHEDULED"
	ACQUIRED       State = "ACQUIRED"
	SUCCESS        State = "SUCCESS"
	ERROR          State = "ERROR"
	CRITICAL_ERROR State = "CRITICAL_ERROR"
)

// Action - scheduler's possible actions
type Action string

const (
	DUMMY  Action = "DUMMY"
	EXPORT Action = "EXPORT"
)

// Task - ...
type Task struct {
	ID        int
	Action    Action
	Payload   map[string]string
	CreatedDt time.Time
	UpdatedDt time.Time
	State     State
	Result    map[string]string
	Error     map[string]string
	Attempts  int
}
