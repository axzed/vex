// Copyright 2022 Xue WenChao. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package vex

import (
	"errors"
	"github.com/axzed/vex/render"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var defaultMaxMemory = 32 << 20 // 32M

// Context is the most important part of gin. It allows us to pass variables between middleware,
// manage the flow, validate the JSON of a request and render a JSON response for example
type Context struct {
	W          http.ResponseWriter // response
	R          *http.Request       // request
	engine     *Engine             // Context's engine
	queryCache url.Values          // handle the query of url
	formCache  url.Values          // handle the query by HTML post
}

// initQueryCache get the query param in request url
func (c *Context) initQueryCache() {
	if c.R != nil {
		c.queryCache = c.R.URL.Query()
	} else {
		c.queryCache = url.Values{}
	}
}

// DefaultQuery if you have not set the key return the default value
func (c *Context) DefaultQuery(key, defaultValue string) string {
	values, ok := c.GetAllQuery(key)
	if !ok {
		return defaultValue
	}
	return values[0]
}

// GetQuery if url is ?key:value you want to get value by using key
func (c *Context) GetQuery(key string) string {
	c.initQueryCache()
	return c.queryCache.Get(key)
}

// GetAllQuery if you want to all the url query param by using key like ?id=1&id=2 return [1, 2]
func (c *Context) GetAllQuery(key string) ([]string, bool) {
	c.initQueryCache()
	values, ok := c.queryCache[key]
	return values, ok
}

// QueryArray return query param without check
func (c *Context) QueryArray(key string) (values []string) {
	c.initQueryCache()
	values, _ = c.queryCache[key]
	return values
}

// get to get the url's mapping param
func (c *Context) get(cache map[string][]string, key string) (map[string]string, bool) {
	dicts := make(map[string]string)
	exist := false
	for k, value := range cache {
		if i := strings.IndexByte(k, '['); i >= 1 && k[0:i] == key {
			if j := strings.IndexByte(k[i+1:], ']'); j >= 1 {
				exist = true
				dicts[k[i+1:][:j]] = value[0]
			}
		}
	}
	return dicts, exist
}

// GetQueryMap get the mapping
func (c *Context) GetQueryMap(key string) (map[string]string, bool) {
	c.initQueryCache()
	return c.get(c.queryCache, key)
}

// QueryMap get the query map without check
func (c *Context) QueryMap(key string) (dicts map[string]string) {
	dicts, _ = c.GetQueryMap(key)
	return
}

// initPostFormCache init the post form param
func (c *Context) initPostFormCache() {
	if c.R != nil {
		err := c.R.ParseMultipartForm(int64(defaultMaxMemory))
		if err != nil {
			if errors.Is(err, http.ErrNotMultipart) {
				log.Println(err)
			}
		}
		c.formCache = c.R.PostForm
	} else {
		c.formCache = url.Values{}
	}
}

// GetPostArray get the post form data list
func (c *Context) GetPostArray(key string) (values []string, ok bool) {
	c.initPostFormCache()
	values, ok = c.formCache[key]
	return
}

// GetPostArray get the post form data list without check
func (c *Context) PostArray(key string) (values []string) {
	values, _ = c.GetPostArray(key)
	return
}

// GetPost get the post form data
func (c *Context) GetPost(key string) (string, bool) {
	values, ok := c.GetPostArray(key)
	if ok {
		return values[0], ok
	}
	return "", false
}

// Get the post form's map
// GetPostMap get the post form map
func (c *Context) GetPostMap(key string) (map[string]string, bool) {
	c.initQueryCache()
	return c.get(c.formCache, key)
}

// PostMap get the post form map without check
func (c *Context) PostMap(key string) (dicts map[string]string) {
	dicts, _ = c.GetPostMap(key)
	return
}

// HTML Render the HTML files to request
// it return pure HTML files, don't need any data
func (c *Context) HTML(status int, html string) error {
	return c.Render(status, &render.HTML{
		Data:       html,
		IsTemplate: false,
	})
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
	return c.Render(http.StatusOK, &render.HTML{
		Data:       data,
		IsTemplate: true,
		Template:   c.engine.HTMLRender.Template,
		Name:       name,
	})
}

// JSON serializes the given struct as JSON into the response body.
// It also sets the Content-Type as "application/json".
func (c *Context) JSON(status int, data any) error {
	return c.Render(status, &render.JSON{Data: data})
}

// XML serializes the given struct as XML into the response body.
// It also sets the Content-Type as "application/xml".
func (c *Context) XML(status int, data any) error {
	return c.Render(status, &render.XML{
		Data: data,
	})
}

// File writes the specified file into the body stream in an efficient way.
func (c *Context) File(fileName string) {
	http.ServeFile(c.W, c.R, fileName)
}

// FileAttachment writes the specified file into the body stream in an efficient way
// On the client side, the file will typically be downloaded with the given filename
func (c *Context) FileAttachment(filepath, filename string) {
	if isASCII(filename) {
		c.W.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	} else {
		c.W.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
	}
	http.ServeFile(c.W, c.R, filepath)
}

// FileFromFS writes the specified file from http.FileSystem into the body stream in an efficient way.
func (c *Context) FileFromFS(filepath string, fs http.FileSystem) {
	defer func(old string) {
		c.R.URL.Path = old
	}(c.R.URL.Path)

	c.R.URL.Path = filepath

	http.FileServer(fs).ServeHTTP(c.W, c.R)
}

func (c *Context) Redirect(status int, url string) error {
	return c.Render(status, &render.Redirect{
		Code:     status,
		Request:  c.R,
		Location: url,
	})
}

// String return c.String()
func (c *Context) String(status int, format string, values ...any) error {
	return c.Render(status, &render.String{Format: format, Data: values})
}

func (c *Context) Render(statusCode int, r render.Render) error {
	err := r.Render(c.W)
	if statusCode != http.StatusOK {
		c.W.WriteHeader(statusCode)
	}
	return err
}
