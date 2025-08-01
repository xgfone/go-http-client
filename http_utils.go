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
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var errMissingContentType = errors.New("missing header Content-Type")

var bufpool = sync.Pool{New: func() any {
	return bytes.NewBuffer(make([]byte, 0, 1024))
}}

func getBuffer() *bytes.Buffer    { return bufpool.Get().(*bytes.Buffer) }
func putBuffer(buf *bytes.Buffer) { buf.Reset(); bufpool.Put(buf) }

type bytesT struct{ Data []byte }

var bytespool = sync.Pool{New: func() any {
	return &bytesT{Data: make([]byte, 1024)}
}}

func getBytes() *bytesT  { return bytespool.Get().(*bytesT) }
func putBytes(b *bytesT) { bytespool.Put(b) }

func cloneQuery(query url.Values) url.Values {
	return url.Values(http.Header(query).Clone())
}

func getContentType(header http.Header) (mime string) {
	mime = header.Get("Content-Type")
	if index := strings.IndexByte(mime, ';'); index > -1 {
		mime = strings.TrimSpace(mime[:index])
	}
	return
}

func shouldDecodeResponseBody(resp *http.Response) bool {
	if resp.ContentLength > 0 {
		return true
	}

	switch resp.Header.Get("Content-Length") {
	case "", "-1":
	case "0":
		return false
	default:
		return true
	}

	switch resp.Header.Get("Transfer-Encoding") {
	case "", "null":
	default:
		return true
	}

	// We do not support scenarios where both "Content-Encoding" and "Transfer-Encoding"
	// headers are absent, but the "Connection" header is present and set to "close".

	if resp.Header.Get("Connection") == "close" {
		const s = "use HTTP/1.0 response, but missing 'Content-Length' or 'Transfer-Encoding'"
		if r := resp.Request; r != nil {
			slog.Warn(s, "host", r.Host, "path", r.URL.Path, "method", r.Method)
		} else {
			slog.Warn(s)
		}
	}

	if resp.Request.Method == http.MethodGet {
		return true
	}

	return false
}

// EncodeData encodes the data by contentType and writes it into w.
//
// It writes the data into w directly instead of encoding it
// if contentType is one of the types:
//   - []byte
//   - string
//   - io.Reader
//   - io.WriterTo
func EncodeData(w io.Writer, contentType string, data any) (err error) {
	switch v := data.(type) {
	case *bytes.Buffer:
		_, err = w.Write(v.Bytes())
	case io.Reader:
		buf := getBytes()
		_, err = io.CopyBuffer(w, v, buf.Data)
		putBytes(buf)
	case []byte:
		_, err = w.Write(v)
	case string:
		_, err = io.WriteString(w, v)
	case io.WriterTo:
		_, err = v.WriteTo(w)
	default:
		switch contentType {
		case "":
			err = errMissingContentType
		case "application/xml":
			err = xml.NewEncoder(w).Encode(data)
		case "application/json":
			enc := json.NewEncoder(w)
			enc.SetEscapeHTML(false)
			err = enc.Encode(data)
		case "application/x-www-form-urlencoded":
			switch v := data.(type) {
			case url.Values:
				_, err = io.WriteString(w, v.Encode())

			case map[string]string:
				vs := make(url.Values, len(v))
				for key, value := range v {
					vs[key] = []string{value}
				}
				_, err = io.WriteString(w, vs.Encode())

			case map[string]any:
				vs := make(url.Values, len(v))
				for key, value := range v {
					vs[key] = []string{fmt.Sprint(value)}
				}
				_, err = io.WriteString(w, vs.Encode())

			case interface{ MarshalForm() ([]byte, error) }:
				var _data []byte
				if _data, err = v.MarshalForm(); err == nil {
					_, err = w.Write(_data)
				}

			default:
				err = fmt.Errorf("not support to encode %T to application/x-www-form-urlencoded", data)
			}
		default:
			err = fmt.Errorf("unsupported Content-Type '%s'", contentType)
		}
	}
	return
}

// DecodeFromReader reads the data from r and decode it to dst.
//
// If ct is equal to "application/xml" or "application/json", it will use
// the xml or json decoder to decode the data. Or returns an error.
func DecodeFromReader(dst any, ct string, r io.Reader) (err error) {
	switch ct {
	case "":
		err = errMissingContentType
	case "application/xml":
		err = xml.NewDecoder(r).Decode(dst)
	case "application/json":
		err = json.NewDecoder(r).Decode(dst)
	default:
		err = fmt.Errorf("unsupported response Content-Type '%s'", ct)
	}
	return
}

// DecodeResponseBody is a response handler to decode the response body
// into dst.
func DecodeResponseBody(dst any, resp *http.Response) (err error) {
	if dst == nil || resp.StatusCode == 204 || !shouldDecodeResponseBody(resp) {
		return
	}
	return DecodeFromReader(dst, getContentType(resp.Header), resp.Body)
}

// ReadResponseBodyAsError is a response handler to read the response body
// as the error to be returned.
func ReadResponseBodyAsError(dst any, resp *http.Response) error {
	if resp.StatusCode >= 300 && resp.StatusCode < 400 { // For 3xx
		return nil
	}

	err := Error{Code: resp.StatusCode}
	err.Err = fmt.Errorf("got status code %d", resp.StatusCode)

	if req := resp.Request; req != nil {
		err.Method = req.Method
		err.URL = req.URL.String()
	}

	buf := getBuffer()
	bytebuf := getBytes()
	_, _ = io.CopyBuffer(buf, resp.Body, bytebuf.Data)
	err.Data = buf.String()
	putBytes(bytebuf)
	putBuffer(buf)

	return err
}
