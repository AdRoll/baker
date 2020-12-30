// +build !windows

package baker

import (
	"syscall"
	"unsafe"
)

func terminalWidth() uint {
	const (
		maxWidth     = 140 // don't go over 140 chars anyway
		defaultWidth = 110 // in case we can't get the terminal width
	)

	var w uint

	defer func() {
		if err := recover(); err != nil {
			w = defaultWidth
		}
	}()

	ws := &struct{ Row, Col, Xpixel, Ypixel uint16 }{}

	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		panic(errno)
	}

	if ws.Col > maxWidth {
		return maxWidth
	}

	w = uint(ws.Col)
	return w
}
