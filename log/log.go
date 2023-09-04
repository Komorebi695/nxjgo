package log

import (
	"fmt"
	"github.com/Komorebi695/nxjgo/internal/nxjstrings"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

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

type LoggerLevel int

type Fields map[string]any

const (
	LevelDebug LoggerLevel = iota
	LevelInfo
	LevelError
)

type Logger struct {
	Formatter    LoggerFormatter
	Level        LoggerLevel
	Outs         []*LoggerWriter
	LoggerFields Fields
	LogPath      string
	LogFileSize  int64 // M为单位
}

type LoggerWriter struct {
	Level LoggerLevel
	Out   io.Writer
}

type LoggerFormatter interface {
	Formatter(params *LoggingFormatterParam) string
}

type LoggingFormatterParam struct {
	Color        bool
	Level        LoggerLevel
	LoggerFields Fields
	Msg          any
}

//type LoggerFormatter struct {
//	Color        bool
//	Level        LoggerLevel
//	LoggerFields Fields
//}

func New() *Logger {
	return &Logger{}
}

func Default() *Logger {
	logger := New()
	out := &LoggerWriter{Level: LevelDebug, Out: os.Stdout}
	logger.Outs = append(logger.Outs, out)
	logger.Level = LevelDebug
	logger.Formatter = &TextFormatter{}
	return logger
}

func (l *Logger) Debug(msg any) {
	l.Print(LevelDebug, msg)
}

func (l *Logger) Info(msg any) {
	l.Print(LevelInfo, msg)
}

func (l *Logger) Error(msg any) {
	l.Print(LevelError, msg)
}

func (l *Logger) Print(level LoggerLevel, msg interface{}) {
	if l.Level > level {
		// Do not print logs if the level is not met.
		return
	}

	//l.Formatter.Level = level
	//l.Formatter.LoggerFields = l.LoggerFields
	paramFormatter := &LoggingFormatterParam{
		Level:        level,
		LoggerFields: l.LoggerFields,
		Msg:          msg,
	}
	formatter := l.Formatter.Formatter(paramFormatter)
	for _, out := range l.Outs {
		if out.Out == os.Stdout {
			paramFormatter.Color = true
			formatter = l.Formatter.Formatter(paramFormatter)
			_, _ = fmt.Fprintln(out.Out, formatter)
		}
		if out.Level == -1 || out.Level == level {
			paramFormatter.Color = false
			_, _ = fmt.Fprintln(out.Out, formatter)
			l.CheckFileSize(out)
		}
	}
}

func (l *Logger) WithFields(fields Fields) *Logger {
	return &Logger{
		Formatter:    l.Formatter,
		Outs:         l.Outs,
		Level:        l.Level,
		LoggerFields: fields,
		LogPath:      l.LogPath,
		LogFileSize:  l.LogFileSize,
	}
}

//
//func (f *LoggerFormatter) formatter(msg any) string {
//	now := time.Now()
//	var format string
//	if f.Color {
//		// Bring color.
//		// The color of the error is red, while the information is green and the debug is blue.
//		levelColor := f.LevelColor()
//		msgColor := f.MsgColor()
//		format = fmt.Sprintf("%s[nxjgo]%s %s%v%s |%s %s %s| msg =%s %#v %s", yellow, reset, blue, now.Format("2006/01/02 - 15:04:05"), reset,
//			levelColor, f.Level.Level(), reset, msgColor, msg, reset)
//	} else {
//		format = fmt.Sprintf("[nxjgo] %v | %s | msg =%#v", now.Format("2006/01/02 - 15:04:05"),
//			f.Level.Level(), msg)
//	}
//	if f.LoggerFields != nil {
//		format = fmt.Sprintf("%s %#v", format, f.LoggerFields)
//	}
//	return format
//}
//
//func (f *LoggerFormatter) LevelColor() string {
//	switch f.Level {
//	case LevelDebug:
//		return blue
//	case LevelInfo:
//		return green
//	case LevelError:
//		return red
//	default:
//		return cyan
//	}
//}
//
//func (f *LoggerFormatter) MsgColor() string {
//	switch f.Level {
//	case LevelError:
//		return red
//	default:
//		return ""
//	}
//}

func (level LoggerLevel) Level() string {
	switch level {
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

func (l *Logger) FileWriter(name string) (io.Writer, error) {
	return os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
}

func (l *Logger) SetLogPath(logPath string) {
	l.LogPath = logPath
	all, err := l.FileWriter(path.Join(logPath, "all.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, &LoggerWriter{Level: -1, Out: all})
	debug, err := l.FileWriter(path.Join(logPath, "debug.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, &LoggerWriter{Level: LevelDebug, Out: debug})
	info, err := l.FileWriter(path.Join(logPath, "info.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, &LoggerWriter{Level: LevelInfo, Out: info})
	logError, err := l.FileWriter(path.Join(logPath, "error.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, &LoggerWriter{Level: LevelError, Out: logError})
}

func (l *Logger) CloseWriter() {
	for _, out := range l.Outs {
		file := out.Out.(*os.File)
		if file != nil {
			_ = file.Close()
		}
	}
}

func (l *Logger) CheckFileSize(out *LoggerWriter) {
	osFile := out.Out.(*os.File)
	if osFile != nil {
		stat, err := osFile.Stat()
		if err != nil {
			log.Println("logger checkFileSize error info:", err)
			return
		}
		size := stat.Size()
		// The size needs to be checked.
		// If the conditions are met, the file needs to be recreated and the output in the logger needs to be replaced.
		if l.LogFileSize <= 0 {
			// Default 100M
			l.LogFileSize = 100 << 20
		}
		if size >= l.LogFileSize {
			_, fileName := path.Split(osFile.Name())
			name := fileName[0:strings.Index(fileName, ".")]
			w, err := l.FileWriter(path.Join(l.LogPath, nxjstrings.JoinStrings(name, ".", time.Now().UnixMilli(), ".log")))
			if err != nil {
				log.Println("logger checkFileSize error info :", err)
				return
			}
			out.Out = w
		}
	}

}
