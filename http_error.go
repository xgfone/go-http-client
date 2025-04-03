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

import "fmt"

// Error represents an response error from the http client.
type Error struct {
	Method string `json:"method" xml:"method"`
	URL    string `json:"url" xml:"url"`

	Code int    `json:"code" xml:"code"`
	Data string `json:"data" xml:"data"`
	Err  error  `json:"err" xml:"err"`
}

// NewError returns a new Error.
func NewError(method, url string, err error) Error {
	return Error{Method: method, URL: url, Err: err}
}

func (e Error) Unwrap() error { return e.Err }
func (e Error) Error() string { return e.String() }
func (e Error) String() string {
	buf := getBuffer()
	defer putBuffer(buf)

	_, _ = fmt.Fprintf(buf, "method=%s, url=%s", e.Method, e.URL)

	if e.Code > 0 {
		_, _ = fmt.Fprintf(buf, ", statuscode=%d", e.Code)
	}

	if e.Data != "" {
		buf.WriteString(", data=")
		buf.WriteString(e.Data)
	}

	if e.Err != nil {
		buf.WriteString(", err=")
		buf.WriteString(e.Err.Error())
	}

	return buf.String()
}

// StatusCode returns the status code.
func (e Error) StatusCode() int { return e.Code }

// WithCode returns the new Error with the given code.
func (e Error) WithCode(code int) Error { e.Code = code; return e }

// WithData returns the new Error with the given response data.
func (e Error) WithData(data string) Error { e.Data = data; return e }

// WithErr returns the new Error with the given error.
func (e Error) WithErr(err error) Error { e.Err = err; return e }
