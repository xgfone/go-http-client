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
	"fmt"
	"net/http"
	"strings"
)

// Doer is an interface to send a http request and get a response.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// DoFunc is a function to send a http request and get a response.
type DoFunc func(*http.Request) (*http.Response, error)

// DoFunc implements the Doer interface.
func (f DoFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

// Client is a http client to build a request and parse the response.
type Client struct {
	baseurl string
	common[Client]
}

// NewClient returns a new Client with the http client.
func NewClient(client Doer) *Client {
	c := &Client{}
	c.common = newCommon(c)
	c.SetHTTPClient(client)
	c.SetAccepts("application/json")
	c.SetContentType("application/json; charset=UTF-8")
	c.SetResponseHandler2xx(DecodeResponseBody)
	c.SetResponseHandlerDefault(ReadResponseBodyAsError)
	return c
}

// Do implements the Doer interface.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// Clone clones itself and returns the new one.
func (c *Client) Clone() *Client {
	newclient := &Client{baseurl: c.baseurl}
	newclient.common = c.common.clone(newclient)
	return newclient
}

// BaseURL returns the inner base url.
func (c *Client) BaseURL() string {
	return c.baseurl
}

// SetBaseURL sets the default base url.
//
// If baseurl is empty, it will clear the base url.
func (c *Client) SetBaseURL(baseurl string) *Client {
	c.baseurl = strings.TrimRight(baseurl, "/")
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
	if !strings.HasPrefix(requrl, "https://") && !strings.HasPrefix(requrl, "http://") {
		if c.baseurl == "" {
			err = fmt.Errorf("invalid request url '%s'", requrl)
		} else {
			_url = mergeurl(c.baseurl, requrl)
		}
	}

	req := &Request{method: method, url: _url, err: err}
	req.common.target = req
	copyCommon(&req.common, &c.common)
	return req
}
