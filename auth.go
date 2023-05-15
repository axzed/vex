package vex

import (
	"encoding/base64"
	"net/http"
)

type Accounts struct {
	UnAuthHandler func(ctx *Context)
	Users         map[string]string
}

// BasicAuth is the basic auth handler. It returns a 401 Unauthorized if the request is not authorized
func (a *Accounts) BasicAuth(next HandleFunc) HandleFunc {
	return func(ctx *Context) {
		// get the Authorization from header (base64)
		//判断请求中是否有Authorization的Header
		username, password, ok := ctx.R.BasicAuth()
		if !ok {
			a.UnAuthHandlers(ctx)
			return
		}
		pwd, ok := a.Users[username]
		if !ok {
			a.UnAuthHandlers(ctx)
			return
		}
		if pwd != password {
			a.UnAuthHandlers(ctx)
			return
		}
		ctx.Set("user", username)
		next(ctx)
	}
}

// UnAuthHandlers is the default unauthorized handler. It sends a 401 response
func (a *Accounts) UnAuthHandlers(ctx *Context) {
	if a.UnAuthHandler != nil {
		a.UnAuthHandler(ctx)
	} else {
		ctx.W.WriteHeader(http.StatusUnauthorized)
	}
}

// BasicAuth returns the base64 encoding of username and password
func BasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
