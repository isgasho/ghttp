package ghttp

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type (
	PostmanResponse struct {
		Args          H      `json:"args,omitempty"`
		Authenticated bool   `json:"authenticated,omitempty"`
		Cookies       H      `json:"cookies,omitempty"`
		Data          string `json:"data,omitempty"`
		Files         H      `json:"files,omitempty"`
		Form          H      `json:"form,omitempty"`
		Headers       H      `json:"headers,omitempty"`
		JSON          H      `json:"json,omitempty"`
		Method        string `json:"method,omitempty"`
		Origin        string `json:"origin,omitempty"`
		Token         string `json:"token,omitempty"`
		URL           string `json:"url,omitempty"`
		User          string `json:"user,omitempty"`
	}
)

func TestValues_Set(t *testing.T) {
	v := make(Values)
	assert.Len(t, v, 0)
	v.Set("k1", "v1")
	v.Set("k2", "v2")
	assert.Len(t, v, 2)
}

func TestValues_Decode(t *testing.T) {
	v := Values{
		"stringVal":   "hello",
		"stringSlice": []string{"hello", "hi"},
		"invalid":     testInvalidVal,
	}
	vv := v.Decode()
	assert.Len(t, vv, 2)
	assert.Equal(t, []string{"hello"}, vv["stringVal"])
	assert.Equal(t, []string{"hello", "hi"}, vv["stringSlice"])
}

func TestValues_URLEncode(t *testing.T) {
	v := Values{
		"expr":        "1+2",
		"stringVal":   "hello",
		"stringSlice": []string{"hello", "hi"},
	}
	// fmt.Printf("%q\n", v.URLEncode(true))
	want := "expr=1%2B2&stringSlice=hello&stringSlice=hi&stringVal=hello"
	assert.Equal(t, want, v.URLEncode(true))

	want = "expr=1+2&stringSlice=hello&stringSlice=hi&stringVal=hello"
	assert.Equal(t, want, v.URLEncode(false))
}

func TestValues_Marshal(t *testing.T) {
	v := Values{
		"text": "<p>Hello World</p>",
	}
	want := "{\"text\":\"<p>Hello World</p>\"}"
	assert.Equal(t, want, v.Marshal())
}

func TestCookies_Set(t *testing.T) {
	c := make(Cookies)
	assert.Len(t, c, 0)
	c.Set("n1", "v1")
	c.Set("n2", "v2")
	assert.Len(t, c, 2)
}

func TestCookies_Decode(t *testing.T) {
	c := Cookies{
		"n1": "v1",
		"n2": "v2",
	}
	assert.Len(t, c.Decode(), 2)
}

func TestFile_SetFilename(t *testing.T) {
	const (
		filename = "hello.txt"
	)

	file := FileFromReader(strings.NewReader("hello world"))
	file.SetFilename(filename)
	assert.Equal(t, filename, file.Filename)
}

func TestFile_SetMIME(t *testing.T) {
	const (
		mime = "text/html"
	)

	file := FileFromReader(strings.NewReader("<p>hello world</p>"))
	file.SetMIME(mime)
	assert.Equal(t, mime, file.MIME)
}

func TestFile_Read(t *testing.T) {
	const (
		msg = "hello world"
	)

	file := FileFromReader(strings.NewReader(msg))
	n, err := io.Copy(ioutil.Discard, file)
	if assert.NoError(t, err) {
		assert.Equal(t, int64(len(msg)), n)
	}
}

func TestOpen(t *testing.T) {
	const (
		fileExist    = "./testdata/testfile1.txt"
		fileNotExist = "./testdata/file_not_exist.txt"
	)

	f, err := Open(fileExist)
	if assert.NoError(t, err) {
		assert.NoError(t, f.Close())
	}

	_, err = Open(fileNotExist)
	assert.Error(t, err)

	assert.Panics(t, func() {
		_ = MustOpen(fileNotExist)
	})
}
