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
	"net/http"
	"slices"
)

// Hook is a hook to wrap and modify the http request.
type Hook interface {
	Request(*http.Request) (*http.Request, error)
}

// HookFunc is a hook function.
type HookFunc func(*http.Request) (*http.Request, error)

// Request implements the interface Hook.
func (f HookFunc) Request(r *http.Request) (*http.Request, error) { return f(r) }

// Hooks is a set of hooks.
type Hooks []Hook

// Request implements the interface Hook.
func (hs Hooks) Request(r *http.Request) (*http.Request, error) {
	for _, hook := range hs {
		var err error
		r, err = hook.Request(r)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

func cloneHook(hook Hook) Hook {
	if hooks, ok := hook.(Hooks); ok && len(hooks) > 0 {
		hook = slices.Clone(hooks)
	}
	return hook
}
