package log

import (
	"fmt"
	"io"
	"os"
	"time"
)

const (
	LevelDebug LoggerLevel = iota
	LevelInfo
	LevelError
)

// color of logger
const (
	greenBg   = "\033[97;42m"
	whiteBg   = "\033[90;47m"
	yellowBg  = "\033[90;43m"
	redBg     = "\033[97;41m"
	blueBg    = "\033[97;44m"
	magentaBg = "\033[97;45m"
	cyanBg    = "\033[97;46m"
	green     = "\033[32m"
	white     = "\033[37m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
	reset     = "\033[0m"
)

// LoggerLevel is the level of the log
type LoggerLevel int

// Level to get the level logger's level and return
func (l LoggerLevel) Level() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	default:
		return ""
	}
}

// Fields to show the k-v msg
type Fields map[string]any

// LogFormatParam is a struct get the log's param
type LogFormatParam struct {
	Level          LoggerLevel
	IsDisplayColor bool
	LoggerFields   Fields // loggerFields
	Msg            any
}

// LogFormatter is an interface to format print log
type LogFormatter interface {
	Format(param *LogFormatParam) string
}

// LoggerFormatter is the print format of your log
type LoggerFormatter struct {
	Level          LoggerLevel
	IsDisplayColor bool
	LoggerFields   Fields // loggerFields
}

// Logger is your log
type Logger struct {
	Formatter    LogFormatter // print format
	Level        LoggerLevel  // log's level
	Outs         []io.Writer  // output of log
	LoggerFields Fields       // loggerFields
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
	// init with a default interface impl
	logger.Formatter = &TextFormatter{}
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

// PrintLog is a method to print the information of log
func (l *Logger) PrintLog(level LoggerLevel, msg any) {
	// if level > print level do not print the log in the same level
	if l.Level > level {
		return
	}
	// init param
	param := &LogFormatParam{
		Level:        level,
		LoggerFields: l.LoggerFields,
		Msg:          msg,
	}
	// change the interface method
	str := l.Formatter.Format(param)
	for _, out := range l.Outs {
		// if this log is a standard output in console set the color
		if out == os.Stdout {
			param.IsDisplayColor = true
			str = l.Formatter.Format(param)
		}
		fmt.Fprintln(out, str)
	}
}

func (l *Logger) WithFields(fields Fields) *Logger {
	// rebuild a logger with Fields
	return &Logger{
		Formatter:    l.Formatter,
		Outs:         l.Outs,
		Level:        l.Level,
		LoggerFields: fields,
	}
}

func (f *LoggerFormatter) format(msg any) string {
	now := time.Now()
	if f.IsDisplayColor {
		// set the color to level logger | error -> red	| info -> green | debug -> blue
		levelColor := f.LevelColor()
		msgColor := f.MsgColor()
		return fmt.Sprintf("%s [vex] %s %s%v%s | level %s %s %s | msg=%s %#v %s | fields=%v\n",
			yellow, reset, blue, now.Format("2006/01/02 - 15:04:05"), reset,
			levelColor, f.Level.Level(), reset, msgColor, msg, reset, f.LoggerFields,
		)
	}
	return fmt.Sprintf("[vex] %v | level=%s | msg=%#v | fields=%v",
		now.Format("2006/01/02 - 15:04:05"),
		f.Level.Level(),
		msg,
		f.LoggerFields,
	)
}

// LevelColor is the method to set the color for the relative log's level
func (f *LoggerFormatter) LevelColor() string {
	switch f.Level {
	case LevelDebug:
		return blue
	case LevelInfo:
		return green
	case LevelError:
		return red
	default:
		return cyan
	}
}

// MsgColor is the method to set the color for the relative msg's level
func (f *LoggerFormatter) MsgColor() string {
	switch f.Level {
	case LevelDebug:
		return blue
	case LevelInfo:
		return green
	case LevelError:
		return red
	default:
		return ""
	}
}
