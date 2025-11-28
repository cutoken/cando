package logging

import (
	"log"
	"os"
)

var (
	// DevMode indicates if development logging is enabled
	DevMode = os.Getenv("DEV_MODE") == "1"
	// Logger is the shared logger instance
	Logger *log.Logger
)

func init() {
	Logger = log.Default()
}

// DevLog logs only when DEV_MODE=1
func DevLog(format string, args ...interface{}) {
	if DevMode {
		Logger.Printf("[DEV] "+format, args...)
	}
}

// UserLog logs important user-facing information (always visible)
func UserLog(format string, args ...interface{}) {
	Logger.Printf("[USER] "+format, args...)
}

// ErrorLog logs errors (always visible)
func ErrorLog(format string, args ...interface{}) {
	Logger.Printf("[ERROR] "+format, args...)
}
