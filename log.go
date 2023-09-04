package nxjgo

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
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

var DefaultWriter io.Writer = os.Stdout

type LoggingConfig struct {
	Formatter LoggerFormatter
	out       io.Writer
	IsColor   bool
}

type LoggerFormatter func(params LogFormatterParams) string

type LogFormatterParams struct {
	Request        *http.Request
	TimeStamp      time.Time
	StatusCode     int
	Latency        time.Duration
	ClientIP       net.IP
	Method         string
	Path           string
	IsDisplayColor bool
}

func (p *LogFormatterParams) StatusCodeColor() string {
	code := p.StatusCode
	switch code {
	case http.StatusOK:
		return green
	default:
		return red
	}
}

func (p *LogFormatterParams) ResetColor() string {
	return reset
}

func Logging(next HandlerFunc) HandlerFunc {
	return LoggerWithConfig(LoggingConfig{}, next)
}

func LoggerWithConfig(conf LoggingConfig, next HandlerFunc) HandlerFunc {
	formatter := conf.Formatter
	if formatter == nil {
		formatter = defaultLogFormatter
	}
	out := conf.out
	displayColor := conf.IsColor
	if out == nil {
		out = DefaultWriter
	}
	if out != os.Stdout {
		displayColor = false
	}
	return func(ctx *Context) {
		start := time.Now()
		path := ctx.R.URL.Path
		raw := ctx.R.URL.RawQuery
		next(ctx)
		stop := time.Now()
		latency := stop.Sub(start)
		ip, _, _ := net.SplitHostPort(strings.TrimSpace(ctx.R.RemoteAddr))
		clientIP := net.ParseIP(ip)
		method := ctx.R.Method
		statusCode := ctx.StatusCode

		if raw != "" {
			path = path + "?" + raw
		}
		params := LogFormatterParams{
			Request:        ctx.R,
			TimeStamp:      stop,
			Latency:        latency,
			ClientIP:       clientIP,
			Method:         method,
			Path:           path,
			StatusCode:     statusCode,
			IsDisplayColor: displayColor,
		}
		params.IsDisplayColor = true // todo 删除
		fmt.Fprintf(out, formatter(params))
	}
}

var defaultLogFormatter = func(params LogFormatterParams) string {
	if params.Latency > time.Minute {
		params.Latency = params.Latency.Truncate(time.Second)
	}
	statusCodeColor := params.StatusCodeColor()
	resetColor := params.ResetColor()
	if params.IsDisplayColor {
		return fmt.Sprintf("%s[nxjgo]%s %s%v%s | %s %3d %s |%s %13v %s| %15s  |%s %-7s %s %s %#v %s\n",
			yellow, resetColor, blue, params.TimeStamp.Format("2006/01/02 - 15:04:05"), resetColor,
			statusCodeColor, params.StatusCode, resetColor,
			red, params.Latency, resetColor,
			params.ClientIP,
			magenta, params.Method, resetColor,
			cyan, params.Path, resetColor,
		)
	}
	return fmt.Sprintf("[nxjgo] %v | %3d | %13v | %15s |%-7s %#v\n",
		params.TimeStamp.Format("2006/01/02 - 15:04:05"),
		params.StatusCode,
		params.Latency, params.ClientIP, params.Method, params.Path)
}
