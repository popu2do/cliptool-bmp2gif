//go:build windows

package applog

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func redirectStandardStreams(target *os.File) error {
	handle := windows.Handle(target.Fd())
	if err := windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, handle); err != nil {
		return fmt.Errorf("重定向标准输出失败: %w", err)
	}
	if err := windows.SetStdHandle(windows.STD_ERROR_HANDLE, handle); err != nil {
		return fmt.Errorf("重定向标准错误失败: %w", err)
	}
	os.Stdout = target
	os.Stderr = target
	return nil
}
