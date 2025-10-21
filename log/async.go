package log

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const backupTimeFormat = "2006_01_02_15"

type TimeTicker struct {
	stop chan struct{}
	C    <-chan time.Time
}

func NewTimeTicker(rotateHours uint) *TimeTicker {
	rotateHours = cmp.Or(rotateHours, 24)
	ch := make(chan time.Time, 1)
	tt := TimeTicker{
		stop: make(chan struct{}),
		C:    ch,
	}
	tt.startTicker(ch, rotateHours)
	return &tt
}

func (tt *TimeTicker) Stop() {
	select {
	case <-tt.stop:
	default:
		close(tt.stop)
	}
}

func (tt *TimeTicker) startTicker(ch chan time.Time, rotateHours uint) {
	go func() {
		timer := time.NewTimer(0)
		defer timer.Stop()
		for {
			select {
			case t := <-timer.C:
				ch <- t
				now := time.Now()
				next := nextAlignedTime(now, rotateHours)
				timer.Reset(next.Sub(now))
			case <-tt.stop:
				return
			}
		}
	}()
}

func nextAlignedTime(now time.Time, rotateHours uint) time.Time {
	next := now.Truncate(time.Hour).Add(time.Hour)
	rh := int(rotateHours)
	if m := next.Hour() % rh; m != 0 {
		next = next.Add(time.Duration(rh-m) * time.Hour)
	}
	return time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), 0, 0, 0, next.Location())
}

type AsyncFileWriter struct {
	filePath string
	fd       *os.File

	wg         sync.WaitGroup
	started    int32
	buf        chan []byte
	stop       chan struct{}
	timeTicker *TimeTicker

	rotateHours uint
	maxBackups  int
}

func NewAsyncFileWriter(filePath string, maxBytesSize int64, maxBackups int, rotateHours uint) (*AsyncFileWriter, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("get file path of logger error. filePath=%s, err=%s", filePath, err)
	}
	if _, err = os.Stat(filepath.Dir(absFilePath)); os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(absFilePath), os.ModePerm); err != nil {
			return nil, fmt.Errorf("mkdir error path=%s, err=%s", filepath.Dir(absFilePath), err)
		}
	}

	// 整点切换目录
	asyncFileWriter := &AsyncFileWriter{
		filePath:    absFilePath,
		buf:         make(chan []byte, maxBytesSize),
		stop:        make(chan struct{}),
		rotateHours: rotateHours,
		maxBackups:  maxBackups,
		timeTicker:  NewTimeTicker(rotateHours),
	}

	if err = asyncFileWriter.Start(); err != nil {
		return nil, fmt.Errorf("file writer start error. filePath=%s, err=%s", filePath, err)
	}

	return asyncFileWriter, nil
}

func (w *AsyncFileWriter) initLogFile() error {
	var (
		fd  *os.File
		err error
	)
	realFilePath := w.timeFilePath(w.filePath)
	// 检查目录是否存在
	if _, err = os.Stat(filepath.Dir(realFilePath)); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(realFilePath), os.ModePerm)
	}
	fd, err = os.OpenFile(realFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	w.fd = fd
	_, err = os.Lstat(w.filePath)
	if err == nil || os.IsExist(err) {
		err = os.Remove(w.filePath)
		if err != nil {
			return err
		}
	}
	os.Symlink(realFilePath, w.filePath)
	return nil
}

func (w *AsyncFileWriter) Start() error {
	if !atomic.CompareAndSwapInt32(&w.started, 0, 1) {
		return errors.New("logger has already been started")
	}
	err := w.initLogFile()
	if err != nil {
		return err
	}
	w.wg.Add(1)
	go func() {
		defer func() {
			atomic.StoreInt32(&w.started, 0)

			w.flushBuffer()
			w.flushAndClose()

			w.wg.Done()
		}()

		for {
			select {
			case msg, ok := <-w.buf:
				if !ok {
					fmt.Fprintln(os.Stderr, "buf channel has been closed.")
					return
				}
				w.SyncWrite(msg)
			case <-w.stop:
				return
			}
		}
	}()
	return nil
}

func (w *AsyncFileWriter) flushBuffer() {
	for {
		select {
		case msg := <-w.buf:
			w.SyncWrite(msg)
		default:
			return
		}
	}
}

func (w *AsyncFileWriter) SyncWrite(msg []byte) {
	w.rotateFile()
	if w.fd != nil {
		w.fd.Write(msg)
	}
}

func (w *AsyncFileWriter) rotateFile() {
	select {
	case <-w.timeTicker.C:
		if err := w.flushAndClose(); err != nil {
			fmt.Fprintf(os.Stderr, "flush and close file error. err=%s", err)
		}
		if err := w.initLogFile(); err != nil {
			fmt.Fprintf(os.Stderr, "init log file error. err=%s", err)
		}
		if err := w.removeExpiredFile(); err != nil {
			fmt.Fprintf(os.Stderr, "remove expired file error. err=%s", err)
		}
	default:
	}
}

func (w *AsyncFileWriter) Stop() {
	w.stop <- struct{}{}
	w.wg.Wait()
	w.timeTicker.Stop()
}

func (w *AsyncFileWriter) Write(msg []byte) (n int, err error) {
	buf := make([]byte, len(msg))
	copy(buf, msg)
	select {
	case w.buf <- buf:
	default:
	}
	return 0, nil
}

func (w *AsyncFileWriter) Flush() error {
	if w.fd == nil {
		return nil
	}
	return w.fd.Sync()
}

func (w *AsyncFileWriter) flushAndClose() error {
	if w.fd == nil {
		return nil
	}

	err := w.fd.Sync()
	if err != nil {
		return err
	}

	return w.fd.Close()
}

func (w *AsyncFileWriter) timeFilePath(filePath string) string {
	return filePath + "." + time.Now().Format(backupTimeFormat)
}

func (w *AsyncFileWriter) getExpiredFile(filePath string, maxBackups int, rotateHours uint) string {
	if rotateHours > 0 {
		maxBackups = int(rotateHours) * maxBackups
	}
	return filePath + "." + time.Now().Add(-time.Hour*time.Duration(maxBackups)).Format(backupTimeFormat)
}

func (w *AsyncFileWriter) removeExpiredFile() error {
	if w.maxBackups == 0 {
		return nil
	}

	oldFilepath := w.getExpiredFile(w.filePath, w.maxBackups, w.rotateHours)
	_, err := os.Stat(oldFilepath)
	if os.IsNotExist(err) {
		return nil
	}
	errRemove := os.Remove(oldFilepath)
	if err != nil {
		return errRemove
	}
	return err
}
