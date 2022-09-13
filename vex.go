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

// Routing groups
type routerGroup struct {
	name             string                           // Router group's name
	handleFuncMap    map[string]map[string]HandleFunc // Each routing group's handler's function
	handlerMethodMap map[string][]string              // Support different request methods && its urls (store different request method type)
	treeNode         *treeNode
}

// handle use this function to set the HandleFunc of the mapping url
func (r *routerGroup) handle(name string, method string, handleFunc HandleFunc) {
	_, ok := r.handleFuncMap[name]
	if !ok {
		r.handleFuncMap[name] = make(map[string]HandleFunc)
	}
	_, ok = r.handleFuncMap[name][method]
	if ok {
		panic("With duplicate routes")
	}
	r.handleFuncMap[name][method] = handleFunc
	r.treeNode.Put(name)
}

//	Any Get Post Put Delete is restful api
//
// Any is a method support any type of request to our router
func (r *routerGroup) Any(name string, handleFunc HandleFunc) {
	r.handle(name, ANY, handleFunc)
}

func (r *routerGroup) Get(name string, handleFunc HandleFunc) {
	r.handle(name, http.MethodGet, handleFunc)
}

func (r *routerGroup) Post(name string, handleFunc HandleFunc) {
	r.handle(name, http.MethodPost, handleFunc)
}

func (r *routerGroup) Delete(name string, handleFunc HandleFunc) {
	r.handle(name, http.MethodDelete, handleFunc)
}

func (r *routerGroup) Put(name string, handleFunc HandleFunc) {
	r.handle(name, http.MethodPut, handleFunc)
}

func (r *routerGroup) Patch(name string, handleFunc HandleFunc) {
	r.handle(name, http.MethodPatch, handleFunc)
}

func (r *routerGroup) Options(name string, handleFunc HandleFunc) {
	r.handle(name, http.MethodOptions, handleFunc)
}

func (r *routerGroup) Head(name string, handleFunc HandleFunc) {
	r.handle(name, http.MethodHead, handleFunc)
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
		name:             name,
		handleFuncMap:    make(map[string]map[string]HandleFunc),
		handlerMethodMap: make(map[string][]string),
		treeNode:         &treeNode{name: "/", children: make([]*treeNode, 0)},
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
				handle(ctx)
				return
			}
			handle, ok = group.handleFuncMap[node.routerName][method]
			if ok {
				handle(ctx)
				return
			}
			// url matched but not in a correct method return 405
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "%s %s not allowed\n", r.RequestURI, method)
			return
		}
		//for name, methodHandle := range group.handleFuncMap {
		//	url := group.name + name
		//	// url match
		//	if r.RequestURI == url {
		//
		//	}
		//}
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
