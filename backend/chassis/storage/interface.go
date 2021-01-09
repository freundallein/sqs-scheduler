package storage

const (
	// TaskMaxRetries before setting CRITICAL_ERROR state,
	TaskMaxRetries = "10"
)

// Config - ...
type Config struct {
	DSN string
}

// TaskRepository - ...
type TaskRepository interface {
	Enqueue(*Task) error
	SelectTask() (*Task, error)
	SetTaskResult(*Task) error
	RepairStaleTasks(timeout int, batchSize int) (int, error)
}
