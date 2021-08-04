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

import "net/http"

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
