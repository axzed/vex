package vex

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
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

// DefaultWriter standard output
var DefaultWriter io.Writer = os.Stdout

type LoggerConfig struct {
	Formatter LoggerFormatter
	out       io.Writer
}

type LoggerFormatter = func(params *LogFormatterParams) string

// LogFormatterParams information that the logger want to display
type LogFormatterParams struct {
	Request        *http.Request
	TimeStamp      time.Time
	StatusCode     int
	Latency        time.Duration
	ClientIP       net.IP
	Method         string
	Path           string
	IsDisplayColor bool // do you want to display the log with color ?
}

// StatusCodeColor set the color of StatusCode
func (p *LogFormatterParams) StatusCodeColor() string {
	code := p.StatusCode
	switch code {
	case http.StatusOK:
		return green
	default:
		return red
	}
}

// ResetColor when you change the color in an attributed, you need to reset it to the normal case when this case end
func (p *LogFormatterParams) ResetColor() string {
	return reset
}

var defaultFormatter = func(params *LogFormatterParams) string {
	var statusCodeColor = params.StatusCodeColor()
	var reset = params.ResetColor()
	// change to second when you process time > 1min
	if params.Latency > time.Minute {
		params.Latency = params.Latency.Truncate(time.Second)
	}
	if params.IsDisplayColor {
		return fmt.Sprintf("%s [VEX] %s |%s %v %s| %s %3d %s |%s %13v %s| %15s  |%s %-7s %s %s %#v %s\n",
			yellow, reset, blue, params.TimeStamp.Format("2006/01/02 - 15:04:05"), reset,
			statusCodeColor, params.StatusCode, reset,
			red, params.Latency, reset,
			params.ClientIP,
			magenta, params.Method, reset,
			cyan, params.Path, reset,
		)
	}
	return fmt.Sprintf("[msgo] %v | %3d | %13v | %15s |%-7s %#v",
		params.TimeStamp.Format("2006/01/02 - 15:04:05"),
		params.StatusCode,
		params.Latency, params.ClientIP, params.Method, params.Path,
	)

}

// LoggerWithConfig init the logger with the configuration
func LoggerWithConfig(conf LoggerConfig, next HandleFunc) HandleFunc {
	formatter := conf.Formatter
	if formatter == nil {
		formatter = defaultFormatter
	}
	out := conf.out
	displayColor := false
	if out == nil {
		out = DefaultWriter
		displayColor = true
	}
	if formatter == nil {
		formatter = defaultFormatter
	}
	return func(ctx *Context) {
		// get request from context's r
		r := ctx.R
		param := &LogFormatterParams{
			Request:        r,
			IsDisplayColor: displayColor,
		}
		// Start timer
		start := time.Now()
		// URL
		path := r.URL.Path
		// Query param
		raw := r.URL.RawQuery
		next(ctx)
		stop := time.Now()
		latency := stop.Sub(start)
		// get query ip address
		ip, _, _ := net.SplitHostPort(strings.TrimSpace(ctx.R.RemoteAddr))
		clientIP := net.ParseIP(ip)
		// query method
		method := r.Method
		// query status code
		statusCode := ctx.StatusCode

		// log display middleware
		if raw != "" {
			path = path + "?" + raw
		}
		param.ClientIP = clientIP
		param.TimeStamp = stop
		param.Latency = latency
		param.StatusCode = statusCode
		param.Method = method
		param.Path = path
		fmt.Fprintf(out, formatter(param))
	}
}

// Logger add middleware method
func Logger(next HandleFunc) HandleFunc {
	return LoggerWithConfig(LoggerConfig{}, next)
}
