package nxjgo

import (
	"encoding/base64"
	"net/http"
)

type Accounts struct {
	UnAuthHandler func(ctx *Context)
	Users         map[string]string
}

func (a *Accounts) BasicAuth(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		username, password, ok := ctx.R.BasicAuth()
		if !ok {
			a.unAuthHandlers(ctx)
			return
		}
		pwd, ok := a.Users[username]
		if !ok {
			a.unAuthHandlers(ctx)
			return
		}
		if pwd != password {
			a.unAuthHandlers(ctx)
			return
		}
		ctx.Set("user", username)
		next(ctx)
	}
}

func (a *Accounts) unAuthHandlers(ctx *Context) {
	if a.UnAuthHandler != nil {
		a.UnAuthHandler(ctx)
	} else {
		ctx.W.WriteHeader(http.StatusUnauthorized)
	}
}

func BasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
