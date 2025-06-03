package logs

import (
	"fmt"

	"github.com/pterm/pterm"
)

type LogLevel int

const (
	DebugLevel LogLevel = 2
	InfoLevel  LogLevel = 3
	WarnLevel  LogLevel = 4
	ErrorLevel LogLevel = 5
)

var logger *pterm.Logger

func Debug(v ...any) {
	logger.Debug(fmt.Sprintln(v...))
}

func Info(v ...any) {
	logger.Info(fmt.Sprintln(v...))
}

func Warn(v ...any) {
	logger.WithCaller().WithCallerOffset(1).Warn(fmt.Sprintln(v...))
}

func Error(v ...any) {
	logger.WithCaller().WithCallerOffset(1).Error(fmt.Sprintln(v...))
}

func SetLogLevel(level LogLevel) {
	switch level {
	case DebugLevel:
		logger.Level = pterm.LogLevelDebug
	case InfoLevel:
		logger.Level = pterm.LogLevelInfo
	case WarnLevel:
		logger.Level = pterm.LogLevelWarn
	case ErrorLevel:
		logger.Level = pterm.LogLevelError
	}
}

func init() {
	logger = pterm.DefaultLogger.WithLevel(pterm.LogLevelTrace)
}
