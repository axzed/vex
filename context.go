// Copyright 2022 Xue WenChao. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package vex

import (
	"encoding/json"
	"encoding/xml"
	"html/template"
	"net/http"
)

// Context is the most important part of gin. It allows us to pass variables between middleware,
// manage the flow, validate the JSON of a request and render a JSON response for example
type Context struct {
	W      http.ResponseWriter
	R      *http.Request
	engine *Engine
}

// HTML Render the HTML files to request
// it return pure HTML files, don't need any data
func (c *Context) HTML(status int, html string) error {
	// Default status 200
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.W.WriteHeader(status)
	_, err := c.W.Write([]byte(html))
	return err
}

// HTMLTemplate is the function to render the HTML Template
// return HTML template files with data
func (c *Context) HTMLTemplate(name string, data any, filename ...string) error {
	// Default status 200
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	t, err := t.ParseFiles(filename...)
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}

// HTMLTemplate is the function to render the HTML Template you set
func (c *Context) HTMLTemplateGlob(name string, data any, pattern string) error {
	// Default status 200
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	t, err := t.ParseGlob(pattern)
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}

// Template set the content to the memory and load all HTML template files to system
func (c *Context) Template(name string, data any) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := c.engine.HTMLRender.Template.ExecuteTemplate(c.W, name, data)
	return err
}

// JSON serializes the given struct as JSON into the response body.
// It also sets the Content-Type as "application/json".
func (c *Context) JSON(status int, data any) error {
	c.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.W.WriteHeader(status)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = c.W.Write(jsonData)
	return err
}

// XML serializes the given struct as XML into the response body.
// It also sets the Content-Type as "application/xml".
func (c *Context) XML(status int, data any) error {
	c.W.Header().Set("Content-Type", "application/xml; charset=utf-8")
	c.W.WriteHeader(status)
	err := xml.NewEncoder(c.W).Encode(data)
	return err
}
