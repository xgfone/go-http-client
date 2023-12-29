// Copyright 2023 xgfone
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

//go:build !go1.21
// +build !go1.21

package httpclient

import "log"

func logOnResponse(r *Response) {
	log.Printf("log the http request: method=%s, url=%s, statuscode=%d, cost=%s, err=%s",
		r.Method(), r.Url(), r.StatusCode(), r.Cost().String(), r.Error())
}
