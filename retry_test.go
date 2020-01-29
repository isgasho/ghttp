package ghttp

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	testBackoff = NewExponentialBackoff(1*time.Second, 30*time.Second, true)
)

func TestRetry(t *testing.T) {
	const (
		token = "ghttp"
	)

	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if r.Form.Get("token") == token {
			attempts++
		}
		if attempts == 5 {
			http.SetCookie(w, &http.Cookie{
				Name:  "uid",
				Value: "10086",
			})
		}
	}))
	defer ts.Close()

	trigger := func(resp *Response) bool {
		_, err := resp.Cookie("uid")
		return err != nil
	}
	client := New()
	cookie, err := client.
		Post(ts.URL,
			WithForm(Form{
				"token": token,
			}),
			WithRetry(NewRetrier(10, testBackoff, trigger)),
		).
		EnsureStatusOk().
		Cookie("uid")
	if assert.NoError(t, err) {
		assert.Equal(t, "10086", cookie.Value)
	}

	attempts = 0
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = client.
		Get(ts.URL,
			WithContext(ctx),
			WithRetry(NewRetrier(10, testBackoff, trigger)),
		).
		EnsureStatusOk().
		Cookie("uid")
	assert.Error(t, err)

	resp := client.
		Get("https://httpbin.org/get",
			WithRetry(NewRetrier(10, testBackoff)), // cover no triggers
		).
		EnsureStatusOk()
	assert.NoError(t, resp.Err())
}

func TestRetryWithBody(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		_ = r.ParseMultipartForm(1024)
		file, _, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		var buf bytes.Buffer
		_, err = buf.ReadFrom(file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(buf.Bytes())
	}))
	defer ts.Close()

	client := New()
	trigger := func(resp *Response) bool {
		return resp.Err() != nil || resp.StatusCode != http.StatusCreated
	}
	data, err := client.
		Post(ts.URL,
			WithMultipart(Files{
				"file": MustOpen("./testdata/testfile1.txt"),
			}, nil),
			WithRetry(NewRetrier(3, testBackoff, trigger)),
		).
		EnsureStatus(http.StatusCreated).
		Text()
	if assert.NoError(t, err) {
		assert.Equal(t, "testfile1.txt", data)
	}

	resp := client.Post("https://httpbin.org/post",
		WithBody(&dummyBody{
			s:       "hello world",
			errFlag: errRead,
		}),
		WithRetry(NewRetrier(3, testBackoff)),
	)
	assert.Error(t, resp.Err())
}
