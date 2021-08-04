// Copyright 2021 xgfone
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
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}

		if v := r.Header.Get("Key"); v != "value" {
			rw.WriteHeader(400)
			fmt.Fprintf(rw, "unknown header Key: %s", v)
		} else if v := r.Header.Get("Accept"); v != "application/json" {
			rw.WriteHeader(400)
			fmt.Fprintf(rw, "unknown header Accept: %s", v)
		} else if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			rw.WriteHeader(400)
			fmt.Fprintf(rw, err.Error())
		} else if req.Name != "xgfone" {
			rw.WriteHeader(400)
			fmt.Fprintf(rw, "unknown request name '%s'", req.Name)
		} else {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(200)
			json.NewEncoder(rw).Encode(map[string]string{
				"username": "xgfone",
				"password": "123456",
			})
		}
	})
	server := &http.Server{Addr: "localhost:12345", Handler: handler}
	go server.ListenAndServe()
	defer server.Shutdown(context.TODO())
	time.Sleep(time.Second)

	var result struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	_, err := Get("http://localhost:12345").
		AddHeader("Key", "value").
		AddAccept("application/json").
		SetBody(map[string]string{"name": "xgfone"}).
		Do(context.Background(), &result).
		Result()

	if err != nil {
		t.Error(err)
	} else if result.Username != "xgfone" || result.Password != "123456" {
		t.Error(result)
	}
}
