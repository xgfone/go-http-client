// Copyright 2025 xgfone
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
	"io"
	"net/http"
	"net/url"

	"github.com/xgfone/go-toolkit/httpx"
)

type (
	// Encoder is used to encode the data by the content type to dst.
	Encoder func(dst io.Writer, contentType string, data any) error

	// Handler is used to handle the response.
	Handler func(dst any, resp *http.Response) error
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

type common[T any] struct {
	target *T

	hook    Hook
	query   url.Values
	header  http.Header
	onresp  func(*Response)
	handler respHandler
	encoder Encoder
	client  Doer

	ignore404   bool
	clonehook   bool
	clonequery  bool
	cloneheader bool
}

func newCommon[T any](target *T) common[T] {
	c := common[T]{
		target: target,
		header: make(http.Header, 4),
		query:  make(url.Values, 4),
	}

	c.SetHTTPClient(http.DefaultClient)
	c.SetBodyEncoder(EncodeData)
	c.OnResponse(logOnResponse)
	return c
}

func copyCommon(dst *common[Request], src *common[Client]) {
	dst.hook = src.hook
	dst.query = src.query
	dst.header = src.header
	dst.client = src.client
	dst.onresp = src.onresp
	dst.handler = src.handler
	dst.encoder = src.encoder
	dst.ignore404 = src.ignore404
	dst.clonehook = true
	dst.clonequery = true
	dst.cloneheader = true
}

func (c common[T]) clone(target *T) common[T] {
	_c := c
	_c.hook = cloneHook(c.hook)
	_c.query = cloneQuery(c.query)
	_c.header = c.header.Clone()
	_c.target = target
	return _c
}

func (c *common[T]) cloneQuery() {
	if c.clonequery {
		c.query = cloneQuery(c.query)
		c.clonequery = false
	}
}

func (c *common[T]) cloneHeader() {
	if c.cloneheader {
		c.header = c.header.Clone()
		c.cloneheader = false
	}
}

// Ignore404 sets whether to ignore the status code 404.
// If true, the 4xx response handler won't be called
// when the http server returns 404.
//
// Default: false
func (c *common[T]) Ignore404(ignore bool) *T {
	c.ignore404 = ignore
	return c.target
}

// OnResponse sets a callback function to wrap the response,
// which can be used to log the request and response result.
//
// For the default, it will log the method, url, status code and cost duration.
func (c *common[T]) OnResponse(f func(*Response)) *T {
	c.onresp = f
	return c.target
}

// GetHTTPClient returns the inner http client.
func (c *common[T]) GetHTTPClient() Doer {
	return c.client
}

// SetHTTPClient resets the http client.
func (c *common[T]) SetHTTPClient(client Doer) *T {
	c.client = client
	return c.target
}

// SetHook resets the request hook.
func (c *common[T]) SetHook(hook Hook) *T {
	c.clonehook = false
	c.hook = hook
	return c.target
}

// AddHook appends the request hook.
func (c *common[T]) AddHook(hook Hook) *T {
	if hook == nil {
		panic("AddHook: the hook must not be nil")
	}

	switch hooks := c.hook.(type) {
	case nil:
		c.hook = hook

	case Hooks:
		if c.clonehook {
			_hooks := make(Hooks, 0, len(hooks)+1)
			_hooks = append(_hooks, hooks...)
			_hooks = append(_hooks, hook)
			c.SetHook(_hooks)
		} else {
			c.hook = append(hooks, hook)
		}

	default:
		c.SetHook(Hooks{c.hook, hook})
	}

	return c.target
}

// AddQueries adds the request queries.
func (c *common[T]) AddQueries(queries url.Values) *T {
	if len(queries) == 0 {
		return c.target
	}

	c.cloneQuery()
	for key, values := range queries {
		c.query[key] = values
	}
	return c.target
}

// AddQueryMap adds the request queries as a map type.
func (c *common[T]) AddQueryMap(queries map[string]string) *T {
	if len(queries) == 0 {
		return c.target
	}

	c.cloneQuery()
	for key, value := range queries {
		c.query.Add(key, value)
	}
	return c.target
}

// AddQuery appends the value for the query key.
func (c *common[T]) AddQuery(key, value string) *T {
	c.cloneQuery()
	c.query.Add(key, value)
	return c.target
}

// SetQuery sets the query key to the value.
func (c *common[T]) SetQuery(key, value string) *T {
	c.cloneQuery()
	c.query.Set(key, value)
	return c.target
}

// AddHeaders adds the request headers.
func (c *common[T]) AddHeaders(headers http.Header) *T {
	if len(headers) == 0 {
		return c.target
	}

	c.cloneQuery()
	for key, values := range headers {
		c.header[key] = values
	}
	return c.target
}

// AddHeaderMap adds the request headers as a map type.
func (c *common[T]) AddHeaderMap(headers map[string]string) *T {
	if len(headers) == 0 {
		return c.target
	}

	c.cloneQuery()
	for key, value := range headers {
		c.header.Add(key, value)
	}
	return c.target
}

// AddHeader adds the default request header as "key: value".
func (c *common[T]) AddHeader(key, value string) *T {
	c.cloneQuery()
	c.header.Add(key, value)
	return c.target
}

// SetHeader sets the default request header as "key: value".
func (c *common[T]) SetHeader(key, value string) *T {
	c.cloneQuery()
	c.header.Set(key, value)
	return c.target
}

// SetContentType sets the default Content-Type, which is equal to
// SetHeader("Content-Type", ct).
//
// Default: "application/json; charset=UTF-8"
func (c *common[T]) SetContentType(ct string) *T {
	return c.SetHeader(httpx.HeaderContentType, ct)
}

// SetAccepts resets the accepted types of the response body to accepts.
func (c *common[T]) SetAccepts(accepts ...string) *T {
	if len(accepts) == 0 {
		return c.target
	}

	c.cloneHeader()
	c.header[httpx.HeaderAccept] = accepts
	return c.target
}

// AddAccept adds the accepted types of the response body, which is equal to
// AddHeader("Accept", contentType).
func (c *common[T]) AddAccept(contentType string) *T {
	return c.AddHeader(httpx.HeaderAccept, contentType)
}

// SetBodyEncoder sets the encoder to encode the request body.
//
// Default: EncodeData.
func (c *common[T]) SetBodyEncoder(encoder Encoder) *T {
	c.encoder = encoder
	return c.target
}

// ClearAllResponseHandlers clears all the set response handlers.
func (c *common[T]) ClearAllResponseHandlers() *T {
	c.handler = respHandler{}
	return c.target
}

// SetResponseHandler sets the handler of the response, which hides
// all the XXX handlers if set, but you can set it to nil to cancel it.
//
// Default: nil
func (c *common[T]) SetResponseHandler(handler Handler) *T {
	c.handler.All = handler
	return c.target
}

// SetResponseHandler1xx sets the handler of the response status code 1xx.
//
// Default: nil
func (c *common[T]) SetResponseHandler1xx(handler Handler) *T {
	c.handler.H1xx = handler
	return c.target
}

// SetResponseHandler2xx sets the handler of the response status code 2xx.
//
// Default: DecodeResponseBody
func (c *common[T]) SetResponseHandler2xx(handler Handler) *T {
	c.handler.H2xx = handler
	return c.target
}

// SetResponseHandler3xx sets the handler of the response status code 3xx.
//
// Default: nil
func (c *common[T]) SetResponseHandler3xx(handler Handler) *T {
	c.handler.H3xx = handler
	return c.target
}

// SetResponseHandler4xx sets the handler of the response status code 4xx.
//
// Default: nil
func (c *common[T]) SetResponseHandler4xx(handler Handler) *T {
	c.handler.H4xx = handler
	return c.target
}

// SetResponseHandler5xx sets the handler of the response status code 5xx.
//
// Default: nil
func (c *common[T]) SetResponseHandler5xx(handler Handler) *T {
	c.handler.H5xx = handler
	return c.target
}

// SetResponseHandlerDefault sets the default handler of the response.
//
// Default: ReadResponseBodyAsError
func (c *common[T]) SetResponseHandlerDefault(handler Handler) *T {
	c.handler.Default = handler
	return c.target
}
