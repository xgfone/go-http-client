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
	"testing"
)

type testHook struct{ Name string }

func newTestHook(name string) Hook                                { return testHook{name} }
func (h testHook) Request(r *http.Request) (*http.Request, error) { return r, nil }

func hookAddQuery(key, value string) HookFunc {
	return func(r *http.Request) (*http.Request, error) {
		query := r.URL.Query()
		query.Add(key, value)
		r.URL.RawQuery = query.Encode()
		return r, nil
	}
}

func TestAddHook(t *testing.T) {
	client := NewClient(nil)
	client.AddHook(newTestHook("hook1"))
	client.AddHook(newTestHook("hook2"))

	req := client.Get("http://127.0.0.1")
	req.AddHook(newTestHook("hook3"))

	if hooks := client.hook.(Hooks); len(hooks) != 2 {
		t.Errorf("expect 2 hooks, but got %d", len(hooks))
	} else if cap(hooks) != 2 {
		t.Errorf("expect hooks cap 2, but got %d", cap(hooks))
	}

	if hooks, ok := req.hook.(Hooks); !ok {
		t.Errorf("expect type Hooks, but got '%T'", req.hook)
	} else if len(hooks) != 3 {
		t.Errorf("expect 3 hooks, but got %d", len(hooks))
	} else {
		for i, hook := range hooks {
			switch name := hook.(testHook).Name; name {
			case "hook1", "hook2", "hook3":
			default:
				t.Errorf("%d: unexpected hook named '%s'", i, name)
			}
		}

		hooks[0] = newTestHook("hook4")
		if name := client.hook.(Hooks)[0].(testHook).Name; name != "hook1" {
			t.Errorf("expect hook named '%s', but got '%s'", "hook1", name)
		}
	}

	req.AddHook(hookAddQuery("q1", "v1"))
	req.SetHTTPClient(DoFunc(func(r *http.Request) (*http.Response, error) {
		v := r.URL.Query().Get("q1")
		if v != "v1" {
			return nil, fmt.Errorf("expect query value 'v1', but got '%s'", v)
		}

		return &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.0",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(nil),
		}, nil
	}))

	err := req.Do(context.Background(), nil).Unwrap()
	if err != nil {
		t.Error(err)
	}
}
