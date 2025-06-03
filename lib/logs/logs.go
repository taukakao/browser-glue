package logs

import (
	"fmt"
	"log"
	"os"

	"github.com/pterm/pterm"
)

func init() {
	pterm.DefaultLogger.Level = pterm.LogLevelTrace
}

var InfoLogger = log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
var WarnLogger = log.New(os.Stderr, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
var ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)

func Debug(v ...any) {
	pterm.DefaultLogger.Debug(fmt.Sprintln(v...))
}

func Info(v ...any) {
	// InfoLogger.Println(v...)
	pterm.DefaultLogger.Info(fmt.Sprintln(v...))
}

func Warn(v ...any) {
	// WarnLogger.Println(v...)
	pterm.DefaultLogger.WithCaller().WithCallerOffset(1).Warn(fmt.Sprintln(v...))
}

func Error(v ...any) {
	// ErrorLogger.Println(v...)
	pterm.DefaultLogger.WithCaller().WithCallerOffset(1).Error(fmt.Sprintln(v...))
}
