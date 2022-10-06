package log

import (
	"fmt"
	"io"
	"os"
)

const (
	LevelDebug LoggerLevel = iota
	LevelInfo
	LevelError
)

// LoggerLevel is the level of the log
type LoggerLevel int

// LoggerFormatter is the print format of your log
type LoggerFormatter struct {
}

// Logger is your log
type Logger struct {
	Formatter LoggerFormatter // print format
	Level     LoggerLevel     // log's level
	Outs      []io.Writer     // output of log
}

func New() *Logger {
	return &Logger{}
}

// Default init the default setting for logger in vex
// LevelDebug default level is
// os.Stdout is the default method to show the log
// LoggerFormatter is the default formatter for the log
func Default() *Logger {
	logger := New()
	logger.Level = LevelDebug
	logger.Outs = append(logger.Outs, os.Stdout)
	logger.Formatter = LoggerFormatter{}
	return logger
}

// Debug print the debug level log
func (l *Logger) Debug(msg any) {
	l.PrintLog(LevelDebug, msg)
}

// Info print the info level log
func (l *Logger) Info(msg any) {
	l.PrintLog(LevelInfo, msg)
}

// Error print the error level log
func (l *Logger) Error(msg any) {
	l.PrintLog(LevelError, msg)
}

func (l *Logger) PrintLog(level LoggerLevel, msg any) {
	// if level > print level do not print the log in the same level
	if l.Level > level {
		return
	}
	for _, out := range l.Outs {
		fmt.Fprintln(out, msg)
	}
}
