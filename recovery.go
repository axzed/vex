package vex

import (
	"errors"
	"fmt"
	"github.com/axzed/vex/verror"
	"net/http"
	"runtime"
	"strings"
)

// detailMsg is a method to print stack & heap info
func detailMsg(err any) string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:])
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%v\n", err))
	for _, pc := range pcs[0:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		sb.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return sb.String()
}

// Recovery is a middleware function to handle the panic
func Recovery(next HandleFunc) HandleFunc {
	return func(ctx *Context) {
		// exec the recover logic
		defer func() {
			if err := recover(); err != nil {
				err2 := err.(error)
				if err2 != nil {
					var vError *verror.VError
					if errors.As(err2, &vError) {
						vError.ExecResult()
						return
					}
				}
				// print the error's log to console
				// 裸的err Msg
				// ctx.Logger.Error(err)
				// 通过detail函数包装堆栈信息
				ctx.Logger.Error(detailMsg(err))
				ctx.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		next(ctx)
	}
}
