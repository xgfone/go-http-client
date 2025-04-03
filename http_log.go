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
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"unsafe"

	"github.com/xgfone/go-toolkit/httpx"
)

type logAttr struct {
	Attrs []slog.Attr
}

var logattrpool = sync.Pool{New: func() any {
	return &logAttr{Attrs: make([]slog.Attr, 0, 16)}
}}

func getlogattrs() *logAttr     { return logattrpool.Get().(*logAttr) }
func putlogattrs(attr *logAttr) { logattrpool.Put(attr) }

// LogOnResponse is used to log the request on response.
//
// appendAttrs may be nil.
func LogOnResponse(resp *Response, level slog.Level, appendAttrs func(r *Response, attrs []slog.Attr) []slog.Attr) {
	_logOnResponse(resp, level, appendAttrs)
}

func logOnResponse(r *Response) { _logOnResponse(r, slog.LevelDebug, nil) }

func _logOnResponse(r *Response, level slog.Level,
	appendAttrs func(r *Response, kvs []slog.Attr) []slog.Attr,
) {
	ctx := context.Background()
	if r.req != nil {
		ctx = r.req.Context()
	}

	if !slog.Default().Enabled(ctx, level) {
		return
	}

	attr := getlogattrs()
	defer putlogattrs(attr)

	attr.Attrs = append(attr.Attrs,
		slog.String("method", r.Method()),
		slog.String("url", r.Url()),
	)

	if r.req != nil {
		attr.Attrs = append(attr.Attrs, slog.Any("reqheaders", r.req.Header))
		if ct := httpx.ContentType(r.req.Header); _logreqbody(ct) {
			switch body := r.ReqBody().(type) {
			case string:
				data := unsafe.Slice(unsafe.StringData(body), len(body))
				attr.Attrs = append(attr.Attrs, slog.Any("reqbody", _bodydata(ct, data)))
			case []byte:
				attr.Attrs = append(attr.Attrs, slog.Any("reqbody", _bodydata(ct, body)))
			case json.RawMessage:
				attr.Attrs = append(attr.Attrs, slog.Any("reqbody", body))
			case fmt.Stringer:
				attr.Attrs = append(attr.Attrs, slog.String("reqbody", body.String()))
			case io.Reader: // Ignore
			default:
				attr.Attrs = append(attr.Attrs, slog.Any("reqbody", body))
			}
		}
	}

	attr.Attrs = append(attr.Attrs,
		slog.String("cost", r.Cost().String()),
		slog.Int("statuscode", r.StatusCode()),
	)

	if r.resp != nil {
		attr.Attrs = append(attr.Attrs, slog.Any("respheaders", r.resp.Header))
	}

	if appendAttrs != nil {
		attr.Attrs = appendAttrs(r, attr.Attrs)
	}

	if err := r.Result(); err != nil {
		attr.Attrs = append(attr.Attrs, slog.Any("err", err))
	}

	slog.LogAttrs(ctx, level, "log the http request", attr.Attrs...)
}

func _bodydata(ct string, data []byte) any {
	if ct == httpx.MIMEApplicationJSON && len(data) > 0 && (data[0] == '{' || data[0] == '[') {
		return json.RawMessage(data)
	}
	return unsafe.String(unsafe.SliceData(data), len(data))
}

func _logreqbody(ct string) bool {
	switch ct {
	case httpx.MIMEApplicationJSON,
		httpx.MIMEApplicationForm,
		httpx.MIMEApplicationXML,
		httpx.MIMETextPlain,
		httpx.MIMETextHTML:
		return true
	default:
		return false
	}
}
