package sync

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
)

type Result[T any] struct {
	Val T
	Err error
}

type WaitGroup struct {
	sync.WaitGroup
	sync.Mutex
	err []error
}

func (wg *WaitGroup) setErr(err error) {
	wg.Lock()
	defer wg.Unlock()
	wg.err = append(wg.err, err)
}

func (wg *WaitGroup) Go(f func() error) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				wg.setErr(fmt.Errorf("panic: %v\n stack:%s", r, debug.Stack()))
			}
		}()
		if err := f(); err != nil {
			wg.setErr(err)
		}
	}()
}

func (wg *WaitGroup) Wait() error {
	wg.WaitGroup.Wait()
	wg.Mutex.Lock()
	defer wg.Mutex.Unlock()
	if len(wg.err) == 0 {
		return nil
	}
	return fmt.Errorf("%v", wg.err)
}

type WaitGroupWithResult[T any] struct {
	wg      sync.WaitGroup
	mutex   sync.Mutex
	results []T
	errs    []error
}

func (wg *WaitGroupWithResult[T]) addResult(result T) {
	wg.mutex.Lock()
	defer wg.mutex.Unlock()
	wg.results = append(wg.results, result)
}

func (wg *WaitGroupWithResult[T]) addErr(err error) {
	wg.mutex.Lock()
	defer wg.mutex.Unlock()
	wg.errs = append(wg.errs, err)
}

func (wg *WaitGroupWithResult[T]) Go(f func() (T, error)) {
	wg.wg.Add(1)
	go func() {
		defer wg.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				wg.addErr(fmt.Errorf("panic: %v\n stack:%s", r, debug.Stack()))
			}
		}()
		if result, err := f(); err != nil {
			wg.addErr(err)
		} else {
			wg.addResult(result)
		}
	}()
}

func (wg *WaitGroupWithResult[T]) Wait() {
	wg.wg.Wait()
}

func (wg *WaitGroupWithResult[T]) GetResults() []T {
	wg.mutex.Lock()
	defer wg.mutex.Unlock()
	return wg.results
}

func (wg *WaitGroupWithResult[T]) Err() error {
	wg.mutex.Lock()
	defer wg.mutex.Unlock()
	return errors.Join(wg.errs...)
}
