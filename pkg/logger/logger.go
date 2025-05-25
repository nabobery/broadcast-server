package logger

import (
	"log"
	"os"
	"time"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
)

func init() {
	infoLogger = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime)
	errorLogger = log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime)
	debugLogger = log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime)
}

// Info logs an informational message
func Info(format string, v ...interface{}) {
	infoLogger.Printf(format, v...)
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	errorLogger.Printf(format, v...)
}

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	debugLogger.Printf(format, v...)
}

// LogRequest logs details about a request
func LogRequest(remoteAddr string, method string, path string) {
	Info("[%s] %s %s %s", remoteAddr, method, path, time.Now().Format(time.RFC3339))
}

// Fatal logs an error message and exits the program
func Fatal(format string, v ...interface{}) {
	errorLogger.Printf(format, v...)
	os.Exit(1)
}

// PrintServerBanner prints a server startup banner
func PrintServerBanner(port int) {
	Info("-------------------------------------")
	Info("       Broadcast Server Started      ")
	Info("        Listening on port %d        ", port)
	Info("-------------------------------------")
}
