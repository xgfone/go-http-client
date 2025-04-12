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
	"encoding/json"
	"net/http"
	"testing"
)

type _NoJsonError struct{ Data string }

func (e _NoJsonError) Error() string {
	return e.Data
}

type _JsonError struct{ Data string }

func (e _JsonError) Error() string {
	return e.Data
}

func (e _JsonError) MarshalJSON() ([]byte, error) {
	type E _JsonError
	return json.Marshal(E(e))
}

func TestErrorJSON(t *testing.T) {
	s1 := `{"method":"GET","url":"http://localhost?k=v","err":"not found"}`
	e1 := NewError(http.MethodGet, "http://localhost?k=v", _NoJsonError{Data: "not found"})
	data1, err := json.Marshal(e1)
	if err != nil {
		t.Error(err)
	} else if s := string(data1); s != s1 {
		t.Errorf("expect json data '%s', but got '%s'", s1, s)
	}

	s2 := `{"method":"GET","url":"http://localhost?k=v","err":{"Data":"not found"}}`
	e2 := NewError(http.MethodGet, "http://localhost?k=v", _JsonError{Data: "not found"})
	data2, err := json.Marshal(e2)
	if err != nil {
		t.Error(err)
	} else if s := string(data2); s != s2 {
		t.Errorf("expect json data '%s', but got '%s'", s2, s)
	}
}
