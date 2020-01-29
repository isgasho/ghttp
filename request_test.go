package ghttp

import (
	"bytes"
	"context"
	"encoding/xml"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequest(t *testing.T) {
	const (
		invalidMethod = "@"
	)

	_, err := NewRequest(invalidMethod, "https://httpbin.org/post")
	assert.Error(t, err)
}

func TestRequest_SetBody(t *testing.T) {
	client := New()
	body := bytes.NewBuffer([]byte{})
	resp := client.
		Post("https://httpbin.org/post",
			WithBody(body), // cover http.NoBody
			WithContentType("text/plain"),
		)
	assert.NoError(t, resp.Err())
}

func TestRequest_SetQuery(t *testing.T) {
	client := New()
	resp := new(PostmanResponse)
	err := client.
		Get("https://httpbin.org/get",
			WithQuery(testValues),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, testStringVal, resp.Args.GetString("stringVal"))
		assert.Equal(t, testStringSlice, resp.Args.GetStringSlice("stringSlice"))
	}

	_resp := new(PostmanResponse)
	err = client.
		Get("https://httpbin.org/get?k1=old&k2=old",
			WithQuery(Params{
				"k2": "new",
			}),
		).
		EnsureStatusOk().
		JSON(_resp)
	if assert.NoError(t, err) {
		assert.Equal(t, "old", _resp.Args.GetString("k1"))
		assert.Equal(t, "new", _resp.Args.GetString("k2"))
	}
}

func TestRequest_SetHost(t *testing.T) {
	const (
		host = "google.com"
	)

	client := New()
	resp := new(PostmanResponse)
	err := client.
		Get("https://httpbin.org/get",
			WithHost(host),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, host, resp.Headers.GetString("Host"))
	}
}

func TestRequest_SetHeaders(t *testing.T) {
	client := New()
	resp := new(PostmanResponse)
	err := client.
		Get("https://httpbin.org/get",
			WithHeaders(testHeaders),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, testStringVal, resp.Headers.GetString("String-Val"))
		assert.Equal(t, strings.Join(testStringSlice, ","), resp.Headers.GetString("String-Slice"))
	}
}

func TestRequest_SetUserAgent(t *testing.T) {
	const (
		userAgent = "Go-http-client"
	)

	client := New()
	resp := new(PostmanResponse)
	err := client.
		Get("https://httpbin.org/get",
			WithUserAgent(userAgent),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, userAgent, resp.Headers.GetString("User-Agent"))
	}
}

func TestRequest_SetOrigin(t *testing.T) {
	const (
		origin = "https://www.google.com"
	)

	client := New()
	resp := new(PostmanResponse)
	err := client.
		Get("https://httpbin.org/get",
			WithOrigin(origin),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, origin, resp.Headers.GetString("Origin"))
	}
}

func TestRequest_SetReferer(t *testing.T) {
	const (
		referer = "https://www.google.com"
	)

	client := New()
	resp := new(PostmanResponse)
	err := client.
		Get("https://httpbin.org/get",
			WithReferer(referer),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, referer, resp.Headers.GetString("Referer"))
	}
}

func TestRequest_SetCookies(t *testing.T) {
	client := New()
	resp := new(PostmanResponse)
	err := client.
		Get("https://httpbin.org/cookies",
			WithCookies(Cookies{
				"n1": "v1",
				"n2": "v2",
			}),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, "v1", resp.Cookies.GetString("n1"))
		assert.Equal(t, "v2", resp.Cookies.GetString("n2"))
	}
}

func TestRequest_SetContent(t *testing.T) {
	const (
		text = "hello world"
	)

	client := New()
	err := client.
		Post("https://httpbin.org/post",
			WithContent([]byte(text)),
			WithContentType("text/plain"),
		).
		EnsureStatusOk().
		Verbose(ioutil.Discard, true) // cover *bytes.Buffer body
	assert.NoError(t, err)

	resp := new(PostmanResponse)
	err = client.
		Post("https://httpbin.org/post",
			WithContent([]byte(text)),
			WithContentType("text/plain"),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, text, resp.Data)
	}
}

func TestRequest_SetText(t *testing.T) {
	const (
		text = "hello world"
	)

	client := New()
	err := client.
		Post("https://httpbin.org/post",
			WithText(text),
		).
		EnsureStatusOk().
		Err()
	assert.NoError(t, err)

	resp := new(PostmanResponse)
	err = client.
		Post("https://httpbin.org/post",
			WithText(text),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, text, resp.Data)
	}
}

func TestRequest_SetForm(t *testing.T) {
	client := New()
	resp := new(PostmanResponse)
	err := client.
		Post("https://httpbin.org/post",
			WithForm(testValues),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, testStringVal, resp.Form.GetString("stringVal"))
		assert.Equal(t, testStringSlice, resp.Form.GetStringSlice("stringSlice"))
	}
}

