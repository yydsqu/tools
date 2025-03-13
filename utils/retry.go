package utils

import (
	"time"
)

var (
	DefaultRetryWaitDuration = time.Millisecond * 100
	DefaultAttempts          = 3
)

type RetryableFunc func() error

type RetryableFuncWithResult[T any] func() (T, error)

func Retry(handle RetryableFunc) (err error) {
	for i := 0; i < DefaultAttempts; i++ {
		if err = handle(); err == nil {
			return
		}
		time.Sleep(DefaultRetryWaitDuration)
	}
	return
}

func RetryWithResult[T any](handle RetryableFuncWithResult[T]) (empty T, err error) {
	for i := 0; i < DefaultAttempts; i++ {
		if empty, err = handle(); err == nil {
			return
		} else {
			time.Sleep(DefaultRetryWaitDuration)
		}
	}
	return
}
