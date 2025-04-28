package main

import (
	"log"
	"os"
	"runtime"
	"sync"
)

type LogType string

const (
	DebugLog LogType = "debug"
	WarnLog  LogType = "warn"
	FatalLog LogType = "fatal"
	InfoLog  LogType = "info"
)

func checkFileExists(filePath string) bool {
	// Check if the file exists
	if _, err := os.Stat(filePath); err == nil {
		return true
	}
	return false
}
func checkDirectoryExists(dirPath string) bool {
	// Check if the directory exists
	if _, err := os.Stat(dirPath); err == nil {
		return true
	}
	return false
}

func getVersion() string {
	// Get the version of the application
	return "0.0.1"
}

func getApplicationName() string {
	// Get the name of the application
	return "StreamAPI"
}

var (
	logLevel = DebugLog
	mu       = sync.Mutex{}
)

func setLogLevel(level LogType) {
	mu.Lock()
	defer mu.Unlock()
	// Set the log level
	logLevel = level
}

// logWithCaller logs a message with the class name and line number
func logWithCaller(message string, logtype LogType) {
	// Get the caller information
	pc, file, line, ok := runtime.Caller(1) // 1 means the immediate caller of this function
	if !ok {
		log.Fatalf("[%s:%d] %s - FATAL: %s", file, line, "logWithCaller", message)
		return
	}

	// Get the function name
	funcName := runtime.FuncForPC(pc).Name()

	switch logtype {
	case DebugLog:
		if logLevel == DebugLog {
			log.Printf("[%s:%d] %s - DEBUG: %s", file, line, funcName, message)
		}
	case InfoLog:
		if logLevel == DebugLog || logLevel == InfoLog {
			log.Printf("[%s:%d] %s - INFO: %s", file, line, funcName, message)
		}
	case WarnLog:
		if logLevel == DebugLog || logLevel == InfoLog || logLevel == WarnLog {
			log.Printf("[%s:%d] %s - WARN: %s", file, line, funcName, message)
		}
	case FatalLog:
		if logLevel == DebugLog || logLevel == InfoLog || logLevel == WarnLog || logLevel == FatalLog {
			log.Printf("[%s:%d] %s - FATAL: %s", file, line, funcName, message)
		}
	default:
		log.Printf("[%s:%d] %s - %s: %s", file, line, funcName, logtype, message)
	}
}
