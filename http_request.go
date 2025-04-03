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
	"bytes"
	"context"
	"io"
	"maps"
	"net/http"
	"time"

	"github.com/xgfone/go-toolkit/httpx"
)

// Request is a http request.
type Request struct {
	common[Request]

	reqbody io.Reader
	bodybuf *bytes.Buffer
	oldbody any

	method string
	url    string
	err    error
}

// SetBody sets the body of the request.
func (r *Request) SetBody(body any) *Request {
	if r.err != nil {
		return r
	}

	r.oldbody = body
	switch body := body.(type) {
	case nil:
		r.resetBody(nil)

	case io.Reader:
		r.resetBody(body)

	default:
		if r.bodybuf == nil {
			r.bodybuf = getBuffer()
		} else {
			r.bodybuf.Reset()
		}
		r.err = r.encoder(r.bodybuf, httpx.ContentType(r.header), body)
		r.reqbody = r.bodybuf
	}

	return r
}

func (r *Request) resetBody(body io.Reader) {
	if r.bodybuf != nil {
		putBuffer(r.bodybuf)
		r.bodybuf = nil
	}
	r.reqbody = body
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
func (r *Request) Do(c context.Context, result any) (resp *Response) {
	resp = &Response{url: r.url, mhd: r.method, err: r.err, rbody: r.oldbody}
	defer r.resetBody(nil)
	defer onresp(r, resp)

	if resp.err != nil {
		return
	}

	resp.req, resp.err = http.NewRequestWithContext(c, r.method, r.url, r.reqbody)
	if resp.err != nil {
		return
	}

	maps.Copy(resp.req.Header, r.header)

	if len(r.query) > 0 {
		if query := resp.req.URL.Query(); len(query) == 0 {
			resp.req.URL.RawQuery = r.query.Encode()
		} else {
			maps.Copy(query, r.query)
			resp.req.URL.RawQuery = query.Encode()
		}
	}

	if r.hook != nil {
		resp.req, resp.err = r.hook.Request(resp.req)
		if resp.err != nil {
			return
		}
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
