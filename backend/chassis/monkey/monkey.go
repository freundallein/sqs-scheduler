package monkey

import (
	"errors"
	"math/rand"
	"time"
)

const (
	errorChance = 0.05 // 5% error chance
)

// RandomizeError with some probability generates a random "monkey" error.
func RandomizeError(err error) error {
	if err != nil {
		return err
	}
	rand.Seed(time.Now().UnixNano())
	if rand.Float32() > errorChance {
		return nil
	}
	return errors.New("monkey error")
}
