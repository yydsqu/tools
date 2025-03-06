package utils

import (
	"regexp"
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	rule = regexp.MustCompile(`WLAN|以太网`)
)

func Control(network, address string, c syscall.RawConn) (err error) {
	return c.Control(func(fd uintptr) {
		err = windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
	})
}
