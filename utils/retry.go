package utils

import (
	"time"
)

var (
	DefaultRetryWaitDuration = time.Millisecond * 100
	DefaultNumberOfRetries   = 3
)

func Retry(handle func() (err error)) (err error) {
	for i := 0; i < DefaultNumberOfRetries; i++ {
		if err = handle(); err == nil {
			return
		}
		time.Sleep(DefaultRetryWaitDuration)
	}
	return
}

func RetryWithResult[Result any](handle func() (result Result, err error)) (result Result, err error) {
	for i := 0; i < DefaultNumberOfRetries; i++ {
		if result, err = handle(); err == nil {
			return
		} else {
			time.Sleep(DefaultRetryWaitDuration)
		}
	}
	return
}
