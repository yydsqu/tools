//go:build !plan9 && !windows && !wasm && !freebsd

package utils

import (
	"golang.org/x/sys/unix"
	"regexp"
	"syscall"
)

var (
	rule = regexp.MustCompile(`^eth|^enp1|^ens|^enx|^enp`)
)

func Control(network, address string, c syscall.RawConn) (err error) {
	return c.Control(func(fd uintptr) {
		if err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
			return
		}
		unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	})
}
