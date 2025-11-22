package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Logger provides structured logging functionality
type Logger struct {
	verbose bool
}

// New creates a new logger instance
func New() *Logger {
	return &Logger{
		verbose: viper.GetBool("verbose"),
	}
}

// SetVerbose sets the verbose flag
func (l *Logger) SetVerbose(verbose bool) {
	l.verbose = verbose
}

// IsVerbose returns whether verbose logging is enabled
func (l *Logger) IsVerbose() bool {
	return l.verbose
}

// Info prints an info message
func (l *Logger) Info(format string, args ...interface{}) {
	fmt.Printf("ℹ️  "+format+"\n", args...)
}

// Success prints a success message
func (l *Logger) Success(format string, args ...interface{}) {
	fmt.Printf("✅ "+format+"\n", args...)
}

// Warning prints a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	fmt.Printf("⚠️  "+format+"\n", args...)
}

// Error prints an error message
func (l *Logger) Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "❌ "+format+"\n", args...)
}

// Debug prints a debug message (only in verbose mode)
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbose {
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("🔍 [%s] "+format+"\n", append([]interface{}{timestamp}, args...)...)
	}
}

// Verbose prints a verbose message (only in verbose mode)
func (l *Logger) Verbose(format string, args ...interface{}) {
	if l.verbose {
		fmt.Printf("📝 "+format+"\n", args...)
	}
}

// Step prints a step message (only in verbose mode)
func (l *Logger) Step(step int, total int, format string, args ...interface{}) {
	if l.verbose {
		fmt.Printf("📋 [%d/%d] "+format+"\n", append([]interface{}{step, total}, args...)...)
	}
}

// Progress prints a progress message (only in verbose mode)
func (l *Logger) Progress(format string, args ...interface{}) {
	if l.verbose {
		fmt.Printf("⏳ "+format+"\n", args...)
	}
}

// Global logger instance
var globalLogger = New()

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	return globalLogger
}

// UpdateVerbose updates the global logger's verbose setting
func UpdateVerbose() {
	globalLogger.verbose = viper.GetBool("verbose")
}

// Convenience functions for global logger
func Info(format string, args ...interface{}) {
	globalLogger.Info(format, args...)
}

func Success(format string, args ...interface{}) {
	globalLogger.Success(format, args...)
}

func Warning(format string, args ...interface{}) {
	globalLogger.Warning(format, args...)
}

func Error(format string, args ...interface{}) {
	globalLogger.Error(format, args...)
}

func Debug(format string, args ...interface{}) {
	globalLogger.Debug(format, args...)
}

func Verbose(format string, args ...interface{}) {
	globalLogger.Verbose(format, args...)
}

func Step(step int, total int, format string, args ...interface{}) {
	globalLogger.Step(step, total, format, args...)
}

func Progress(format string, args ...interface{}) {
	globalLogger.Progress(format, args...)
}

func IsVerbose() bool {
	return globalLogger.IsVerbose()
}
