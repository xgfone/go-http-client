// Copyright 2021 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// Pre-define some constants.
const (
	HeaderAccept           = "Accept"
	HeaderAcceptedLanguage = "Accept-Language"
	HeaderAcceptEncoding   = "Accept-Encoding"
	HeaderAuthorization    = "Authorization"
	HeaderUserAgent        = "User-Agent"

	MIMEMultipartForm   = "multipart/form-data"
	MIMEApplicationForm = "application/x-www-form-urlencoded"
	MIMEApplicationXML  = "application/xml; charset=UTF-8"
	MIMEApplicationJSON = "application/json; charset=UTF-8"
)

var bufpool = sync.Pool{New: func() interface{} {
	return bytes.NewBuffer(make([]byte, 0, 1024))
}}

func getBuffer() *bytes.Buffer    { return bufpool.Get().(*bytes.Buffer) }
func putBuffer(buf *bytes.Buffer) { buf.Reset(); bufpool.Put(buf) }

type bytesT struct{ Data []byte }

var bytespool = sync.Pool{New: func() interface{} {
	return &bytesT{Data: make([]byte, 1024)}
}}

func getBytes() *bytesT  { return bytespool.Get().(*bytesT) }
func putBytes(b *bytesT) { bytespool.Put(b) }

type discarder struct{}

func (d discarder) Write(p []byte) (int, error) { return len(p), nil }

// CloseBody closes the io reader, which will discard all the data from r
// before closing it, but does nothing if r is nil
func CloseBody(r io.ReadCloser) (err error) {
	if r != nil {
		buf := getBytes()
		io.CopyBuffer(discarder{}, r, buf.Data[:])
		err = r.Close()
		putBytes(buf)
	}
	return
}

func cloneQuery(query url.Values) url.Values {
	return url.Values(cloneHeader(http.Header(query)))
}

// GetContentType returns the Content-Type from the header,
// which will remove the charset part.
func GetContentType(header http.Header) string {
	ct := header.Get("Content-Type")
	if index := strings.IndexAny(ct, ";"); index > 0 {
		ct = strings.TrimSpace(ct[:index])
	}
	return ct
}

// EncodeData encodes the data by contentType and writes it into w.
//
// It writes the data into w directly instead of encoding it
// if contentType is one of the types:
//   - []byte
//   - string
//   - io.Reader
//   - io.WriterTo
//
func EncodeData(w io.Writer, contentType string, data interface{}) (err error) {
	switch v := data.(type) {
	case *bytes.Buffer:
		_, err = w.Write(v.Bytes())
	case io.Reader:
		_, err = io.CopyBuffer(w, v, make([]byte, 1024))
	case []byte:
		_, err = w.Write(v)
	case string:
		_, err = io.WriteString(w, v)
	case io.WriterTo:
		_, err = v.WriteTo(w)
	default:
		switch contentType {
		case "":
			err = errors.New("no request header Content-Type")
		case "application/xml":
			err = xml.NewEncoder(w).Encode(data)
		case "application/json":
			err = json.NewEncoder(w).Encode(data)
		default:
			err = fmt.Errorf("unsupported request Content-Type '%s'", contentType)
		}
	}
	return
}

// DecodeFromReader reads the data from r and decode it to dst.
//
// If ct is equal to "application/xml" or "application/json", it will use
// the xml or json decoder to decode the data. Or returns an error.
func DecodeFromReader(dst interface{}, ct string, r io.Reader) (err error) {
	if dst != nil {
		switch ct {
		case "":
			err = errors.New("no response header Content-Type")
		case "application/xml":
			err = xml.NewDecoder(r).Decode(dst)
		case "application/json":
			err = json.NewDecoder(r).Decode(dst)
		default:
			err = fmt.Errorf("unsupported response Content-Type '%s'", ct)
		}
	}
	return
}

// DecodeResponseBody is a response handler to decode the response body
// into dst.
func DecodeResponseBody(dst interface{}, resp *http.Response) (err error) {
	return DecodeFromReader(dst, GetContentType(resp.Header), resp.Body)
}

