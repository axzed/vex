package vex

import (
	"log"
	"net"
	"strings"
	"time"
)

type LoggerConfig struct {
}

// LoggerWithConfig init the logger with the configuration
func LoggerWithConfig(conf LoggerConfig, next HandleFunc) HandleFunc {
	return func(ctx *Context) {
		r := ctx.R
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
		log.Printf("[VEX] %v | %3d | %13v | %15s |%-7s %#v",
			stop.Format("2006/01/02 - 15:04:05"),
			statusCode, latency, clientIP, method, path,
		)
	}
}

// Logger add middleware method
func Logger(next HandleFunc) HandleFunc {
	return LoggerWithConfig(LoggerConfig{}, next)
}
