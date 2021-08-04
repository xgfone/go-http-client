# go-http-client [![Build Status](https://api.travis-ci.com/xgfone/go-http-client.svg?branch=master)](https://travis-ci.com/github/xgfone/go-http-client) [![GoDoc](https://pkg.go.dev/badge/github.com/xgfone/go-http-client)](https://pkg.go.dev/github.com/xgfone/go-http-client) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](https://raw.githubusercontent.com/xgfone/go-http-client/master/LICENSE)

A chained go HTTP client.

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
	client := httpclient.Clone().SetContentType("application/json").AddAccept("application/json")

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

	err := client.
		Get("http://127.0.0.1/path").
		AddAccept("application/xml").
		SetBody(map[string]string{"username": "xgfone", "password": "123456"}).
		Do(context.Background(), &result).
		Close().
		Unwrap()

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(result)
	}
}
```
