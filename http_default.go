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
	"net/http"
)

// DefaultClient is the default global client.
var DefaultClient = NewClient(http.DefaultClient)

// Clone is equal to DefaultClient.Clone().
func Clone() *Client { return DefaultClient.Clone() }

// Get is equal to DefaultClient.Get(url).
func Get(url string) *Request { return DefaultClient.Get(url) }

// Put is equal to DefaultClient.Put(url).
func Put(url string) *Request { return DefaultClient.Put(url) }

// Head is equal to DefaultClient.Head(url).
func Head(url string) *Request { return DefaultClient.Head(url) }

// Post is equal to DefaultClient.Post(url).
func Post(url string) *Request { return DefaultClient.Post(url) }

// Patch is equal to DefaultClient.Patch(url).
func Patch(url string) *Request { return DefaultClient.Patch(url) }

// Delete is equal to DefaultClient.Delete(url).
func Delete(url string) *Request { return DefaultClient.Delete(url) }

// Options is equal to DefaultClient.Options(url).
func Options(url string) *Request { return DefaultClient.Options(url) }

// GetContext is a convenient function to get the response data from the remote server.
func GetContext(ctx context.Context, url string, respBody any) error {
	return Get(url).Do(ctx, respBody).Unwrap()
}

// PutContext is a convenient function to send the request with the method PUT.
func PutContext(ctx context.Context, url string, respBody, reqBody any) error {
	return Put(url).SetBody(reqBody).Do(ctx, respBody).Unwrap()
}

// PostContext is a convenient function to send the request with the method Post.
func PostContext(ctx context.Context, url string, respBody, reqBody any) error {
	return Post(url).SetBody(reqBody).Do(ctx, respBody).Unwrap()
}

// PatchContext is a convenient function to send the request with the method Patch.
func PatchContext(ctx context.Context, url string, respBody, reqBody any) error {
	return Patch(url).SetBody(reqBody).Do(ctx, respBody).Unwrap()
}

// DeleteContext is a convenient function to send the request with the method Delete.
func DeleteContext(ctx context.Context, url string, respBody, reqBody any) error {
	return Delete(url).SetBody(reqBody).Do(ctx, respBody).Unwrap()
}
