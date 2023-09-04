package nxjgo

import (
	"errors"
	"fmt"
	"github.com/Komorebi695/nxjgo/nxjerror"
	"net/http"
	"runtime"
	"strings"
)

func detailMsg(err any) string {
	var sb strings.Builder
	pcs := make([]uintptr, 32)
	n := runtime.Callers(3, pcs[:])
	sb.WriteString(fmt.Sprintf("%v\n", err))
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		sb.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return sb.String()
}

func Recovery(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				err2 := err.(error)
				var nxjError *nxjerror.NxjError
				if errors.As(err2, &nxjError) {
					nxjError.ExecResult()
					return
				}
				ctx.Logger.Error(detailMsg(err))
				ctx.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		next(ctx)
	}
}
