package logger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	backupTimeFormat = "2006_01_02"
)

var (
	DAY                     = time.Hour * 24
	ErrLoggerAlreadyStarted = errors.New("logger has already been started")
	ErrLoggerNotStarted     = errors.New("logger not started")
	ErrBufferFull           = errors.New("async file writer buffer full")
)

type AsyncFileWriter struct {
	filePath   string
	fd         *os.File
	running    atomic.Bool
	buf        chan []byte
	maxBackups int
	timeTimer  *time.Timer
	wg         sync.WaitGroup
}

func (w *AsyncFileWriter) Close() error {
	if w.running.CompareAndSwap(true, false) {
		close(w.buf)
	}
	w.wg.Wait()
	return nil
}

func (w *AsyncFileWriter) initLogFile() error {
	realFilePath := w.timeFilePath(w.filePath)

	// 确保目录存在
	if _, err := os.Stat(filepath.Dir(realFilePath)); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(filepath.Dir(realFilePath), os.ModePerm); mkErr != nil {
			return mkErr
		}
	}

	fd, err := os.OpenFile(realFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	w.fd = fd

	// 删除旧的 symlink（如果存在）
	if _, err := os.Lstat(w.filePath); err == nil {
		if rmErr := os.Remove(w.filePath); rmErr != nil {
			return rmErr
		}
	}

	// 创建新的 symlink 指向当前实际文件
	if err := os.Symlink(realFilePath, w.filePath); err != nil {
		return err
	}
	return nil
}

func (w *AsyncFileWriter) Start() error {
	if w.running.CompareAndSwap(false, true) == false {
		return ErrLoggerAlreadyStarted
	}
	if err := w.initLogFile(); err != nil {
		return err
	}

	w.wg.Add(1)

	go func() {
		defer func() {
			w.flushBuffer()
			_ = w.flushAndClose()
			w.wg.Done()
		}()
		for {
			select {
			case msg, ok := <-w.buf:
				if !ok {
					fmt.Fprintln(os.Stderr, "buf channel has been closed.")
					return
				}
				w.write(msg)
			}
		}
	}()

	return nil
}

func (w *AsyncFileWriter) flushBuffer() {
	for msg := range w.buf {
		w.write(msg)
	}
}

func (w *AsyncFileWriter) write(msg []byte) {
	select {
	case now := <-w.timeTimer.C:
		if err := w.flushAndClose(); err != nil {
			fmt.Fprintf(os.Stderr, "flush and close file error. err=%s", err)
		}
		if err := w.initLogFile(); err != nil {
			fmt.Fprintf(os.Stderr, "init log file error. err=%s", err)
		}
		if err := w.removeExpiredFile(); err != nil {
			fmt.Fprintf(os.Stderr, "remove expired file error. err=%s", err)
		}
		date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Add(DAY)
		w.timeTimer.Reset(date.Sub(now))
	default:
		w.fd.Write(msg)
	}
}

func (w *AsyncFileWriter) Write(msg []byte) (n int, err error) {
	if w.running.Load() == false {
		return 0, ErrLoggerNotStarted
	}
	buf := make([]byte, len(msg))
	copy(buf, msg)
	select {
	case w.buf <- buf:
		return len(msg), nil
	default:
		return 0, ErrBufferFull
	}
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

	if err := w.fd.Sync(); err != nil {
		return err
	}

	err := w.fd.Close()
	w.fd = nil
	return err
}

func (w *AsyncFileWriter) timeFilePath(filePath string) string {
	return filePath + "." + time.Now().Format("2006_01_02")
}

func (w *AsyncFileWriter) parseBackupTime(path string) (time.Time, error) {
	prefix := w.filePath + "."
	ts := strings.TrimPrefix(path, prefix)
	return time.ParseInLocation(backupTimeFormat, ts, time.Local)
}

func (w *AsyncFileWriter) removeExpiredFile() error {
	if w.maxBackups <= 0 {
		return nil
	}
	files, err := filepath.Glob(w.filePath + ".*")
	if err != nil {
		return err
	}
	before := time.Now().Add(DAY * time.Duration(-w.maxBackups))
	for _, filePath := range files {
		stat, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		if stat.ModTime().Before(before) {
			os.Remove(filePath)
		}
	}
	return nil
}

func NewAsyncFileWriter(filePath string, maxBytesSize int, maxBackups int) (*AsyncFileWriter, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("get file path of logger error. filePath=%s, err=%s", filePath, err)
	}

	if _, err = os.Stat(filepath.Dir(absFilePath)); os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(absFilePath), os.ModePerm); err != nil {
			return nil, fmt.Errorf("mkdir error path=%s, err=%s", filepath.Dir(absFilePath), err)
		}
	}

	queueSize := maxBytesSize
	if queueSize <= 0 {
		queueSize = 1024
	}

	if queueSize > 1_000_000 {
		queueSize = 1_000_000
	}

	asyncFileWriter := &AsyncFileWriter{
		filePath:   absFilePath,
		buf:        make(chan []byte, queueSize),
		maxBackups: maxBackups,
		timeTimer:  time.NewTimer(0),
	}

	if err = asyncFileWriter.Start(); err != nil {
		return nil, fmt.Errorf("file writer start error. filePath=%s, err=%s", filePath, err)
	}

	return asyncFileWriter, nil
}
