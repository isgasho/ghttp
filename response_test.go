package ghttp

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

const (
	testFileName = "testdata.json"
)

func TestResponse_Raw(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()

	client := New()
	resp, err := client.
		Send(MethodGet, ts.URL).
		Raw()
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestResponse_Prefetch(t *testing.T) {
	client := New()
	r := client.
		Get("https://www.google.com/404").
		EnsureStatusOk().
		Prefetch()
	assert.Error(t, r.Err())

	r = client.
		Get("https://httpbin.org/get",
			WithQuery(Params{
				"k1": "v1",
				"k2": "v2",
			}),
		).
		Prefetch()
	require.NoError(t, r.Err())

	_, err := r.Content()
	assert.NoError(t, err)

	data, err := r.Text()
	if assert.NoError(t, err) {
		assert.NotEmpty(t, data)
	}

	resp := new(PostmanResponse)
	err = r.JSON(resp)
	if assert.NoError(t, err) {
		assert.Equal(t, "v1", resp.Args.GetString("k1"))
		assert.Equal(t, "v2", resp.Args.GetString("k2"))
	}

	err = r.Save(testFileName, 0664)
	assert.NoError(t, err)

	err = r.Verbose(ioutil.Discard, true)
	assert.NoError(t, err)
}

func TestResponse_Text(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var _w io.Writer
		q := r.URL.Query().Get("e")
		switch {
		case strings.EqualFold(q, "UTF-8"):
			_w = transform.NewWriter(w, unicode.UTF8.NewEncoder())
		case strings.EqualFold(q, "GBK"):
			_w = transform.NewWriter(w, simplifiedchinese.GBK.NewEncoder())
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_w.Write([]byte("你好世界"))
	}))
	defer ts.Close()

	client := New()
	want := "你好世界"
	data, err := client.
		Get(ts.URL,
			WithQuery(Params{
				"e": "utf-8",
			}),
		).
		EnsureStatusOk().
		Text()
	if assert.NoError(t, err) {
		assert.Equal(t, want, data)
	}

	data, err = client.
		Get(ts.URL,
			WithQuery(Params{
				"e": "gbk",
			}),
		).
		EnsureStatus2xx().
		Text(simplifiedchinese.GBK)
	if assert.NoError(t, err) {
		assert.Equal(t, want, data)
	}
}

func TestResponse_JSON(t *testing.T) {
	data := make(map[string]interface{})
	client := New()
	err := client.
		Get("https://www.google.com/404").
		EnsureStatusOk().
		JSON(&data)
	assert.Error(t, err)
}

func TestResponse_H(t *testing.T) {
	client := New()
	h, err := client.
		Get("https://httpbin.org/get",
			WithQuery(Params{
				"k1": "v1",
				"k2": "v2",
			}),
		).
		EnsureStatusOk().
		H()
	if assert.NoError(t, err) {
		args := h.GetH("args")
		assert.Equal(t, "v1", args.GetString("k1"))
		assert.Equal(t, "v2", args.GetString("k2"))
	}

	h, err = client.
		Post("https://httpbin.org/post",
			WithJSON(map[string]interface{}{
				"songs": []map[string]interface{}{
					{
						"id":     29947420,
						"name":   "Fade",
						"artist": "Alan Walker",
						"album":  "Fade",
					},
					{
						"id":     444269135,
						"name":   "Alone",
						"artist": "Alan Walker",
						"album":  "Alone",
					},
				},
			}, true),
		).
		EnsureStatusOk().
		H()
	require.NoError(t, err)

	data := h.GetH("json").GetHSlice("songs")
	if assert.Len(t, data, 2) {
		fade := data[0]
		assert.Equal(t, int64(29947420), fade.GetNumber("id").Int64())
		assert.Equal(t, "Fade", fade.GetString("name"))
		assert.Equal(t, "Alan Walker", fade.GetString("artist"))
	}
}

func TestResponse_XML(t *testing.T) {
	type plant struct {
		XMLName xml.Name `xml:"plant"`
		Id      int      `xml:"id,attr"`
		Name    string   `xml:"name"`
		Origin  []string `xml:"origin"`
	}

	var data plant
	client := New()
	err := client.
		Get("https://www.google.com/404").
		EnsureStatusOk().
		XML(&data)
	assert.Error(t, err)
}

func TestResponse_Dump(t *testing.T) {
	client := New()
	_, err := client.
		Get("https://www.google.com/404").
		EnsureStatusOk().
		Dump(true)
	assert.Error(t, err)

	_, err = client.
		Get("https://www.google.com").
		EnsureStatusOk().
		Dump(true)
	assert.NoError(t, err)
}

func TestResponse_Cookie(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:  "uid",
			Value: "10086",
		})
	}))
	defer ts.Close()

	client := New()
	resp := client.
		Get(ts.URL).
		EnsureStatusOk()

	cookie, err := resp.Cookie("uid")
	if assert.NoError(t, err) {
		assert.Equal(t, "10086", cookie.Value)
	}

	_, err = resp.Cookie("uuid")
	assert.Equal(t, ErrNoCookie, err)
}

func TestResponse_EnsureStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case MethodGet:
			w.WriteHeader(http.StatusOK)
		case MethodPost:
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	defer ts.Close()

	client := New()
	err := client.
		Get(ts.URL).
		EnsureStatusOk().
		Err()
	assert.NoError(t, err)

	err = client.
		Post(ts.URL).
		EnsureStatus2xx().
		Err()
	assert.NoError(t, err)

	err = client.
		Patch(ts.URL).
		EnsureStatus2xx().
		Err()
	assert.Error(t, err)

	err = client.
		Patch(ts.URL).
		EnsureStatusOk().
		EnsureStatus2xx().
		Err()
	assert.Error(t, err)

	err = client.
		Delete(ts.URL).
		EnsureStatus(http.StatusForbidden).
		Err()
	assert.NoError(t, err)

	err = client.
		Delete(ts.URL).
		EnsureStatus(http.StatusOK).
		Err()
	assert.Error(t, err)
}

func TestResponse_Save(t *testing.T) {
	client := New()
	err := client.
		Get("https://www.google.com/404").
		EnsureStatusOk().
		Save(testFileName, 0664)
	assert.Error(t, err)

	err = client.
		Get("https://httpbin.org/get").
		EnsureStatusOk().
		Save(testFileName, 0664)
	assert.NoError(t, err)
}

func TestResponse_Verbose(t *testing.T) {
	client := New()
	err := client.
		Get("https://httpbin.org/get",
			WithQuery(Params{
				"uid": "10086",
			}),
		).
		EnsureStatusOk().
		Verbose(ioutil.Discard, true)
	assert.NoError(t, err)

	err = client.
		Post("https://httpbin.org/post",
			WithForm(Form{
				"uid": "10086",
			}),
		).
		Verbose(ioutil.Discard, true)
	assert.NoError(t, err)

	req, err := NewRequest(MethodPost, "https://httpbin.org/post",
		WithMultipart(Files{
			"file1": MustOpen("./testdata/testfile1.txt"),
			"file2": MustOpen("./testdata/testfile2.txt"),
		}, nil),
	)
	require.NoError(t, err)

	req.TransferEncoding = []string{"chunked"}
	req.Close = true
	err = client.
		Do(req).
		Verbose(ioutil.Discard, true)
	assert.NoError(t, err)
}
