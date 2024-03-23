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
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	ctx := context.Background()
	if r.req != nil {
		ctx = r.req.Context()
	}

	if !slog.Default().Enabled(ctx, level) {
		return
	}

	kvs := make([]slog.Attr, 0, 10)
	kvs = append(kvs,
		slog.String("method", r.Method()),
		slog.String("url", r.Url()),
	)

	if r.req != nil {
		kvs = append(kvs, slog.Any("reqheaders", r.req.Header))
		if ct := GetContentType(r.req.Header); _logreqbody(ct) {
			switch body := r.ReqBody().(type) {
			case string:
				data := unsafe.Slice(unsafe.StringData(body), len(body))
				kvs = append(kvs, slog.Any("reqbody", _bodydata(ct, data)))
			case []byte:
				kvs = append(kvs, slog.Any("reqbody", _bodydata(ct, body)))
			case json.RawMessage:
				kvs = append(kvs, slog.Any("reqbody", body))
			case fmt.Stringer:
				kvs = append(kvs, slog.String("reqbody", body.String()))
			case io.Reader: // Ignore
			default:
				kvs = append(kvs, slog.Any("reqbody", body))
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

	slog.LogAttrs(ctx, level, "log the http request", kvs...)
}

func _bodydata(ct string, data []byte) any {
	if ct == MIMEApplicationJSON && len(data) > 0 && (data[0] == '{' || data[0] == '[') {
		return json.RawMessage(data)
	}
	return unsafe.String(unsafe.SliceData(data), len(data))
}

func _logreqbody(ct string) bool {
	switch ct {
	case MIMEApplicationJSON,
		MIMEApplicationForm,
		MIMEApplicationXML,
		"text/plain":
		return true
	default:
		return false
	}
}
