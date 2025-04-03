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
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/xgfone/go-toolkit/httpx"
)

func TestUrlMerge(t *testing.T) {
	var baseurl = strings.TrimRight("http://127.0.0.1///", "/")
	var expect string

	expect = "http://127.0.0.1"
	if url := mergeurl(baseurl, ""); url != expect {
		t.Errorf("expect url '%s', but got '%s'", expect, url)
	}

	expect = "http://127.0.0.1/"
	if url := mergeurl(baseurl, "/"); url != expect {
		t.Errorf("expect url '%s', but got '%s'", expect, url)
	}

	expect = "http://127.0.0.1/path"
	if url := mergeurl(baseurl, "/path"); url != expect {
		t.Errorf("expect url '%s', but got '%s'", expect, url)
	}

	expect = "http://127.0.0.1/path"
	if url := mergeurl(baseurl, "path"); url != expect {
		t.Errorf("expect url '%s', but got '%s'", expect, url)
	}
}

func TestClient(t *testing.T) {
	doer := DoFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Scheme != "https" {
			return nil, fmt.Errorf("expect scheme 'https', but got '%s'", req.URL.Scheme)
		} else if req.Host != "example.com" {
			return nil, fmt.Errorf("expect host 'example.com', but got '%s'", req.Host)
		} else if req.URL.Path != "/v1/path/to" {
			return nil, fmt.Errorf("expect path '/v1/path/to', but got '%s'", req.URL.Path)
		}

		buf := strings.NewReader(`{"name":"xgfone","age":18}`)
		return &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.0",
			ProtoMajor: 1,
			ProtoMinor: 1,

			Header:        http.Header{httpx.HeaderContentType: []string{httpx.MIMEApplicationJSON}},
			ContentLength: int64(buf.Len()),

			Close: true,
			Body:  io.NopCloser(buf),
		}, nil
	})

	var resp struct {
		Name string
		Age  int
	}

	DefaultClient.SetBaseURL("https://example.com/v1")
	err := Get("/path/to").
		SetHTTPClient(doer).
		AddAccept(httpx.MIMEApplicationJSON).
		AddHeader("k1", "v1").
		AddQuery("k2", "v2").
		Do(context.Background(), &resp).
		Unwrap()
	if err != nil {
		t.Fatalf("got an error: %+v", err)
	}

	result := struct {
		Name string
		Age  int
	}{
		Name: "xgfone",
		Age:  18,
	}

	if result != resp {
		t.Errorf("expect result '%+v', but got '%+v'", result, resp)
	}
}
