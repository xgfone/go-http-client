// Copyright 2022 xgfone
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
	Code   int    `json:"code" xml:"code"`
	Method string `json:"method" xml:"method"`
	URL    string `json:"url" xml:"url"`
	Data   string `json:"data" xml:"data"`
	Err    error  `json:"err" xml:"err"`
}

// NewError returns a new Error.
func NewError(code int, method, url string) Error {
	return Error{Code: code, Method: method, URL: url}
}

func (e Error) Unwrap() error { return e.Err }
func (e Error) Error() string { return e.String() }
func (e Error) String() string {
	var err string
	if e.Err != nil {
		err = fmt.Sprintf(", err=%s", e.Err.Error())
	}

	var data string
	if e.Data != "" {
		data = fmt.Sprintf(", data=%s", e.Data)
	}

	var code string
	if e.Code > 0 {
		code = fmt.Sprintf(", code=%d", e.Code)
	}

	return fmt.Sprintf("method=%s, url=%s%s%s%s", e.Method, e.URL, code, data, err)
}

// StatusCode returns the status code.
func (e Error) StatusCode() int { return e.Code }

// WithCode returns the new Error with the given code.
func (e Error) WithCode(code int) Error { e.Code = code; return e }

// WithData returns the new Error with the given response data.
func (e Error) WithData(data string) Error { e.Data = data; return e }

// WithErr returns the new Error with the given error.
func (e Error) WithErr(err error) Error { e.Err = err; return e }
