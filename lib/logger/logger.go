package logger

import (
	"log"
	"os"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)

// debugMode is set via -ldflags at build time
var debugMode = "true"

// Debug logs debug messages only when debug mode is enabled
func Debug(format string, v ...interface{}) {
	if debugMode == "true" {
		logger.Printf("[DEBUG] "+format, v...)
	}
}

func Info(format string, v ...interface{}) {
	logger.Printf("[INFO] "+format, v...)
}

func Error(format string, v ...interface{}) {
	logger.Printf("[ERROR] "+format, v...)
}

func Warn(format string, v ...interface{}) {
	logger.Printf("[WARN] "+format, v...)
}
