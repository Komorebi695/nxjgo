package log

import (
	"fmt"
	"time"
)

type TextFormatter struct{}

func (f *TextFormatter) Formatter(params *LoggingFormatterParam) string {
	now := time.Now()
	var format string
	if params.Color {
		// Bring color.
		// The color of the error is red, while the information is green and the debug is blue.
		levelColor := f.LevelColor(params.Level)
		msgColor := f.MsgColor(params.Level)
		format = fmt.Sprintf("%s[nxjgo]%s %s%v%s |%s %s %s| msg:%s %+v %s", yellow, reset, blue, now.Format("2006/01/02 - 15:04:05"), reset,
			levelColor, params.Level.Level(), reset, msgColor, params.Msg, reset)
	} else {
		format = fmt.Sprintf("[nxjgo] %v | %s | msg: %+v", now.Format("2006/01/02 - 15:04:05"),
			params.Level.Level(), params.Msg)
	}
	if params.LoggerFields != nil {
		format = fmt.Sprintf("%s %#v", format, params.LoggerFields)
	}
	return format
}

func (f *TextFormatter) LevelColor(level LoggerLevel) string {
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

func (f *TextFormatter) MsgColor(level LoggerLevel) string {
	switch level {
	case LevelError:
		return red
	default:
		return ""
	}
}
