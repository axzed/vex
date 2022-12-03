package log

import (
	"fmt"
	"strings"
	"time"
)

type TextFormatter struct {
}

// Format impl the LogFormatter interface method
func (t *TextFormatter) Format(param *LogFormatParam) string {
	now := time.Now()
	fieldsStr := ""
	// build fieldsStr
	if param.LoggerFields != nil {
		var sb strings.Builder
		for k, v := range param.LoggerFields {
			fmt.Fprintf(&sb, "%s=%v", k, v)
			fmt.Fprintf(&sb, " ")
		}
		fieldsStr = sb.String()
	}
	if param.IsDisplayColor {
		// set the color to level logger | error -> red	| info -> green | debug -> blue
		levelColor := t.LevelColor(param.Level)
		msgColor := t.MsgColor(param.Level)
		return fmt.Sprintf("%s [vex] %s | %s%v%s | level %s %s %s | msg=%s %v %s | %s %s %s",
			yellow, reset, blue, now.Format("2006/01/02 - 15:04:05"), reset,
			levelColor, param.Level.Level(), reset, msgColor, param.Msg, reset, magenta, fieldsStr, reset,
		)
	}
	return fmt.Sprintf("[vex] | %v | level=%s | msg=%v | %s",
		now.Format("2006/01/02 - 15:04:05"),
		param.Level.Level(),
		param.Msg,
		fieldsStr,
	)
}

// LevelColor to get level color
func (t *TextFormatter) LevelColor(level LoggerLevel) string {
	switch level {
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

// MsgColor to get Msg color
func (t *TextFormatter) MsgColor(level LoggerLevel) string {
	switch level {
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
