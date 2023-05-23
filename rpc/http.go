package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

const (
	HTTP  = "http"
	HTTPS = "https"
)

const (
	GET      = "GET"
	POSTForm = "POST_FORM"
	POSTJson = "POST_JSON"
)

// VexHttpClient is a http client for vex
type VexHttpClient struct {
	client     http.Client
	serviceMap map[string]VexService
}

// HttpConfig is a struct for http config
type HttpConfig struct {
	Protocol string
	Host     string
	Port     int
}

// VexService is an interface for vex service
// when you want to use vex service, you should implement this interface
type VexService interface {
	Env() HttpConfig
}

// VexHttpClientSession is a http client session for vex
// this session confirms the http client and request handler
type VexHttpClientSession struct {
	*VexHttpClient
	ReqHandler func(req *http.Request)
}

// NewHttpClient returns a new VexHttpClient
func NewHttpClient() *VexHttpClient {
	// http.Transport 请求分发 协程安全 连接池
	// 同一Transport对象应该被多个goroutine共享，
	// 并且应该在程序的生命周期内只创建一次，而不是每次发起请求都重新创建一个Transport。
	// 这样可以提高效率，并且可以避免某些网络问题。
	client := http.Client{
		Timeout: time.Duration(3) * time.Second,
		// 设置Transport 用于设置连接池
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   5,                // 每个host最大空闲连接
			MaxConnsPerHost:       100,              // 每个host最大连接数
			IdleConnTimeout:       90 * time.Second, // 空闲连接超时时间
			TLSHandshakeTimeout:   10 * time.Second, // tls握手超时时间
			ExpectContinueTimeout: 1 * time.Second,  // 100-continue 超时时间
		},
	}
	return &VexHttpClient{
		client:     client,
		serviceMap: make(map[string]VexService),
	}
}

// GetRequest to construct a http request
func (c *VexHttpClient) GetRequest(method string, url string, args map[string]any) (*http.Request, error) {
	// 如果有参数，拼接到 url 后面 ?a=1&b=2
	if args != nil && len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// FormRequest to construct a http request with form
func (c *VexHttpClient) FormRequest(method string, url string, args map[string]any) (*http.Request, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return req, nil
}

// JsonRequest to construct a http request with json
func (c *VexHttpClient) JsonRequest(method string, url string, args map[string]any) (*http.Request, error) {
	jsonStr, _ := json.Marshal(args)
	req, err := http.NewRequest(method, url, bytes.NewReader(jsonStr))
	if err != nil {
		return nil, err
	}
	return req, nil
}

// Response to send a http request and get response
func (c *VexHttpClientSession) Response(req *http.Request) ([]byte, error) {
	return c.responseHandle(req)
}

// Get to send a get request and get response
func (c *VexHttpClientSession) Get(url string, args map[string]any) ([]byte, error) {
	//get请求的参数 url?
	if args != nil && len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	log.Println(url)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.responseHandle(request)
}

// PostForm to send a post request with form and get response
func (c *VexHttpClientSession) PostForm(url string, args map[string]any) ([]byte, error) {
	request, err := http.NewRequest("POST", url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return c.responseHandle(request)
}

// PostJson to send a post request with json and get response
func (c *VexHttpClientSession) PostJson(url string, args map[string]any) ([]byte, error) {
	marshal, _ := json.Marshal(args)
	request, err := http.NewRequest("POST", url, bytes.NewReader(marshal))
	if err != nil {
		return nil, err
	}
	return c.responseHandle(request)
}

// responseHandle to handle response
func (c *VexHttpClientSession) responseHandle(request *http.Request) ([]byte, error) {
	c.ReqHandler(request)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	// 当请求返回的状态码不是200时，返回错误信息
	if response.StatusCode != http.StatusOK {
		info := fmt.Sprintf("response status is %d", response.StatusCode)
		return nil, errors.New(info)
	}
	// 读取返回的body
	reader := bufio.NewReader(response.Body)
	defer response.Body.Close()

	var buf = make([]byte, 127)
	var body []byte
	// 循环读取body
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF || n == 0 {
			break
		}
		body = append(body, buf[:n]...)
		if n < len(buf) {
			break
		}
	}
	return body, nil

}

// toValues converts map to url.Values
func (c *VexHttpClient) toValues(args map[string]any) string {
	// 如果有参数，拼接到 url 后面 ?a=1&b=2
	if args != nil && len(args) > 0 {
		params := url.Values{}
		for k, v := range args {
			params.Set(k, fmt.Sprintf("%v", v))
		}
		return params.Encode()
	}
	return ""
}

// RegisterHttpService to register a service
func (c *VexHttpClient) RegisterHttpService(name string, service VexService) {
	// 如果已经注册过了，就不再注册
	c.serviceMap[name] = service
}

// Session to get a session
func (c *VexHttpClient) Session() *VexHttpClientSession {
	return &VexHttpClientSession{
		c, nil,
	}
}

// Do to do a service
func (c *VexHttpClientSession) Do(service string, method string) VexService {
	msService, ok := c.serviceMap[service]
	if !ok {
		panic(errors.New("service not found"))
	}
	//找到service里面的Field 给其中要调用的方法 赋值
	t := reflect.TypeOf(msService)
	v := reflect.ValueOf(msService)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("service not pointer"))
	}
	tVar := t.Elem()
	vVar := v.Elem()
	fieldIndex := -1
	for i := 0; i < tVar.NumField(); i++ {
		name := tVar.Field(i).Name
		if name == method {
			fieldIndex = i
			break
		}
	}
	if fieldIndex == -1 {
		panic(errors.New("method not found"))
	}
	tag := tVar.Field(fieldIndex).Tag
	rpcInfo := tag.Get("msrpc")
	if rpcInfo == "" {
		panic(errors.New("not msrpc tag"))
	}
	split := strings.Split(rpcInfo, ",")
	if len(split) != 2 {
		panic(errors.New("tag msrpc not valid"))
	}
	methodType := split[0]
	path := split[1]
	httpConfig := msService.Env()

	f := func(args map[string]any) ([]byte, error) {
		if methodType == GET {
			return c.Get(httpConfig.Prefix()+path, args)
		}
		if methodType == POSTForm {
			return c.PostForm(httpConfig.Prefix()+path, args)
		}
		if methodType == POSTJson {
			return c.PostJson(httpConfig.Prefix()+path, args)
		}
		return nil, errors.New("no match method type")
	}
	fValue := reflect.ValueOf(f)
	vVar.Field(fieldIndex).Set(fValue)
	return msService
}

// Prefix to get prefix
func (c HttpConfig) Prefix() string {
	if c.Protocol == "" {
		c.Protocol = HTTP
	}
	switch c.Protocol {
	case HTTP:
		return fmt.Sprintf("http://%s:%d", c.Host, c.Port)
	case HTTPS:
		return fmt.Sprintf("https://%s:%d", c.Host, c.Port)
	}
	return ""
}
