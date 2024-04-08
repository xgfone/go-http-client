// Copyright 2021~2024 xgfone
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
	"time"
)

// Pre-define some constants.
const (
	HeaderAccept           = "Accept"
	HeaderAcceptedLanguage = "Accept-Language"
	HeaderAcceptEncoding   = "Accept-Encoding"
	HeaderAuthorization    = "Authorization"
	HeaderContentType      = "Content-Type"
	HeaderUserAgent        = "User-Agent"

	MIMEMultipartForm              = "multipart/form-data"
	MIMEApplicationForm            = "application/x-www-form-urlencoded"
	MIMEApplicationXML             = "application/xml"
	MIMEApplicationJSON            = "application/json"
	MIMEApplicationXMLCharsetUTF8  = "application/xml; charset=UTF-8"
	MIMEApplicationJSONCharsetUTF8 = "application/json; charset=UTF-8"
)

var bufpool = sync.Pool{New: func() interface{} {
	return bytes.NewBuffer(make([]byte, 0, 512))
}}

func getBuffer() *bytes.Buffer    { return bufpool.Get().(*bytes.Buffer) }
func putBuffer(buf *bytes.Buffer) { buf.Reset(); bufpool.Put(buf) }

type bytesT struct{ Data []byte }

var bytespool = sync.Pool{New: func() interface{} {
	return &bytesT{Data: make([]byte, 256)}
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
		_, _ = io.CopyBuffer(discarder{}, r, buf.Data[:])
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
	ct := header.Get(HeaderContentType)
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
		case MIMEApplicationXML:
			err = xml.NewEncoder(w).Encode(data)
		case MIMEApplicationJSON:
			enc := json.NewEncoder(w)
			enc.SetEscapeHTML(false)
			err = enc.Encode(data)
		case MIMEApplicationForm:
			switch v := data.(type) {
			case url.Values:
				_, err = io.WriteString(w, v.Encode())

			case map[string]string:
				vs := make(url.Values, len(v))
				for key, value := range v {
					vs[key] = []string{value}
				}
				_, err = io.WriteString(w, vs.Encode())

			case map[string]interface{}:
				vs := make(url.Values, len(v))
				for key, value := range v {
					vs[key] = []string{fmt.Sprint(value)}
				}
				_, err = io.WriteString(w, vs.Encode())

			case interface{ MarshalForm() ([]byte, error) }:
				var _data []byte
				if _data, err = v.MarshalForm(); err == nil {
					_, err = w.Write(_data)
				}

			default:
				err = fmt.Errorf("not support to encode %T to %s", data, MIMEApplicationForm)
			}
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
	switch ct {
	case "":
		err = errors.New("no response header Content-Type")
	case MIMEApplicationXML:
		err = xml.NewDecoder(r).Decode(dst)
	case MIMEApplicationJSON:
		err = json.NewDecoder(r).Decode(dst)
	default:
		err = fmt.Errorf("unsupported response Content-Type '%s'", ct)
	}
	return
}

// DecodeResponseBody is a response handler to decode the response body
// into dst.
func DecodeResponseBody(dst interface{}, resp *http.Response) (err error) {
	if dst == nil || resp.StatusCode == 204 {
		return
	}
	return DecodeFromReader(dst, GetContentType(resp.Header), resp.Body)
}

// ReadResponseBodyAsError is a response handler to read the response body
// as the error to be returned.
func ReadResponseBodyAsError(dst interface{}, resp *http.Response) error {
	if resp.StatusCode >= 300 && resp.StatusCode < 400 { // For 3xx
		return nil
	}

	err := Error{Code: resp.StatusCode}
	err.Err = fmt.Errorf("got status code %d", resp.StatusCode)

	if req := resp.Request; req != nil {
		err.Method = req.Method
		err.URL = req.URL.String()
	}

	buf := getBuffer()
	bytebuf := getBytes()
	_, _ = io.CopyBuffer(buf, resp.Body, bytebuf.Data)
	err.Data = buf.String()
	putBytes(bytebuf)
	putBuffer(buf)

	return err
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

func cloneHook(hook Hook) Hook {
	if hooks, ok := hook.(Hooks); ok && len(hooks) > 0 {
		hook = append(Hooks{}, hooks...)
	}
	return hook
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

	Default Handler
}

// Client is a http client to build a request and parse the response.
type Client struct {
	hook    Hook
	query   url.Values
	header  http.Header
	client  *http.Client
	baseurl string
	encoder Encoder
	handler respHandler
	onresp  func(*Response)

	ignore404 bool
}

// NewClient returns a new Client with the http client.
func NewClient(client *http.Client) *Client {
	c := &Client{
		client:  client,
		query:   make(url.Values, 4),
		header:  make(http.Header, 4),
		onresp:  logOnResponse,
		encoder: EncodeData,
	}
	c.SetContentType(MIMEApplicationJSONCharsetUTF8)
	c.SetResponseHandler2xx(DecodeResponseBody)
	c.SetResponseHandlerDefault(ReadResponseBodyAsError)
	return c
}

// Clone clones itself to a new one.
func (c *Client) Clone() *Client {
	return &Client{
		hook:    cloneHook(c.hook),
		client:  c.client,
		query:   cloneQuery(c.query),
		header:  cloneHeader(c.header),
		onresp:  c.onresp,
		baseurl: c.baseurl,
		encoder: c.encoder,
		handler: c.handler,

		ignore404: c.ignore404,
	}
}

// OnResponse sets a callback function to wrap the response,
// which can be used to log the request and response result.
//
// For the default, it will log the method, url, status code and cost duration.
func (c *Client) OnResponse(f func(*Response)) *Client {
	c.onresp = f
	return c
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
	c.baseurl = strings.TrimRight(baseurl, "/")
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
	return c.SetHeader(HeaderContentType, ct)
}

// SetAccepts resets the accepted types of the response body to accepts.
func (c *Client) SetAccepts(accepts ...string) *Client {
	c.header[HeaderAccept] = accepts
	return c
}

// AddAccept adds the accepted types of the response body, which is equal to
// AddHeader("Accept", contentType).
func (c *Client) AddAccept(contentType string) *Client {
	return c.AddHeader(HeaderAccept, contentType)
}

// SetBodyEncoder sets the encoder to encode the request body.
//
// The default encoder is EncodeData.
func (c *Client) SetBodyEncoder(encoder Encoder) *Client {
	c.encoder = encoder
	return c
}

// SetReqBodyEncoder is the alias of SetBodyEncoder.
//
// DEPRECATED! Please use SetBodyEncoder instead.
func (c *Client) SetReqBodyEncoder(encoder Encoder) *Client {
	c.encoder = encoder
	return c
}

// ClearAllResponseHandlers clears all the set response handlers.
func (c *Client) ClearAllResponseHandlers() *Client {
	c.handler = respHandler{}
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

// SetResponseHandlerDefault sets the default handler of the response.
//
// Default: nil
func (c *Client) SetResponseHandlerDefault(handler Handler) *Client {
	c.handler.Default = handler
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

func mergeurl(baseurl, requrl string) string {
	switch _len := len(requrl); {
	case _len == 0:

	case requrl[0] == '/':
		baseurl += requrl

	default:
		baseurl = strings.Join([]string{baseurl, requrl}, "/")
	}

	return baseurl
}

// Request builds and returns a new request.
func (c *Client) Request(method, requrl string) *Request {
	_url := requrl

	var err error
	if !strings.HasPrefix(requrl, "http") {
		if c.baseurl == "" {
			err = fmt.Errorf("invalid request url '%s'", requrl)
		} else {
			_url = mergeurl(c.baseurl, requrl)
		}
	}

	return &Request{
		ignore404: c.ignore404,

		hclone: true,
		qclone: true,
		header: c.header,
		query:  c.query,

		hook:    c.hook,
		encoder: c.encoder,
		handler: c.handler,
		onresp:  c.onresp,
		client:  c.client,
		method:  method,
		url:     _url,
		err:     err,
	}
}

// Request is a http request.
type Request struct {
	ignore404 bool

	header http.Header
	hclone bool

	qclone bool
	query  url.Values

	reqbody io.Reader
	bodybuf *bytes.Buffer
	body    interface{}

	hook    Hook
	hookset bool
	encoder Encoder
	handler respHandler
	onresp  func(*Response)
	client  *http.Client
	method  string
	url     string
	err     error
}

func (r *Request) cloneQuery() {
	if r.qclone {
		r.query = cloneQuery(r.query)
		r.qclone = false
	}
}

func (r *Request) cloneHeader() {
	if r.hclone {
		r.header = cloneHeader(r.header)
		r.hclone = false
	}
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
	if len(queries) == 0 {
		return r
	}

	r.cloneQuery()
	for key, values := range queries {
		r.query[key] = values
	}
	return r
}

// AddQueryMap adds the request queries as a map type.
func (r *Request) AddQueryMap(queries map[string]string) *Request {
	if len(queries) == 0 {
		return r
	}

	r.cloneQuery()
	for key, value := range queries {
		r.query.Add(key, value)
	}
	return r
}

// AddQuery appends the value for the query key.
func (r *Request) AddQuery(key, value string) *Request {
	r.cloneQuery()
	r.query.Add(key, value)
	return r
}

// SetQuery sets the query key to the value.
func (r *Request) SetQuery(key, value string) *Request {
	r.cloneQuery()
	r.query.Set(key, value)
	return r
}

// AddHeader adds the request header as "key: value".
func (r *Request) AddHeader(key, value string) *Request {
	r.cloneHeader()
	r.header.Add(key, value)
	return r
}

// AddHeaders adds the request headers.
func (r *Request) AddHeaders(headers http.Header) *Request {
	if len(headers) == 0 {
		return r
	}

	r.cloneHeader()
	for key, values := range headers {
		r.header[key] = values
	}
	return r
}

// AddHeaderMap adds the request headers as a map type.
func (r *Request) AddHeaderMap(headers map[string]string) *Request {
	if len(headers) == 0 {
		return r
	}

	r.cloneHeader()
	for key, value := range headers {
		r.header.Add(key, value)
	}
	return r
}

// SetHeader adds the request header as "key: value".
func (r *Request) SetHeader(key, value string) *Request {
	r.cloneHeader()
	r.header.Set(key, value)
	return r
}

// SetContentType sets the default Content-Type, which is equal to
// SetHeader("Content-Type", ct).
func (r *Request) SetContentType(ct string) *Request {
	return r.SetHeader(HeaderContentType, ct)
}

// SetAccepts resets the accepted types of the response body to accepts.
func (r *Request) SetAccepts(accepts ...string) *Request {
	if len(accepts) == 0 {
		return r
	}

	r.cloneHeader()
	r.header[HeaderAccept] = accepts
	return r
}

// AddAccept adds the accepted types of the response body, which is equal to
// AddHeader("Accept", contentType).
func (r *Request) AddAccept(contentType string) *Request {
	return r.AddHeader(HeaderAccept, contentType)
}

// SetBody sets the body of the request.
func (r *Request) SetBody(body interface{}) *Request {
	if r.err != nil {
		return r
	}

	r.body = body
	switch body := body.(type) {
	case nil:
		r.cleanBody(nil)

	case io.Reader:
		r.cleanBody(body)

	default:
		if r.bodybuf == nil {
			r.bodybuf = getBuffer()
		} else {
			r.bodybuf.Reset()
		}
		r.err = r.encoder(r.bodybuf, GetContentType(r.header), body)
		r.reqbody = r.bodybuf
	}

	return r
}

func (r *Request) cleanBody(body io.Reader) {
	if r.bodybuf != nil {
		putBuffer(r.bodybuf)
		r.bodybuf = nil
	}
	r.reqbody = body
}

// SetBodyEncoder sets the encoder to encode the request body.
//
// The default encoder is derived from the client.
func (r *Request) SetBodyEncoder(encoder Encoder) *Request {
	r.encoder = encoder
	return r
}

// SetReqBodyEncoder is alias of SetBodyEncoder.
//
// DEPRECATED! Please use SetBodyEncoder instead.
func (r *Request) SetReqBodyEncoder(encoder Encoder) *Request {
	r.encoder = encoder
	return r
}

// ClearAllResponseHandlers clears all the set response handlers.
func (r *Request) ClearAllResponseHandlers() *Request {
	r.handler = respHandler{}
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

// SetResponseHandlerDefault sets the default handler of the response.
//
// Default: nil
func (r *Request) SetResponseHandlerDefault(handler Handler) *Request {
	r.handler.Default = handler
	return r
}

// OnResponse sets a callback function to wrap the response,
// which can be used to log the request and response result.
func (r *Request) OnResponse(f func(*Response)) *Request {
	r.onresp = f
	return r
}

func onresp(req *Request, resp *Response) {
	if req.onresp != nil {
		req.onresp(resp)
	}
}

// Do sends the http request, decodes the body into result,
// and returns the response.
//
// If result is a function, func(*http.Response) error, call it instead
// of calling the response handler.
func (r *Request) Do(c context.Context, result interface{}) (resp *Response) {
	resp = &Response{url: r.url, mhd: r.method, err: r.err, rbody: r.body}
	defer r.cleanBody(nil)
	defer onresp(r, resp)

	if resp.err != nil {
		return
	}

	resp.req, resp.err = NewRequestWithContext(c, r.method, r.url, r.reqbody)
	if resp.err != nil {
		return
	}

	if len(resp.req.Header) == 0 {
		resp.req.Header = r.header
	} else if len(r.header) > 0 {
		for k, vs := range r.header {
			resp.req.Header[k] = vs
		}
	}

	if len(r.query) > 0 {
		if query := resp.req.URL.Query(); len(query) == 0 {
			resp.req.URL.RawQuery = r.query.Encode()
		} else {
			for k, vs := range r.query {
				query[k] = vs
			}
			resp.req.URL.RawQuery = query.Encode()
		}
	}

	if r.hook != nil {
		resp.req = r.hook.Request(resp.req)
	}

	start := time.Now()
	resp.resp, resp.err = r.client.Do(resp.req)
	resp.cost = time.Since(start)
	if resp.err != nil {
		return
	}

	if f, ok := result.(func(*http.Response) error); ok {
		resp.err = f(resp.resp)
		return
	}

	status := resp.resp.StatusCode
	switch {
	case r.handler.All != nil:
		resp.err = r.handler.All(result, resp.resp)

	case r.handler.H1xx != nil && status < 200:
		resp.err = r.handler.H1xx(result, resp.resp)

	case r.handler.H2xx != nil && status < 300:
		resp.err = r.handler.H2xx(result, resp.resp)

	case r.handler.H3xx != nil && status < 400:
		resp.err = r.handler.H3xx(result, resp.resp)

	case r.handler.H4xx != nil && status < 500 &&
		(!r.ignore404 || resp.resp.StatusCode != 404):
		resp.err = r.handler.H4xx(result, resp.resp)

	case r.handler.H5xx != nil:
		resp.err = r.handler.H5xx(result, resp.resp)

	case r.handler.Default != nil:
		resp.err = r.handler.Default(result, resp.resp)
	}

	return
}

// Response is a http response.
type Response struct {
	err    error
	url    string
	mhd    string
	req    *http.Request
	resp   *http.Response
	cost   time.Duration
	rbody  interface{}
	closed bool
}

func (r *Response) close() *Response {
	if !r.closed && r.resp != nil {
		_ = CloseBody(r.resp.Body)
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
		err = r.ToError(r.err)
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

// ToError returns an Error with the given error.
func (r *Response) ToError(err error) Error {
	if r.resp == nil {
		return NewError(0, r.mhd, r.url, err)
	}
	return NewError(r.resp.StatusCode, r.mhd, r.url, err)
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

// Cost returns the cost duration to call the request.
func (r *Response) Cost() time.Duration { return r.cost }

// Url returns the original request url.
func (r *Response) Url() string { return r.url }

// Method returns the original request method.
func (r *Response) Method() string { return r.mhd }

// Result returns the result error, which is an Error if not nil.
func (r *Response) Result() error { return r.getError() }

// ReqBody returns the original request body.
func (r *Response) ReqBody() interface{} { return r.rbody }

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
