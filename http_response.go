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
	"time"
)

// Response is a http response.
type Response struct {
	err   error
	url   string
	mhd   string
	req   *http.Request
	resp  *http.Response
	cost  time.Duration
	rbody any
}

func (r *Response) close() *Response {
	if r.resp != nil {
		_ = r.resp.Body.Close()
	}
	return r
}

func (r *Response) error() (err error) {
	switch r.err.(type) {
	case nil:
	case Error:
		err = r.err
	default:
		err = NewError(r.mhd, r.url, r.err).WithCode(r.StatusCode())
	}
	return
}

// Close closes the body of the response if it exists.
func (r *Response) Close() *Response { return r.close() }

// Unwrap is the same as Result, but also closes the response body.
func (r *Response) Unwrap() error { return r.close().error() }

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

// Cost returns the cost duration to call the request.
func (r *Response) Cost() time.Duration { return r.cost }

// Url returns the original request url.
func (r *Response) Url() string { return r.url }

// Method returns the original request method.
func (r *Response) Method() string { return r.mhd }

// Result returns the result error, which is an Error if not nil.
func (r *Response) Result() error { return r.error() }

// ReqBody returns the original request body.
func (r *Response) ReqBody() any { return r.rbody }

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
	return getContentType(r.resp.Header)
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
	if r.err != nil {
		return "", r.err
	}

	buf := getBuffer()
	_, err = r.WriteTo(buf)
	body = buf.String()
	putBuffer(buf)
	return
}

// WriteTo implements the interface io.WriterTo.
//
// Notice: it will close the response body no matter whether it is successful.
func (r *Response) WriteTo(w io.Writer) (n int64, err error) {
	if r.err != nil {
		return 0, r.err
	}

	if g, ok := w.(interface{ Grow(n int) }); ok && r.resp.ContentLength > 0 {
		if r.resp.ContentLength < 1024 {
			g.Grow(int(r.resp.ContentLength))
		} else {
			g.Grow(1024)
		}
	}

	defer r.resp.Body.Close()

	buf := getBytes()
	n, err = io.CopyBuffer(w, r.resp.Body, buf.Data)
	putBytes(buf)

	return
}
