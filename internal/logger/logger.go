package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// Logger wraps the standard logger with file output for MCP stdio mode
type Logger struct {
	file    *os.File
	logger  *log.Logger
	isStdio bool
}

var (
	// Default logger instance
	defaultLogger *Logger
)

// Init initializes the logger
// If isStdio is true, logs will be written to a file
// Otherwise, logs will be written to stderr
func Init(isStdio bool) error {
	defaultLogger = &Logger{
		isStdio: isStdio,
	}

	if isStdio {
		// Create log directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		logDir := filepath.Join(homeDir, ".blaxel")
		if os.Getenv("LOG_DIR") != "" {
			logDir = os.Getenv("LOG_DIR")
		}
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		logFile := filepath.Join(logDir, "mcp-server.log")

		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}

		defaultLogger.file = file
		defaultLogger.logger = log.New(file, "", log.LstdFlags|log.Lshortfile)

		// Write initial log entry with file location
		defaultLogger.logger.Printf("MCP Server started - Log file: %s", logFile)
	} else {
		// Use stderr for non-stdio mode (doesn't interfere with stdout protocols)
		defaultLogger.logger = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
	}

	return nil
}

// Close closes the log file if open
func Close() error {
	if defaultLogger != nil && defaultLogger.file != nil {
		return defaultLogger.file.Close()
	}
	return nil
}

// GetLogWriter returns the io.Writer for the logger
// This can be used to redirect other outputs
func GetLogWriter() io.Writer {
	if defaultLogger == nil {
		// Fallback to stderr if not initialized
		return os.Stderr
	}

	if defaultLogger.file != nil {
		return defaultLogger.file
	}

	return os.Stderr
}

// Printf logs a formatted message
func Printf(format string, v ...interface{}) {
	if defaultLogger == nil {
		// Fallback to stderr if not initialized
		log.Printf(format, v...)
		return
	}
	defaultLogger.logger.Printf(format, v...)
}

// Println logs a message with a newline
func Println(v ...interface{}) {
	if defaultLogger == nil {
		// Fallback to stderr if not initialized
		log.Println(v...)
		return
	}
	defaultLogger.logger.Println(v...)
}

// Fatalf logs a formatted message and exits
func Fatalf(format string, v ...interface{}) {
	if defaultLogger == nil {
		// Fallback to stderr if not initialized
		log.Fatalf(format, v...)
		return
	}
	defaultLogger.logger.Fatalf(format, v...)
}

// Fatal logs a message and exits
func Fatal(v ...interface{}) {
	if defaultLogger == nil {
		// Fallback to stderr if not initialized
		log.Fatal(v...)
		return
	}
	defaultLogger.logger.Fatal(v...)
}

// Debugf logs a debug message (only if debug mode is enabled)
func Debugf(format string, v ...interface{}) {
	if defaultLogger == nil {
		return
	}
	// Could check a debug flag here if needed
	defaultLogger.logger.Printf("[DEBUG] "+format, v...)
}

// Warnf logs a warning message
func Warnf(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Printf("[WARNING] "+format, v...)
		return
	}
	defaultLogger.logger.Printf("[WARNING] "+format, v...)
}

// Errorf logs an error message
func Errorf(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Printf("[ERROR] "+format, v...)
		return
	}
	defaultLogger.logger.Printf("[ERROR] "+format, v...)
}

// IsStdio returns whether the logger is in stdio mode
func IsStdio() bool {
	if defaultLogger == nil {
		return false
	}
	return defaultLogger.isStdio
}