// ReadResponseBodyAsError is a response handler to read the response body
// as the error to be returned.
func ReadResponseBodyAsError(dst interface{}, resp *http.Response) (err error) {
	err = fmt.Errorf("got status code %d", resp.StatusCode)
	e := Error{Code: resp.StatusCode, Err: err}
	if req := resp.Request; req != nil {
		e.Method = req.Method
		e.URL = req.URL.String()
	}

	buf := getBuffer()
	io.CopyBuffer(buf, resp.Body, make([]byte, 256))
	e.Data = buf.String()
	putBuffer(buf)

	return e
}

// Hook is a hook to wrap and modify the http request.
type Hook interface {
	Request(*http.Request) *http.Request
}

// Hooks is a set of hooks.
type Hooks []Hook

// Request implements the interface Hook.
func (hs Hooks) Request(r *http.Request) *http.Request {
	for _, hook := range hs {
		r = hook.Request(r)
	}
	return r
}

// HookFunc is a hook function.
type HookFunc func(*http.Request) *http.Request

// Request implements the interface Hook.
func (f HookFunc) Request(r *http.Request) *http.Request { return f(r) }

type (
	// Encoder is used to encode the data by the content type to dst.
	Encoder func(dst io.Writer, contentType string, data interface{}) error

	// Decoder is used to decode the data by the content type to dst.
	Decoder func(dst interface{}, contentType string, data io.Reader) error

	// Handler is used to handle the response.
	Handler func(dst interface{}, resp *http.Response) error
)

type respHandler struct {
	All  Handler
	H1xx Handler
	H2xx Handler
	H3xx Handler
	H4xx Handler
	H5xx Handler
}

// Client is a http client to build a request and parse the response.
type Client struct {
	hook    Hook
	query   url.Values
	header  http.Header
	client  *http.Client
	baseurl *url.URL
	encoder Encoder
	handler respHandler

	ignore404 bool
}

// NewClient returns a new Client with the http client.
func NewClient(client *http.Client) *Client {
	c := &Client{
		client:  client,
		query:   make(url.Values, 4),
		header:  make(http.Header, 4),
		encoder: EncodeData,
	}
	c.SetContentType("application/json; charset=UTF-8")
	c.SetResponseHandler2xx(DecodeResponseBody)
	c.SetResponseHandler4xx(ReadResponseBodyAsError)
	c.SetResponseHandler5xx(ReadResponseBodyAsError)
	return c
}

// Clone clones itself to a new one.
func (c *Client) Clone() *Client {
	return &Client{
		client:  c.client,
		query:   cloneQuery(c.query),
		header:  cloneHeader(c.header),
		baseurl: c.baseurl,
		encoder: c.encoder,
		handler: c.handler,

		ignore404: c.ignore404,
	}
}

// Ignore404 sets whether to ignore the status code 404, that's, if true,
// 404 is an error.
//
// Notice: if ignoring 404, the 4xx response handler won't be called
// when the http server returns 404.
//
// Default: false
func (c *Client) Ignore404(ignore bool) *Client {
	c.ignore404 = ignore
	return c
}

// GetHTTPClient returns the inner http client.
func (c *Client) GetHTTPClient() *http.Client {
	return c.client
}

// SetHTTPClient resets the http client.
func (c *Client) SetHTTPClient(client *http.Client) *Client {
	c.client = client
	return c
}

// SetHook resets the request hook.
func (c *Client) SetHook(hook Hook) *Client {
	c.hook = hook
	return c
}

// AddHook appends the request hook.
func (c *Client) AddHook(hook Hook) *Client {
	if hook == nil {
		panic("Client.AddHook: the hook must not be nil")
	}

	switch hooks := c.hook.(type) {
	case nil:
		c.hook = hook
	case Hooks:
		c.hook = append(hooks, hook)
	default:
		c.hook = Hooks{c.hook, hook}
	}

	return c
}

// SetBaseURL sets the default base url.
//
// If baseurl is empty, it will clear the base url.
func (c *Client) SetBaseURL(baseurl string) *Client {
	if baseurl == "" {
		c.baseurl = nil
	} else if u, err := url.Parse(baseurl); err != nil {
		panic(fmt.Errorf("invalid base url '%s'", baseurl))
	} else {
		c.baseurl = u
	}
	return c
}