func TestRequest_SetJSON(t *testing.T) {
	client := New()
	err := client.
		Post("https://httpbin.org/post",
			WithJSON(map[string]interface{}{
				"num": math.Inf(1),
			}, true),
		).
		EnsureStatusOk().
		Verbose(ioutil.Discard, false) // cover resp.err != nil
	assert.Error(t, err)

	err = client.
		Post("https://httpbin.org/post",
			WithJSON(map[string]interface{}{
				"msg": "hi&hello",
				"num": 2019,
			}, true),
		).
		EnsureStatusOk().
		Verbose(ioutil.Discard, true) // cover *bytes.Reader body
	assert.NoError(t, err)

	resp := new(PostmanResponse)
	err = client.
		Post("https://httpbin.org/post",
			WithJSON(map[string]interface{}{
				"msg": "hi&hello",
				"num": 2019,
			}, false),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, "hi&hello", resp.JSON.GetString("msg"))
		assert.Equal(t, 2019, resp.JSON.GetNumber("num").Int())
	}
}

func TestRequest_SetXML(t *testing.T) {
	type plant struct {
		XMLName xml.Name `xml:"plant"`
		Id      int      `xml:"id,attr"`
		Name    string   `xml:"name"`
		Origin  []string `xml:"origin"`
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data plant
		err := xml.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		xml.NewEncoder(w).Encode(data)
	}))
	defer ts.Close()

	client := New()
	err := client.
		Post(ts.URL,
			WithXML(make(map[string]interface{})),
		).
		EnsureStatusOk().
		Err()
	assert.Error(t, err)

	origin := []string{"Ethiopia", "Brazil"}
	coffee := &plant{
		Id:     27,
		Name:   "Coffee",
		Origin: origin,
	}

	result := new(plant)
	err = client.
		Post(ts.URL,
			WithXML(coffee),
		).
		EnsureStatusOk().
		XML(result)
	if assert.NoError(t, err) {
		assert.Equal(t, 27, result.Id)
		assert.Equal(t, "Coffee", result.Name)
		assert.Equal(t, origin, result.Origin)
	}

	resp := client.
		Post(ts.URL,
			WithXML(coffee),
		).
		EnsureStatusOk().
		Prefetch()
	require.NoError(t, resp.Err())

	_, err = resp.Content()
	assert.NoError(t, err)

	_result := new(plant)
	err = resp.XML(_result)
	if assert.NoError(t, err) {
		assert.Equal(t, 27, _result.Id)
		assert.Equal(t, "Coffee", _result.Name)
		assert.Equal(t, origin, _result.Origin)
	}
}

func TestRequest_SetMultipart(t *testing.T) {
	// For Charles
	// client := New()
	// _ = client.SetProxyFromURL("http://127.0.0.1:7777")

	client := New()
	resp := new(PostmanResponse)
	err := client.
		Post("https://httpbin.org/post",
			WithMultipart(Files{
				"file": FileFromReader(&dummyBody{
					s:       "hello world",
					errFlag: errRead,
				}).SetFilename("dummyBody"),
			}, nil)).
		JSON(resp)
	if assert.NoError(t, err) {
		assert.Empty(t, resp.Files.GetString("file"))
	}

	files := Files{
		"file1": MustOpen("./testdata/testfile1.txt"),
		"file2": MustOpen("./testdata/testfile2.txt").
			SetFilename("testfile2.txt"),
		"file3": FileFromReader(bytes.NewReader([]byte("<p>This is a text file from memory</p>"))).
			SetFilename("testfile3.txt").
			SetMIME("text/html; charset=utf-8"),
		"file4": FileFromReader(strings.NewReader("Filename not specified")),
	}

	_resp := new(PostmanResponse)
	err = client.
		Post("https://httpbin.org/post",
			WithMultipart(files, testValues),
		).
		EnsureStatusOk().
		JSON(_resp)
	if assert.NoError(t, err) {
		assert.Equal(t, "testfile1.txt", _resp.Files.GetString("file1"))
		assert.Equal(t, "testfile2.txt", _resp.Files.GetString("file2"))
		assert.Equal(t, "<p>This is a text file from memory</p>", _resp.Files.GetString("file3"))
		assert.Equal(t, "Filename not specified", _resp.Files.GetString("file4"))
		assert.Equal(t, testStringVal, _resp.Form.GetString("stringVal"))
		assert.Equal(t, testStringSlice, _resp.Form.GetStringSlice("stringSlice"))
	}
}

func TestRequest_SetBasicAuth(t *testing.T) {
	const (
		username = "admin"
		password = "pass"
	)

	client := New()
	resp := new(PostmanResponse)
	err := client.
		Get("https://httpbin.org/basic-auth/admin/pass",
			WithBasicAuth(username, password),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.True(t, resp.Authenticated)
		assert.Equal(t, username, resp.User)
	}
}

func TestRequest_SetBearerToken(t *testing.T) {
	const (
		token = "ghttp"
	)

	client := New()
	resp := new(PostmanResponse)
	err := client.
		Get("https://httpbin.org/bearer",
			WithBearerToken(token),
		).
		EnsureStatusOk().
		JSON(resp)
	if assert.NoError(t, err) {
		assert.True(t, resp.Authenticated)
		assert.Equal(t, token, resp.Token)
	}
}

func TestRequest_SetContext(t *testing.T) {
	client := New()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp := client.
		Get("https://httpbin.org/delay/10",
			WithContext(ctx),
		)
	assert.Error(t, resp.Err())
}
