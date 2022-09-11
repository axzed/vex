// Copyright 2022 Xue WenChao. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package vex

import (
	"log"
	"net/http"
)

// HandleFunc defines the handler used by vex middleware as return value.
type HandleFunc func(w http.ResponseWriter, r *http.Request)

// Routing groups
type routerGroup struct {
	name          string                // router group's name
	handleFuncMap map[string]HandleFunc // Each routing group's handler's function
}

// Add routers in a same group
// name is router's related paths, handleFunc is the function to process the corresponding route
func (r *routerGroup) Add(name string, handleFunc HandleFunc) {
	r.handleFuncMap[name] = handleFunc
}

// router defines a routerGroup info
type router struct {
	routerGroups []*routerGroup // router's group
}

// Group grouping the routes
func (r *router) Group(name string) *routerGroup {
	routerGroup := &routerGroup{
		name:          name,
		handleFuncMap: make(map[string]HandleFunc),
	}
	r.routerGroups = append(r.routerGroups, routerGroup)
	return routerGroup
}

// Engine is the framework's instance, it contains the muxer, middleware and configuration settings.
// Create an instance of Engine, by using New() or Default().
type Engine struct {
	router
}

// New returns a new blank Engine instance without any middleware attached.
func New() *Engine {
	return &Engine{}
}

// Run attaches the router to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (e *Engine) Run() {
	for _, group := range e.routerGroups {
		for key, value := range group.handleFuncMap {
			http.HandleFunc(group.name+key, value)
		}
	}
	err := http.ListenAndServe(":8111", nil)
	if err != nil {
		log.Fatal(err)
	}
}
