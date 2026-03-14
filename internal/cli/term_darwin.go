//go:build darwin

package cli

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func rawTerminalSupported() bool {
	return true
}

func makeRawTerminal(file *os.File) (func(), error) {
	fd := file.Fd()

	termios, err := getTermios(fd)
	if err != nil {
		return nil, err
	}

	raw := *termios
	raw.Lflag &^= syscall.ICANON | syscall.ECHO | syscall.ISIG
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if err := setTermios(fd, &raw); err != nil {
		return nil, err
	}

	return func() {
		_ = setTermios(fd, termios)
	}, nil
}

func getTermios(fd uintptr) (*syscall.Termios, error) {
	termios := &syscall.Termios{}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGETA), uintptr(unsafe.Pointer(termios)))
	if errno != 0 {
		return nil, fmt.Errorf("读取终端状态失败：%w", errno)
	}
	return termios, nil
}

func setTermios(fd uintptr, termios *syscall.Termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSETA), uintptr(unsafe.Pointer(termios)))
	if errno != 0 {
		return fmt.Errorf("设置终端状态失败：%w", errno)
	}
	return nil
}
