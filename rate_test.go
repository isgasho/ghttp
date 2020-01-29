package ghttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestRegexpLimiter_Allow(t *testing.T) {
	tests := []struct {
		input *http.Request
		want  bool
	}{
		{
			&http.Request{URL: &neturl.URL{
				Scheme: "https",
				Host:   "httpbin.org",
				Path:   "/get",
			}},
			true,
		},
		{
			&http.Request{URL: &neturl.URL{
				Scheme: "https",
				Host:   "httpbin.org",
				Path:   "/head",
			}},
			true,
		},
		{
			&http.Request{URL: &neturl.URL{
				Scheme: "https",
				Host:   "httpbin.org",
				Path:   "/post",
			}},
			false,
		},
		{
			&http.Request{URL: &neturl.URL{
				Scheme: "https",
				Host:   "httpbin.org",
				Path:   "/delete",
			}},
			false,
		},
	}

	limiter := NewRegexpLimiter(rate.NewLimiter(1, 10))
	for _, test := range tests {
		assert.Equal(t, false, limiter.Allow(test.input))
	}

	var pattern *regexp.Regexp
	require.NotPanics(t, func() {
		pattern = regexp.MustCompile("post|delete")
	})

	limiter = NewRegexpLimiter(rate.NewLimiter(1, 10), pattern)
	for _, test := range tests {
		assert.Equal(t, test.want, limiter.Allow(test.input))
	}
}

func TestRegexpLimiter_NoPatterns(t *testing.T) {
	const (
		r           rate.Limit = 1
		bursts                 = 5
		concurrency            = 10
	)

	var counter uint64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&counter, 1)
	}))
	defer ts.Close()

	client := New().UseRateLimiter(NewRegexpLimiter(rate.NewLimiter(r, bursts)))
	wg := new(sync.WaitGroup)
	now := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			client.Get(ts.URL)
			wg.Done()
		}()
	}
	wg.Wait()

	if assert.Equal(t, uint64(concurrency), atomic.LoadUint64(&counter)) {
		assert.GreaterOrEqual(t, int64(time.Since(now)), int64((concurrency-bursts)*time.Second))
	}
}

func TestRegexpLimiter_MatchPatterns(t *testing.T) {
	const (
		r           = 1
		bursts      = 5
		concurrency = 10
	)

	var counter uint64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&counter, 1)
	}))
	defer ts.Close()

	var pattern *regexp.Regexp
	require.NotPanics(t, func() {
		pattern = regexp.MustCompile("127.0.0.1")
	})

	client := New().UseRateLimiter(NewRegexpLimiter(rate.NewLimiter(r, bursts), pattern))
	wg := new(sync.WaitGroup)
	now := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			client.Get(ts.URL)
			wg.Done()
		}()
	}
	wg.Wait()

	if assert.Equal(t, uint64(concurrency), atomic.LoadUint64(&counter)) {
		assert.GreaterOrEqual(t, int64(time.Since(now)), int64((concurrency-bursts)*time.Second))
	}
}

func TestRegexpLimiter_NotMatchPatterns(t *testing.T) {
	const (
		r           = 1
		bursts      = 5
		concurrency = 10
	)

	var counter uint64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&counter, 1)
	}))
	defer ts.Close()

	var pattern *regexp.Regexp
	require.NotPanics(t, func() {
		pattern = regexp.MustCompile("httpbin.org")
	})

	client := New().UseRateLimiter(NewRegexpLimiter(rate.NewLimiter(r, bursts), pattern))
	wg := new(sync.WaitGroup)
	now := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			client.Get(ts.URL)
			wg.Done()
		}()
	}
	wg.Wait()

	if assert.Equal(t, uint64(concurrency), atomic.LoadUint64(&counter)) {
		assert.Less(t, int64(time.Since(now)), int64((concurrency-bursts)*time.Second))
	}
}

func TestClient_LimitWithContext(t *testing.T) {
	const (
		r           = 1
		bursts      = 5
		concurrency = 10
	)

	var counter uint64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&counter, 1)
	}))
	defer ts.Close()

	client := New().UseRateLimiter(NewRegexpLimiter(rate.NewLimiter(r, bursts)))
	ctx, cancel := context.WithCancel(context.Background())
	wg := new(sync.WaitGroup)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			client.Get(ts.URL,
				WithContext(ctx),
			)
			wg.Done()
		}()
	}

	time.AfterFunc(time.Second, cancel)
	wg.Wait()

	assert.Less(t, atomic.LoadUint64(&counter), uint64(concurrency))
}
