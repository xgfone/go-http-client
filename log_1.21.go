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

//go:build go1.21
// +build go1.21

package httpclient

import (
	"encoding/json"
	"log/slog"
	"unsafe"
)

// LogOnResponse is used to log the request on response.
//
// appendAttrs may be nil.
func LogOnResponse(resp *Response, level slog.Level, appendAttrs func(r *Response, attrs []slog.Attr) []slog.Attr) {
	_logOnResponse(resp, level, appendAttrs)
}

func logOnResponse(r *Response) { _logOnResponse(r, slog.LevelDebug, nil) }

func _logOnResponse(r *Response, level slog.Level,
	appendAttrs func(r *Response, kvs []slog.Attr) []slog.Attr) {
	if !slog.Default().Enabled(r.req.Context(), level) {
		return
	}

	kvs := make([]slog.Attr, 0, 10)
	kvs = append(kvs,
		slog.String("method", r.Method()),
		slog.String("url", r.Url()),
	)

	if r.req != nil {
		kvs = append(kvs, slog.Any("reqheaders", r.req.Header))
		if v, ok := r.req.Body.(interface{ Body() any }); ok {
			if GetContentType(r.req.Header) == MIMEApplicationJSON {
				switch body := v.Body().(type) {
				case string:
					data := unsafe.Slice(unsafe.StringData(body), len(body))
					kvs = append(kvs, slog.Any("reqbody", json.RawMessage(data)))
				case []byte:
					kvs = append(kvs, slog.Any("reqbody", json.RawMessage(body)))
				case json.RawMessage:
					kvs = append(kvs, slog.Any("reqbody", body))
				default:
					kvs = append(kvs, slog.Any("reqbody", v.Body()))
				}
			} else {
				kvs = append(kvs, slog.Any("reqbody", v.Body()))
			}
		}
	}

	kvs = append(kvs,
		slog.String("cost", r.Cost().String()),
		slog.Int("statuscode", r.StatusCode()),
	)

	if r.resp != nil {
		kvs = append(kvs, slog.Any("respheaders", r.resp.Header))
	}

	if appendAttrs != nil {
		kvs = appendAttrs(r, kvs)
	}

	if err := r.Result(); err != nil {
		kvs = append(kvs, slog.Any("err", err))
	}

	slog.LogAttrs(r.req.Context(), level, "log the http request", kvs...)
}