// AddQueries adds the request queries.
func (c *Client) AddQueries(queries url.Values) *Client {
	for key, values := range queries {
		c.query[key] = values
	}
	return c
}

// AddQueryMap adds the request queries as a map type.
func (c *Client) AddQueryMap(queries map[string]string) *Client {
	for key, value := range queries {
		c.query.Add(key, value)
	}
	return c
}

// AddQuery appends the value for the query key.
func (c *Client) AddQuery(key, value string) *Client {
	c.query.Add(key, value)
	return c
}

// SetQuery sets the query key to the value.
func (c *Client) SetQuery(key, value string) *Client {
	c.query.Set(key, value)
	return c
}

// AddHeaders adds the request headers.
func (c *Client) AddHeaders(headers http.Header) *Client {
	for key, values := range headers {
		c.header[key] = values
	}
	return c
}

// AddHeaderMap adds the request headers as a map type.
func (c *Client) AddHeaderMap(headers map[string]string) *Client {
	for key, value := range headers {
		c.header.Add(key, value)
	}
	return c
}

// AddHeader adds the default request header as "key: value".
func (c *Client) AddHeader(key, value string) *Client {
	c.header.Add(key, value)
	return c
}

// SetHeader sets the default request header as "key: value".
func (c *Client) SetHeader(key, value string) *Client {
	c.header.Set(key, value)
	return c
}

// SetContentType sets the default Content-Type, which is equal to
// SetHeader("Content-Type", ct).
//
// The default Content-Type is "application/json; charset=UTF-8".
func (c *Client) SetContentType(ct string) *Client {
	return c.SetHeader("Content-Type", ct)
}

// SetAccepts resets the accepted types of the response body to accepts.
func (c *Client) SetAccepts(accepts ...string) *Client {
	c.header["Accept"] = accepts
	return c
}

// AddAccept adds the accepted types of the response body, which is equal to
// AddHeader("Accept", contentType).
func (c *Client) AddAccept(contentType string) *Client {
	return c.AddHeader("Accept", contentType)
}

// SetReqBodyEncoder sets the encoder to encode the request body.
//
// The default encoder is EncodeData.
func (c *Client) SetReqBodyEncoder(encode Encoder) *Client {
	c.encoder = encode
	return c
}

// SetResponseHandler sets the handler of the response, which hides
// all the XXX handlers if set, but you can set it to nil to cancel it.
//
// Default: nil
func (c *Client) SetResponseHandler(handler Handler) *Client {
	c.handler.All = handler
	return c
}

// SetResponseHandler1xx sets the handler of the response status code 1xx.
//
// Default: nil
func (c *Client) SetResponseHandler1xx(handler Handler) *Client {
	c.handler.H1xx = handler
	return c
}

// SetResponseHandler2xx sets the handler of the response status code 2xx.
//
// Default: DecodeResponseBody
func (c *Client) SetResponseHandler2xx(handler Handler) *Client {
	c.handler.H2xx = handler
	return c
}

// SetResponseHandler3xx sets the handler of the response status code 3xx.
//
// Default: nil
func (c *Client) SetResponseHandler3xx(handler Handler) *Client {
	c.handler.H3xx = handler
	return c
}

// SetResponseHandler4xx sets the handler of the response status code 4xx.
//
// Default: ReadResponseBodyAsError
func (c *Client) SetResponseHandler4xx(handler Handler) *Client {
	c.handler.H4xx = handler
	return c
}

// SetResponseHandler5xx sets the handler of the response status code 5xx.
//
// Default: ReadResponseBodyAsError
func (c *Client) SetResponseHandler5xx(handler Handler) *Client {
	c.handler.H5xx = handler
	return c
}

// Get is a convenient function, which is equal to Request(http.MethodGet, url).
func (c *Client) Get(url string) *Request {
	return c.Request(http.MethodGet, url)
}

// Put is a convenient function, which is equal to Request(http.MethodPut, url).
func (c *Client) Put(url string) *Request {
	return c.Request(http.MethodPut, url)
}

