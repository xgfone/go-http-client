# A chained go HTTP client
[![Build Status](https://github.com/xgfone/go-http-client/actions/workflows/go.yml/badge.svg)](https://github.com/xgfone/go-http-client/actions/workflows/go.yml)
[![GoDoc](https://pkg.go.dev/badge/github.com/xgfone/go-http-client)](https://pkg.go.dev/github.com/xgfone/go-http-client)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](https://raw.githubusercontent.com/xgfone/go-http-client/master/LICENSE)
![Minimum Go Version](https://img.shields.io/github/go-mod/go-version/xgfone/go-http-client?label=Go%2B)
![Latest SemVer](https://img.shields.io/github/v/tag/xgfone/go-http-client?sort=semver)


## Install

```shell
$ go get -u github.com/xgfone/go-http-client
```

## Example

```go
package main

import (
	"context"
	"fmt"

	httpclient "github.com/xgfone/go-http-client"
)

func main() {
	// Create a new HTTP Client instead of the default.
	client := httpclient.Clone().
		SetContentType("application/json").
		AddAccept("application/json")

	// (Optional): Set other options.
	// client.
	//     SetReqBodyEncoder(httpclient.EncodeData).
	//     SetResponseHandler2xx(responseHandler2xx).
	//     SetResponseHandler4xx(responseHandler4xx).
	//     SetResponseHandler5xx(responseHandler5xx)

	var result struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	// client.SetBaseURL("http://localhost:12345").Get("/path")
	err := client.Get("http://127.0.0.1/path").
		AddAccept("application/xml").
		AddHeader("X-Header1", "value1").
		SetHeader("X-Header2", "value2").
		AddQuery("querykey1", "queryvalue1").
		SetQuery("querykey2", "queryvalue2").

		// Use the encoder referring to the request header "Content-Type"
		// to encode the body.
		SetBody(map[string]string{"username": "xgfone", "password": "123456"}).

		// Also use other body types:
		// SetBody([]byte(`this is the body as the type []byte`)).
		// SetBody("this is the body as the type string").
		// SetBody(bytes.NewBufferString("this is the body as the type io.Reader")).

		// Use the decoder referring to the response header "Content-Type"
		// to decode the body into result.
		Do(context.Background(), &result).
		Unwrap() // Close the response body and return the inner error.

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(result)
	}
}
```

There are also some convenient functions, such as `GetContext`, `PutContext`, `PostContext`, `DeleteContext`, etc.

```go
package main

import (
	"context"
	"fmt"

	httpclient "github.com/xgfone/go-http-client"
)

func main() {
	var result struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	err := httpclient.GetContext(context.Background(), "http://127.0.0.1/json_data", &result)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(result)
	}
}
```
