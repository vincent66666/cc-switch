//go:build !darwin

package cli

import (
	"fmt"
	"os"
)

func rawTerminalSupported() bool {
	return false
}

func makeRawTerminal(file *os.File) (func(), error) {
	_ = file
	return nil, fmt.Errorf("当前平台不支持交互终端模式")
}