// Head is a convenient function, which is equal to Request(http.MethodHead, url).
func (c *Client) Head(url string) *Request {
	return c.Request(http.MethodHead, url)
}

// Post is a convenient function, which is equal to Request(http.MethodPost, url).
func (c *Client) Post(url string) *Request {
	return c.Request(http.MethodPost, url)
}

// Patch is a convenient function, which is equal to Request(http.MethodPatch, url).
func (c *Client) Patch(url string) *Request {
	return c.Request(http.MethodPatch, url)
}

// Delete is a convenient function, which is equal to Request(http.MethodDelete, url).
func (c *Client) Delete(url string) *Request {
	return c.Request(http.MethodDelete, url)
}

// Options is a convenient function, which is equal to Request(http.MethodOptions, url).
func (c *Client) Options(url string) *Request {
	return c.Request(http.MethodOptions, url)
}

// Request builds and returns a new request.
func (c *Client) Request(method, requrl string) *Request {
	_url := requrl

	var err error
	if !strings.HasPrefix(requrl, "http") {
		var u *url.URL
		if c.baseurl == nil {
			err = fmt.Errorf("invalid request url '%s'", requrl)
		} else if u, err = url.Parse(requrl); err == nil {
			_url = c.baseurl.ResolveReference(u).String()
		}
	}

	return &Request{
		ignore404: c.ignore404,

		hook:    c.hook,
		encoder: c.encoder,
		handler: c.handler,
		client:  c.client,
		header:  cloneHeader(c.header),
		query:   cloneQuery(c.query),
		method:  method,
		url:     _url,
		err:     err,
	}
}

// Request is a http request.
type Request struct {
	ignore404 bool

	hook    Hook
	hookset bool
	encoder Encoder
	handler respHandler
	reqbody io.Reader
	client  *http.Client
	header  http.Header
	query   url.Values
	method  string
	url     string
	err     error
}

// Ignore404 sets whether to ignore the status code 404, that's, if true,
// 404 is an error.
//
// Notice: if ignoring 404, the 4xx response handler won't be called
// when the http server returns 404.
//
// Default: false
func (r *Request) Ignore404(ignore bool) *Request {
	r.ignore404 = ignore
	return r
}

// SetHook resets the request hook.
func (r *Request) SetHook(hook Hook) *Request {
	r.hookset = true
	r.hook = hook
	return r
}

// AddHook appends the request hook.
func (r *Request) AddHook(hook Hook) *Request {
	if hook == nil {
		panic("Request.AddHook: the hook must not be nil")
	}

	switch hooks := r.hook.(type) {
	case nil:
		r.hook = hook
	case Hooks:
		if r.hookset {
			r.hook = append(hooks, hook)
		} else {
			_len := len(hooks)
			_hooks := make(Hooks, _len+1)
			copy(_hooks, hooks)
			_hooks[_len] = hook
			r.hook = _hooks
		}
	default:
		r.hook = Hooks{r.hook, hook}
	}

	r.hookset = true
	return r
}

// AddQueries adds the request queries.
func (r *Request) AddQueries(queries url.Values) *Request {
	for key, values := range queries {
		r.query[key] = values
	}
	return r
}

// AddQueryMap adds the request queries as a map type.
func (r *Request) AddQueryMap(queries map[string]string) *Request {
	for key, value := range queries {
		r.query.Add(key, value)
	}
	return r
}

// AddQuery appends the value for the query key.
func (r *Request) AddQuery(key, value string) *Request {
	r.query.Add(key, value)
	return r
}

// SetQuery sets the query key to the value.
func (r *Request) SetQuery(key, value string) *Request {
	r.query.Set(key, value)
	return r
}

// AddHeader adds the request header as "key: value".
func (r *Request) AddHeader(key, value string) *Request {
	r.header.Add(key, value)
	return r
}

// AddHeaders adds the request headers.
func (r *Request) AddHeaders(headers http.Header) *Request {
	for key, values := range headers {
		r.header[key] = values
	}
	return r
}

// AddHeaderMap adds the request headers as a map type.
func (r *Request) AddHeaderMap(headers map[string]string) *Request {
	for key, value := range headers {
		r.header.Add(key, value)
	}
	return r
}

