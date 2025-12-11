package sync

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRacer(t *testing.T) {
	res, err := Racer(
		func() (int, error) {
			fmt.Println("1")
			return 1, nil
		},
		func() (int, error) {
			time.Sleep(time.Second * 5)
			fmt.Println("2")
			return 2, nil
		},
		func() (int, error) {
			time.Sleep(time.Second)
			fmt.Println("3")
			return 3, nil
		},
	)

	if err != nil {
		fmt.Println("All failed:", err)
	} else {
		fmt.Println("First success:", res)
	}

	time.Sleep(time.Minute * 100)
}

func TestName(t *testing.T) {
	var err error
	var err1 error

	fmt.Println(errors.Join(err, err1))
}
