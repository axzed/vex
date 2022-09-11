// Copyright 2022 Xue WenChao. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package vex

import (
	"fmt"
	"log"
	"net/http"
)

// HandleFunc defines the handler used by vex middleware as return value.
type HandleFunc func(w http.ResponseWriter, r *http.Request)

// Routing groups
type routerGroup struct {
	name             string                // Router group's name
	handleFuncMap    map[string]HandleFunc // Each routing group's handler's function
	handlerMethodMap map[string][]string   // Support different request methods && its urls (store different request method type)
}

// Add routers in a same group
// name is router's related paths, handleFunc is the function to process the corresponding route
func (r *routerGroup) Add(name string, handleFunc HandleFunc) {
	r.handleFuncMap[name] = handleFunc
}

//	Any Get Post Put Delete is restful api
//
// Any is a method support any type of request to our router
func (r *routerGroup) Any(name string, handleFunc HandleFunc) {
	r.handleFuncMap[name] = handleFunc
	r.handlerMethodMap["ANY"] = append(r.handlerMethodMap["ANY"], name)
}

func (r *routerGroup) Get(name string, handleFunc HandleFunc) {
	r.handleFuncMap[name] = handleFunc
	r.handlerMethodMap[http.MethodGet] = append(r.handlerMethodMap[http.MethodGet], name)
}

func (r *routerGroup) Post(name string, handleFunc HandleFunc) {
	r.handleFuncMap[name] = handleFunc
	r.handlerMethodMap[http.MethodPost] = append(r.handlerMethodMap[http.MethodPost], name)
}

// router defines a routerGroup's slice info
type router struct {
	routerGroups []*routerGroup // router's group
}

// Group grouping the routes
func (r *router) Group(name string) *routerGroup {
	routerGroup := &routerGroup{
		name:             name,
		handleFuncMap:    make(map[string]HandleFunc),
		handlerMethodMap: make(map[string][]string),
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

// implement the interface method ServeHTTP
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	for _, group := range e.routerGroups {
		for name, methodHandle := range group.handleFuncMap {
			url := group.name + name
			// url match
			if r.RequestURI == url {
				routers, ok := group.handlerMethodMap["ANY"]
				if ok {
					// handle router
					for _, routerName := range routers {
						if routerName == name {
							methodHandle(w, r)
							return
						}
					}
				}
				// method compare
				routers, ok = group.handlerMethodMap[method]
				if ok {
					for _, routerName := range routers {
						if routerName == name {
							methodHandle(w, r)
							return
						}
					}
				}
				// url matched but not in a correct method return 405
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprintf(w, "%s %s not allowed\n", r.RequestURI, method)
				return
			}
		}
	}
	// if url is not match return 404
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "%s not found\n", r.RequestURI)
}

// Run attaches the router to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (e *Engine) Run() {
	//for _, group := range e.routerGroups {
	//	for key, value := range group.handleFuncMap {
	//		http.HandleFunc(group.name+key, value)
	//	}
	//}
	http.Handle("/", e)
	err := http.ListenAndServe(":8111", nil)
	if err != nil {
		log.Fatal(err)
	}
}