// SetHeader adds the request header as "key: value".
func (r *Request) SetHeader(key, value string) *Request {
	r.header.Set(key, value)
	return r
}

// SetContentType sets the default Content-Type, which is equal to
// SetHeader("Content-Type", ct).
func (r *Request) SetContentType(ct string) *Request {
	return r.SetHeader("Content-Type", ct)
}

// SetAccepts resets the accepted types of the response body to accepts.
func (r *Request) SetAccepts(accepts ...string) *Request {
	r.header["Accept"] = accepts
	return r
}

// AddAccept adds the accepted types of the response body, which is equal to
// AddHeader("Accept", contentType).
func (r *Request) AddAccept(contentType string) *Request {
	return r.AddHeader("Accept", contentType)
}

// SetBody sets the body of the request.
func (r *Request) SetBody(body interface{}) *Request {
	if r.err == nil && body != nil {
		buf := getBuffer()
		r.reqbody = buf
		r.err = r.encoder(buf, GetContentType(r.header), body)
	}
	return r
}

// SetReqBodyEncoder sets the encoder to encode the request body.
//
// The default encoder is derived from the client.
func (r *Request) SetReqBodyEncoder(encode Encoder) *Request {
	r.encoder = encode
	return r
}

// SetResponseHandler sets the handler of the response, which hides
// all the XXX handlers if set, but you can set it to nil to cancel it.
//
// The default response handler is derived from the client.
func (r *Request) SetResponseHandler(handler Handler) *Request {
	r.handler.All = handler
	return r
}

// SetResponseHandler1xx sets the handler of the response status code 1xx.
//
// The default response handler is derived from the client.
func (r *Request) SetResponseHandler1xx(handler Handler) *Request {
	r.handler.H1xx = handler
	return r
}

// SetResponseHandler2xx sets the handler of the response status code 2xx.
//
// The default response handler is derived from the client.
func (r *Request) SetResponseHandler2xx(handler Handler) *Request {
	r.handler.H2xx = handler
	return r
}

// SetResponseHandler3xx sets the handler of the response status code 3xx.
//
// The default response handler is derived from the client.
func (r *Request) SetResponseHandler3xx(handler Handler) *Request {
	r.handler.H3xx = handler
	return r
}

// SetResponseHandler4xx sets the handler of the response status code 4xx.
//
// The default response handler is derived from the client.
func (r *Request) SetResponseHandler4xx(handler Handler) *Request {
	r.handler.H4xx = handler
	return r
}

// SetResponseHandler5xx sets the handler of the response status code 5xx.
//
// The default response handler is derived from the client.
func (r *Request) SetResponseHandler5xx(handler Handler) *Request {
	r.handler.H5xx = handler
	return r
}

// Do sends the http request, decodes the body into result,
// and returns the response.
func (r *Request) Do(c context.Context, result interface{}) *Response {
	if r.err != nil {
		return &Response{url: r.url, mhd: r.method, err: r.err}
	}

	req, err := NewRequestWithContext(c, r.method, r.url, r.reqbody)
	if err != nil {
		return &Response{url: r.url, mhd: r.method, err: err}
	}

	if len(req.Header) == 0 {
		req.Header = r.header
	} else if len(r.header) > 0 {
		for k, vs := range r.header {
			req.Header[k] = vs
		}
	}

	if len(r.query) > 0 {
		if query := req.URL.Query(); len(query) == 0 {
			req.URL.RawQuery = r.query.Encode()
		} else {
			for k, vs := range r.query {
				query[k] = vs
			}
			req.URL.RawQuery = query.Encode()
		}
	}

	if r.hook != nil {
		req = r.hook.Request(req)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return &Response{url: r.url, mhd: r.method, err: err, req: req, resp: resp}
	} else if r.handler.All != nil {
		err = r.handler.All(result, resp)
	} else if resp.StatusCode < 200 { // 1xx
		if r.handler.H1xx != nil {
			err = r.handler.H1xx(result, resp)
		}
	} else if resp.StatusCode < 300 { // 2xx
		if r.handler.H2xx != nil {
			err = r.handler.H2xx(result, resp)
		}
	} else if resp.StatusCode < 400 { // 3xx
		if r.handler.H3xx != nil {
			err = r.handler.H3xx(result, resp)
		}
	} else if resp.StatusCode < 500 { // 4xx
		if (!r.ignore404 || resp.StatusCode != 404) && r.handler.H4xx != nil {
			err = r.handler.H4xx(result, resp)
		}
	} else { // 5xx
		if r.handler.H5xx != nil {
			err = r.handler.H5xx(result, resp)
		}
	}

	return &Response{url: r.url, mhd: r.method, err: err, req: req, resp: resp}
}

