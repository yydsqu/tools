package log

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	backupTimeFormat = "20060102150405"
)

var (
	ErrLoggerAlreadyStarted = errors.New("logger has already been started")
	ErrLoggerNotStarted     = errors.New("logger not started")
	ErrBufferFull           = errors.New("async file writer buffer full")
)

// TimeTicker 在指定小时对齐的时间点触发（比如每 2 小时的 0:00/2:00/4:00 ...）
type TimeTicker struct {
	stop chan struct{}
	C    <-chan time.Time
}

func NewTimeTicker(rotateHours uint) *TimeTicker {
	// 默认 24 小时
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
				select {
				case ch <- t:
				default:
				}
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

	wg      sync.WaitGroup
	started int32 // 0: 未启动 / 已停; 1: 正在运行
	stopped int32 // Stop() 调用标记

	buf        chan []byte
	stop       chan struct{}
	timeTicker *TimeTicker

	rotateHours uint
	maxBackups  int
}

// NewAsyncFileWriter
// - filePath: 日志文件路径（会创建 symlink 指向当前日志文件）
// - maxBytesSize: 实际被当作“队列最大条数”，而不是字节数（为保持签名不改动）
// - maxBackups: 保留的历史文件个数
// - rotateHours: 轮转间隔（小时），0 默认 24
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

	// 统一 rotateHours 默认值，保证 TimeTicker 和删除过期文件逻辑一致
	rotateHours = cmp.Or(rotateHours, 24)

	// 把 maxBytesSize 当成“队列最大条数”使用，为避免极端情况，做个 sane 限制
	queueSize := int(maxBytesSize)
	if queueSize <= 0 {
		queueSize = 1024
	}
	if queueSize > 1_000_000 {
		queueSize = 1_000_000
	}

	asyncFileWriter := &AsyncFileWriter{
		filePath:    absFilePath,
		buf:         make(chan []byte, queueSize),
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
	if !atomic.CompareAndSwapInt32(&w.started, 0, 1) {
		return ErrLoggerAlreadyStarted
	}

	if err := w.initLogFile(); err != nil {
		atomic.StoreInt32(&w.started, 0)
		return err
	}

	w.wg.Add(1)
	go func() {
		defer func() {
			// 标记已停止
			atomic.StoreInt32(&w.started, 0)

			// 把缓冲区中的所有日志写完
			w.flushBuffer()
			// 刷盘并关闭文件
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
		_, _ = w.fd.Write(msg)
	}
}

func (w *AsyncFileWriter) rotateFile() {
	select {
	case <-w.timeTicker.C:
		if err := w.flushAndClose(); err != nil {
			fmt.Fprintf(os.Stderr, "flush and close file error. err=%s\n", err)
		}
		if err := w.initLogFile(); err != nil {
			fmt.Fprintf(os.Stderr, "init log file error. err=%s\n", err)
		}
		if err := w.removeExpiredFile(); err != nil {
			fmt.Fprintf(os.Stderr, "remove expired file error. err=%s\n", err)
		}
	default:
	}
}

func (w *AsyncFileWriter) Stop() {
	if !atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		// 已经 Stop 过了，直接返回
		return
	}

	// 关闭 stop channel，通知后台 goroutine 退出
	close(w.stop)

	// 等待后台 goroutine 完成 flush + 关闭文件
	w.wg.Wait()

	// 停止时间 ticker
	w.timeTicker.Stop()
}

func (w *AsyncFileWriter) Write(msg []byte) (n int, err error) {
	if atomic.LoadInt32(&w.started) == 0 {
		return 0, ErrLoggerNotStarted
	}

	buf := make([]byte, len(msg))
	copy(buf, msg)

	select {
	case w.buf <- buf:
		return len(msg), nil
	default:
		// 队列已满：这里选择返回 error，避免默默丢日志
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
	return filePath + "." + time.Now().Format(backupTimeFormat)
}

func (w *AsyncFileWriter) parseBackupTime(path string) (time.Time, error) {
	prefix := w.filePath + "."
	ts := strings.TrimPrefix(path, prefix)
	return time.ParseInLocation(backupTimeFormat, ts, time.Local)
}

type fileInfo struct {
	path string
	t    time.Time
}

func (w *AsyncFileWriter) removeExpiredFile() error {
	if w.maxBackups <= 0 {
		return nil
	}
	pattern := w.filePath + ".*"
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	if len(files) <= w.maxBackups {
		return nil
	}

	var infos []fileInfo
	for _, f := range files {
		t, err := w.parseBackupTime(f)
		if err != nil {
			t = time.Time{}
		}
		infos = append(infos, fileInfo{path: f, t: t})
	}

	// 按时间排序（旧的在前），解析失败的在最前
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].t.Equal(infos[j].t) {
			return infos[i].path < infos[j].path
		}
		return infos[i].t.Before(infos[j].t)
	})

	// 需要删除的数量
	toDelete := infos[0 : len(infos)-w.maxBackups]

	var lastErr error
	for _, fi := range toDelete {
		if err := os.Remove(fi.path); err != nil && !os.IsNotExist(err) {
			lastErr = err
		}
	}
	return lastErr
}
