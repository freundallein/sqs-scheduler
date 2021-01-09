package storage

import (
	"time"
)

// Object - simple representation of business-logic objects
type Object struct {
	ID        int
	Data      map[string]string
	CreatedDt time.Time
}