// Response is a http response.
type Response struct {
	err    error
	url    string
	mhd    string
	req    *http.Request
	resp   *http.Response
	closed bool
}

func (r *Response) close() *Response {
	if !r.closed && r.resp != nil {
		CloseBody(r.resp.Body)
		r.closed = true
	}
	return r
}

func (r *Response) getError() (err error) {
	switch r.err.(type) {
	case nil:
	case Error:
		err = r.err

	default:
		if r.resp == nil {
			err = NewError(r.mhd, r.url, r.err)
		} else {
			err = NewError(r.mhd, r.url, r.err).WithCode(r.resp.StatusCode)
		}
	}

	return
}

// Close closes the body of the response if it exists.
func (r *Response) Close() *Response { return r.close() }

// Unwrap is the same as Result, but also closes the response body.
func (r *Response) Unwrap() error { return r.close().getError() }

// UnwrapWithStatusCode is the same as Unwrap, but also returns the status code.
func (r *Response) UnwrapWithStatusCode() (int, error) {
	return r.StatusCode(), r.Unwrap()
}

// Error implements the interface error.
//
// Please use the method Unwrap instead if expecting the error type.
func (r *Response) Error() string {
	if r.err == nil {
		return ""
	}
	return r.err.Error()
}

// Result returns the result error, which is an Error if not nil.
func (r *Response) Result() error { return r.getError() }

// Request returns http.Request.
func (r *Response) Request() *http.Request { return r.req }

// Response returns http.Response.
func (r *Response) Response() *http.Response { return r.resp }

// StatusCode returns the status code.
//
// Return 0 if there is an error when sending the request.
func (r *Response) StatusCode() int {
	if r.resp == nil {
		return 0
	}
	return r.resp.StatusCode
}

// ContentLength returns the length of the response body,
// which is the value of the header "Content-Length".
//
// Return 0 if there is an error when sending the request.
func (r *Response) ContentLength() int64 {
	if r.resp == nil {
		return 0
	}
	return r.resp.ContentLength
}

// ContentType returns the type of the response body,
// which is the value of the header "Content-Type".
//
// Return "" if there is an error when sending the request.
func (r *Response) ContentType() string {
	if r.resp == nil {
		return ""
	}
	return GetContentType(r.resp.Header)
}

// Body returns the response body,
//
// Return nil if there is an error when sending the request.
func (r *Response) Body() io.ReadCloser {
	if r.resp == nil {
		return nil
	}
	return r.resp.Body
}

// ReadBody reads all the body data of the response as string.
//
// Notice: it will close the response body no matter whether it is successful.
func (r *Response) ReadBody() (body string, err error) {
	if r.resp != nil {
		buf := getBuffer()
		_, err = r.WriteTo(buf)
		r.resp.Body.Close()
		body = buf.String()
		putBuffer(buf)
	} else {
		err = r.err
	}
	return
}

// WriteTo implements the interface io.WriterTo.
//
// Notice: it will close the response body no matter whether it is successful.
func (r *Response) WriteTo(w io.Writer) (n int64, err error) {
	if r.resp != nil {
		if g, ok := w.(interface{ Grow(n int) }); ok && r.resp.ContentLength > 0 {
			if r.resp.ContentLength < 1024 {
				g.Grow(int(r.resp.ContentLength))
			} else {
				g.Grow(1024)
			}
		}

		n, err = io.CopyBuffer(w, r.resp.Body, make([]byte, 1024))
		r.resp.Body.Close()
	} else {
		err = r.err
	}
	return
}
