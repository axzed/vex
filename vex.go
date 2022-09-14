// Copyright 2022 Xue WenChao. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package vex

import (
	"fmt"
	"log"
	"net/http"
)

// ANY is the other name of "ANY" means url use ANY method
const ANY = "ANY"

// HandleFunc defines the handler used by vex middleware as return value.
// Context is the wrap of (w *http.ResponseWriter, r http.Request)
type HandleFunc func(ctx *Context)

// MiddlewareFunc is code before or after HandleFunc
// input the handleFunc before process then return the handle Func which after process
type MiddlewareFunc func(handleFunc HandleFunc) HandleFunc

// Routing groups
type routerGroup struct {
	name               string                                 // Router group's name
	handleFuncMap      map[string]map[string]HandleFunc       // Each routing group's handler's function
	middlewaresFuncMap map[string]map[string][]MiddlewareFunc // Each routing group's middlewaresFunction's function
	handlerMethodMap   map[string][]string                    // Support different request methods && its urls (store different request method type)
	treeNode           *treeNode                              // prefix router match tree
	middlewares        []MiddlewareFunc                       // middlewares function list
}

// Use function to add Middleware to the handleFunc
// ... means you can add multi middleware to the func
func (r *routerGroup) Use(middlewareFunc ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewareFunc...)
}

// methodHandle is the function when you need to executive the middleware in request
func (r *routerGroup) methodHandle(name string, method string, handleFunc HandleFunc, ctx *Context) {
	// if you have set the Middleware
	// exec Middleware
	// common level middleware
	if r.middlewares != nil {
		for _, middlewareFunc := range r.middlewares {
			handleFunc = middlewareFunc(handleFunc)
		}
	}
	// middlewareFuncs is router's middlewares function you set
	middlewareFuncs := r.middlewaresFuncMap[name][method]
	// exec the routerLevel middlewares
	if middlewareFuncs != nil {
		for _, middlewareFunc := range middlewareFuncs {
			handleFunc = middlewareFunc(handleFunc)
		}
	}
	// exec handleFunc you set in api
	handleFunc(ctx)
}

// handle use this function to set the HandleFunc and middlewares into the mapping url
func (r *routerGroup) handle(name string, method string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	// use group's name to init handleFunc and middlewares list
	_, ok := r.handleFuncMap[name]
	// init the function of this group of routes
	if !ok {
		r.handleFuncMap[name] = make(map[string]HandleFunc)
		r.middlewaresFuncMap[name] = make(map[string][]MiddlewareFunc)
	}
	_, ok = r.handleFuncMap[name][method]
	if ok {
		panic("With duplicate routes")
	}
	// add the handleFunc for the mapping route
	r.handleFuncMap[name][method] = handleFunc
	// add the middlewaresFunc for the mapping route
	r.middlewaresFuncMap[name][method] = append(r.middlewaresFuncMap[name][method], middlewareFunc...)
	// set the prefix tree's root node
	r.treeNode.Put(name)
}

// Any Get Post Put Delete is restful api
// Any is a method support any type of request to our router
func (r *routerGroup) Any(name string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, ANY, handleFunc, middlewareFunc...)
}

func (r *routerGroup) Get(name string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodGet, handleFunc, middlewareFunc...)
}

func (r *routerGroup) Post(name string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPost, handleFunc, middlewareFunc...)
}

func (r *routerGroup) Delete(name string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodDelete, handleFunc, middlewareFunc...)
}

func (r *routerGroup) Put(name string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPut, handleFunc, middlewareFunc...)
}

func (r *routerGroup) Patch(name string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPatch, handleFunc, middlewareFunc...)
}

func (r *routerGroup) Options(name string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodOptions, handleFunc, middlewareFunc...)
}

func (r *routerGroup) Head(name string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodHead, handleFunc, middlewareFunc...)
}

// router defines a routerGroup's slice info
type router struct {
	routerGroups []*routerGroup // router's group
}

// Group grouping the routes
// initialize the routerGroups by using the Group function
// take the routerGroup to manipulate the url
func (r *router) Group(name string) *routerGroup {
	routerGroup := &routerGroup{
		name:               name,
		handleFuncMap:      make(map[string]map[string]HandleFunc),
		middlewaresFuncMap: make(map[string]map[string][]MiddlewareFunc),
		handlerMethodMap:   make(map[string][]string),
		treeNode:           &treeNode{name: "/", children: make([]*treeNode, 0)},
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
	e.httpRequestHandle(w, r)
}

// httpRequestHandle is a function to handle the router's request
func (e *Engine) httpRequestHandle(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	for _, group := range e.routerGroups {
		routerName := SubStringLast(r.RequestURI, group.name)
		// get/1
		// node has all routerName match to change the dynamic url like :id ---> 1
		node := group.treeNode.Get(routerName)
		// match
		// node.isEnd means this tree node is at the end of url
		// ps: if node is end of url then you url has not in a same method, so return 405
		// ps: if node is not the end means this node is not the end you need to return 404
		if node != nil && node.isEnd {
			ctx := &Context{
				W: w,
				R: r,
			}
			handle, ok := group.handleFuncMap[node.routerName][ANY]
			if ok {
				group.methodHandle(node.routerName, ANY, handle, ctx)
				return
			}
			handle, ok = group.handleFuncMap[node.routerName][method]
			if ok {
				group.methodHandle(node.routerName, method, handle, ctx)
				return
			}
			// url matched but not in a correct method return 405
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "%s %s not allowed\n", r.RequestURI, method)
			return
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
	http.Handle("/", e)
	err := http.ListenAndServe(":8111", nil)
	if err != nil {
		log.Fatal(err)
	}
}
