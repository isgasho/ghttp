# ghttp

**[ghttp](https://godoc.org/github.com/winterssy/ghttp)** is a simple, user-friendly and concurrent safe HTTP request library for Go, inspired by Python **[requests](https://requests.readthedocs.io)** .

![Build](https://img.shields.io/github/workflow/status/winterssy/ghttp/Test/master?logo=appveyor) [![codecov](https://codecov.io/gh/winterssy/ghttp/branch/master/graph/badge.svg)](https://codecov.io/gh/winterssy/ghttp) [![Go Report Card](https://goreportcard.com/badge/github.com/winterssy/ghttp)](https://goreportcard.com/report/github.com/winterssy/ghttp) [![GoDoc](https://godoc.org/github.com/winterssy/ghttp?status.svg)](https://godoc.org/github.com/winterssy/ghttp) [![License](https://img.shields.io/github/license/winterssy/ghttp.svg)](LICENSE)

## Notes

- `ghttp` now is under a beta state, use the latest version if you want to try it.
- The author does not provide any backward compatible guarantee at present.

## Features

- Requests-style APIs.
- GET, POST, PUT, PATCH, DELETE, etc.
- Easy set query params, headers and cookies.
- Easy send form, JSON or multipart payload.
- Easy set basic authentication or bearer token.
- Easy set proxy.
- Easy set context.
- Backoff retry mechanism.
- Automatic cookies management.
- Request and response interceptors.
- Rate limiter for handling outbound requests.
- Easy decode responses, raw data, text representation and unmarshal the JSON-encoded data.
- Export curl command.
- Friendly debugging.
- Concurrent safe.

## Install

```sh
go get -u github.com/winterssy/ghttp
```

## Usage

```go
import "github.com/winterssy/ghttp"
```

## Quick Start

The usages of `ghttp` are very similar to `net/http` , you can switch from it to `ghttp` easily. For example, if your HTTP request code like this:

```go
resp, err := http.Get("https://www.google.com")
```

Use `ghttp` you just need to change your code like this:

```go
resp, err := ghttp.Get("https://www.google.com").Raw()
```

You have two convenient ways to access the APIs of `ghttp` .

```go
const (
	url       = "https://httpbin.org/get"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36"
)

client := ghttp.New()
params := ghttp.Params{
	"k1": "v1",
	"k2": "v2",
}

// Go-style
req, err := ghttp.NewRequest("GET", url)
if err != nil {
	panic(err)
}
req.
	SetQuery(params).
	SetUserAgent(userAgent)
err = client.
	Do(req).
	EnsureStatusOk().
	Verbose(ioutil.Discard, false)
if err != nil {
	panic(err)
}

// Requests-style (Recommended)
err = client.
	Get(url,
		ghttp.WithQuery(params),
		ghttp.WithUserAgent(userAgent),
	).
	EnsureStatusOk().
	Verbose(os.Stdout, true)
if err != nil {
	panic(err)
}

// Output:
// > GET /get?k1=v1&k2=v2 HTTP/1.1
// > Host: httpbin.org
// > User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36
// >
// < HTTP/2.0 200 OK
// < Content-Length: 416
// < Server: gunicorn/19.9.0
// < Access-Control-Allow-Origin: *
// < Access-Control-Allow-Credentials: true
// < Date: Wed, 29 Jan 2020 10:13:20 GMT
// < Content-Type: application/json
// <
// {
//   "args": {
//     "k1": "v1",
//     "k2": "v2"
//   },
//   "headers": {
// 	   "Accept-Encoding": "gzip",
// 	   "Host": "httpbin.org",
// 	   "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36",
// 	   "X-Amzn-Trace-Id": "Root=1-5e315ac0-a890998aa8058ee3393cf5d4"
//   },
//   "origin": "8.8.8.8",
//   "url": "https://httpbin.org/get?k1=v1&k2=v2"
// }
```

## License

**[MIT](LICENSE)**