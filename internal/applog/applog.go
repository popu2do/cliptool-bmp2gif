package applog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	mu       sync.Mutex
	file     *os.File
	logger   *log.Logger
	logPath  string
	initOnce sync.Once
)

func Init(appName string) (string, error) {
	var initErr error
	initOnce.Do(func() {
		baseDir, err := executableDir()
		if err != nil {
			initErr = err
			return
		}

		logDir := filepath.Join(baseDir, "logs")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			initErr = err
			return
		}

		name := fmt.Sprintf("%s-%s.log", appName, time.Now().Format("20060102"))
		logPath = filepath.Join(logDir, name)
		file, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			initErr = err
			return
		}
		if err := redirectStandardStreams(file); err != nil {
			initErr = err
			return
		}
		logger = log.New(file, "", log.LstdFlags|log.Lmicroseconds)
	})
	return logPath, initErr
}

func Path() string {
	mu.Lock()
	defer mu.Unlock()
	return logPath
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		_ = file.Sync()
		_ = file.Close()
		file = nil
		logger = nil
	}
}

func Debugf(format string, args ...interface{}) {
	writef("DEBUG", format, args...)
}

func Infof(format string, args ...interface{}) {
	writef("INFO", format, args...)
}

func Warnf(format string, args ...interface{}) {
	writef("WARN", format, args...)
}

func Errorf(format string, args ...interface{}) {
	writef("ERROR", format, args...)
}

func writef(level string, format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if logger == nil {
		return
	}
	logger.Printf("[%s] %s", level, fmt.Sprintf(format, args...))
}

func executableDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取程序路径失败: %w", err)
	}
	return filepath.Dir(exePath), nil
}

type WailsLogger struct{}

func NewWailsLogger() WailsLogger {
	return WailsLogger{}
}

func (WailsLogger) Print(message string) {
	Infof("[Wails] %s", message)
}

func (WailsLogger) Trace(message string) {
	Debugf("[Wails] %s", message)
}

func (WailsLogger) Debug(message string) {
	Debugf("[Wails] %s", message)
}

func (WailsLogger) Info(message string) {
	Infof("[Wails] %s", message)
}

func (WailsLogger) Warning(message string) {
	Warnf("[Wails] %s", message)
}

func (WailsLogger) Error(message string) {
	Errorf("[Wails] %s", message)
}

func (WailsLogger) Fatal(message string) {
	Errorf("[Wails] %s", message)
	os.Exit(1)
}
