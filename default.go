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

// GetJSON is a convenient function to get the JSON data from the remote server.
func GetJSON(url string, respBody any) error {
	return GetJSONContext(context.Background(), url, respBody)
}

// PutJSON is a convenient function to send the JSON data with the method PUT.
func PutJSON(url string, respBody any, reqBody any) error {
	return PutJSONContext(context.Background(), url, respBody, reqBody)
}

// PostJSON is a convenient function to put the JSON data with the method POST.
func PostJSON(url string, respBody any, reqBody any) error {
	return PostJSONContext(context.Background(), url, respBody, reqBody)
}

// PatchJSON is a convenient function to put the JSON data with the method PATCH.
func PatchJSON(url string, respBody any, reqBody any) error {
	return PatchJSONContext(context.Background(), url, respBody, reqBody)
}

// DeleteJSON is a convenient function to the JSON data to with the method DELETE.
func DeleteJSON(url string, respBody any, reqBody any) error {
	return DeleteJSONContext(context.Background(), url, respBody, reqBody)
}

// GetJSONContext is a convenient function to get the JSON data from the remote server.
func GetJSONContext(c context.Context, url string, respBody any) error {
	return Get(url).Do(c, respBody).Close().Unwrap()
}

// PutJSONContext is a convenient function to send the JSON data with the method PUT.
func PutJSONContext(c context.Context, url string, respBody any, reqBody any) error {
	return requestJSON(c, Put(url), respBody, reqBody)
}

// PostJSONContext is a convenient function to put the JSON data with the method POST.
func PostJSONContext(c context.Context, url string, respBody any, reqBody any) error {
	return requestJSON(c, Post(url), respBody, reqBody)
}

// PatchJSONContext is a convenient function to put the JSON data with the method PATCH.
func PatchJSONContext(c context.Context, url string, respBody any, reqBody any) error {
	return requestJSON(c, Patch(url), respBody, reqBody)
}

// DeleteJSONContext is a convenient function to the JSON data to with the method DELETE.
func DeleteJSONContext(c context.Context, url string, respBody any, reqBody any) error {
	return requestJSON(c, Delete(url), respBody, reqBody)
}

func requestJSON(c context.Context, req *Request, respBody any, reqBody any) error {
	return req.SetBody(reqBody).Do(c, respBody).Unwrap()
}
