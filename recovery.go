package vex

import "net/http"

// Recovery is a middleware function to handle the panic
func Recovery(next HandleFunc) HandleFunc {
	return func(ctx *Context) {
		// exec the recover logic
		defer func() {
			if err := recover(); err != nil {
				// print the error's log to console
				ctx.Logger.Error(err)
				ctx.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		next(ctx)
	}
}
