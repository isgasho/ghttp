package ghttp

import (
	"compress/gzip"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	noTransport = &dummyTransport{}
	noJar       = &dummyJar{}
)

type (
	dummyTransport struct{}
	dummyJar       struct{}
)

func (*dummyTransport) RoundTrip(*http.Request) (*http.Response, error) { return nil, nil }

func (*dummyJar) SetCookies(*url.URL, []*http.Cookie) {}

func (*dummyJar) Cookies(*url.URL) []*http.Cookie { return nil }

func TestNewWithHTTPClient(t *testing.T) {
	c := &http.Client{}
	client := NewWithHTTPClient(c)
	assert.Equal(t, c, client.Client)
}

func TestClient_SetTransport(t *testing.T) {
	client := New().SetTransport(noTransport)
	assert.Equal(t, noTransport, client.Transport)
}

func TestClient_DisableRedirect(t *testing.T) {
	client := New().DisableRedirect()
	assert.True(t, client.CheckRedirect(nil, nil) == http.ErrUseLastResponse)
}

func TestClient_EnableSession(t *testing.T) {
	client := New().EnableSession(noJar)
	assert.Equal(t, noJar, client.Jar)
}

func TestClient_SetTimeout(t *testing.T) {
	const (
		timeout = 3 * time.Second
	)

	client := New().SetTimeout(timeout)
	assert.Equal(t, timeout, client.Timeout)
}

func TestClient_SetProxy(t *testing.T) {
	client := New().SetProxy(nil)
	transport, ok := client.Transport.(*http.Transport)
	if assert.True(t, ok) && assert.NotNil(t, transport) {
		assert.Nil(t, transport.Proxy)
	}
}

func TestClient_SetProxyFromURL(t *testing.T) {
	const (
		proxyURL = "http://127.0.0.1:1081"
	)

	client := New().SetProxyFromURL(proxyURL)
	transport, ok := client.Transport.(*http.Transport)
	if assert.True(t, ok) && assert.NotNil(t, transport) {
		assert.NotNil(t, transport.Proxy)
	}
	req, _ := http.NewRequest("GET", "https://www.google.com", nil)
	fixedURL, err := transport.Proxy(req)
	if assert.NoError(t, err) {
		assert.Equal(t, proxyURL, fixedURL.String())
	}
}

func TestClient_DisableProxy(t *testing.T) {
	client := New().DisableProxy()
	transport, ok := client.Transport.(*http.Transport)
	if assert.True(t, ok) && assert.NotNil(t, transport) {
		assert.Nil(t, transport.Proxy)
	}
}

func TestClient_SetTLSClientConfig(t *testing.T) {
	config := &tls.Config{}
	client := New().SetTLSClientConfig(config)
	transport, ok := client.Transport.(*http.Transport)
	if assert.True(t, ok) && assert.NotNil(t, transport) {
		assert.NotNil(t, transport.TLSClientConfig)
	}
}

func TestClient_AppendClientCerts(t *testing.T) {
	cert := tls.Certificate{}
	client := New().AppendClientCerts(cert)
	transport, ok := client.Transport.(*http.Transport)
	if assert.True(t, ok) && assert.NotNil(t, transport) && assert.NotNil(t, transport.TLSClientConfig) {
		assert.Len(t, transport.TLSClientConfig.Certificates, 1)
	}
}

func TestClient_AppendRootCerts(t *testing.T) {
	const (
		pemFileExist    = "./testdata/root-ca.pem"
		pemFileNotExist = "./testdata/root-ca-not-exist.pem"
	)

	var client *Client
	require.NotPanics(t, func() {
		client = New().AppendRootCerts(pemFileExist)
	})

	transport, ok := client.Transport.(*http.Transport)
	if assert.True(t, ok) && assert.NotNil(t, transport) && assert.NotNil(t, transport.TLSClientConfig) {
		assert.NotNil(t, transport.TLSClientConfig.RootCAs)
	}

	assert.Panics(t, func() {
		client.AppendRootCerts(pemFileNotExist)
	})
}

func TestClient_DisableVerify(t *testing.T) {
	client := New().DisableVerify()
	transport, ok := client.Transport.(*http.Transport)
	if assert.True(t, ok) && assert.NotNil(t, transport) && assert.NotNil(t, transport.TLSClientConfig) {
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
	}
}

func TestClient_SetCookies(t *testing.T) {
	const (
		token = "ghttp"
	)

	var (
		cookie = &http.Cookie{
			Name:  "token",
			Value: token,
		}
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("token")
		if err != nil || c.Value != token {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer ts.Close()

	client := New()
	assert.Panics(t, func() {
		_ = client.SetCookies(ts.URL, cookie)
	})
	assert.NotPanics(t, func() {
		_ = client.EnableSession().SetCookies(ts.URL, cookie)
	})
}

func TestClient_OnBeforeRequest(t *testing.T) {
	errMethodNotAllowed := errors.New("method not allowed")
	hook := func(req *Request) error {
		if req.Method == "DELETE" {
			return errMethodNotAllowed
		}

		return nil
	}

	client := New().OnBeforeRequest(hook)
	resp := client.
		Delete("https://httpbin.org/delete",
			WithForm(Form{
				"uid": "10086",
			}),
		)
	assert.Equal(t, errMethodNotAllowed, resp.Err())
}

func TestClient_OnAfterResponse(t *testing.T) {
	errUnauthorized := errors.New("illegal user")
	hook := func(resp *Response) error {
		if resp.StatusCode == http.StatusUnauthorized {
			return errUnauthorized
		}
		return nil
	}

	client := New().OnAfterResponse(hook)
	resp := client.
		Get("https://httpbin.org/basic-auth/admin/pass",
			WithBasicAuth("user", "pass"),
		)
	assert.Equal(t, errUnauthorized, resp.Err())
}

func TestClient_FilterCookie(t *testing.T) {
	const (
		invalidURL = "http://127.0.0.1:8080^"
	)

	var (
		c = &http.Cookie{
			Name:  "uid",
			Value: "10086",
		}
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, c)
	}))
	defer ts.Close()

	client := New().EnableSession()
	resp := client.
		Get(ts.URL).
		EnsureStatusOk()
	require.NoError(t, resp.Err())

	_, err := client.FilterCookie(ts.URL, "uuid")
	assert.Equal(t, ErrNoCookie, err)

	cookie, err := client.FilterCookie(ts.URL, c.Name)
	if assert.NoError(t, err) {
		assert.Equal(t, c.Value, cookie.Value)
	}

	_, err = client.FilterCookie(invalidURL, c.Name)
	assert.Error(t, err)

	client = New()
	cookies, err := client.FilterCookies(ts.URL)
	assert.Equal(t, ErrNilCookieJar, err)
	assert.Empty(t, cookies)
}

func TestClient_Do(t *testing.T) {
	req, err := NewRequest("GET", "https://httpbin.org/get")
	require.NoError(t, err)

	client := New()
	resp := client.
		Do(req).
		EnsureStatusOk()
	assert.NoError(t, resp.Err())
}

func TestAutoGzip(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")

		q := r.URL.Query().Get("q")
		if q == "" {
			return
		}

		zw := gzip.NewWriter(w)
		_, _ = zw.Write([]byte(q))
		zw.Close()
	}))
	defer ts.Close()

	// transport := DefaultTransport()
	// transport.DialContext = printLocalDial
	// client := New().SetTransport(transport)
	// for {
	// 	go func() {
	// 		data, err := client.
	// 			Get(ts.URL,
	// 				WithQuery(Params{
	// 					"q": "hello",
	// 				}),
	// 				WithHeaders(Headers{
	// 					"Accept-Encoding": "gzip",
	// 				}),
	// 			).Text()
	// 		if err != nil {
	// 			return
	// 		}
	// 		fmt.Println(data)
	// 	}()
	//
	// 	go func() {
	// 		data, err := client.
	// 			Get(ts.URL,
	// 				WithQuery(Params{
	// 					"q": "hi",
	// 				}),
	// 				WithHeaders(Headers{
	// 					"Accept-Encoding": "gzip",
	// 				}),
	// 			).Text()
	// 		if err != nil {
	// 			return
	// 		}
	// 		fmt.Println(data)
	// 	}()
	//
	// 	time.Sleep(1 * time.Second)
	// }

	client := New()
	resp := client.
		Get(ts.URL,
			WithHeaders(Headers{
				"Accept-Encoding": "gzip",
			}),
		)
	assert.NoError(t, resp.Err())

	data, err := client.
		Get(ts.URL,
			WithQuery(Params{
				"q": "hello",
			}),
			WithHeaders(Headers{
				"Accept-Encoding": "gzip",
			}),
		).Text()
	if assert.NoError(t, err) {
		assert.Equal(t, "hello", data)
	}
}

func TestGlobalFuncs(t *testing.T) {
	testHead(t)
	testGet(t)
	testPost(t)
	testPut(t)
	testPatch(t)
	testDelete(t)
	testSend(t)
}

func testHead(t *testing.T) {
	resp :=
		Head("https://httpbin.org").
			EnsureStatusOk()
	assert.NoError(t, resp.Err())
}

func testGet(t *testing.T) {
	resp :=
		Get("https://httpbin.org/get").
			EnsureStatusOk()
	assert.NoError(t, resp.Err())
}

func testPost(t *testing.T) {
	resp :=
		Post("https://httpbin.org/post").
			EnsureStatusOk()
	assert.NoError(t, resp.Err())
}

func testPut(t *testing.T) {
	resp :=
		Put("https://httpbin.org/put").
			EnsureStatusOk()
	assert.NoError(t, resp.Err())
}

func testPatch(t *testing.T) {
	resp :=
		Patch("https://httpbin.org/patch").
			EnsureStatusOk()
	assert.NoError(t, resp.Err())
}

func testDelete(t *testing.T) {
	resp :=
		Delete("https://httpbin.org/delete").
			EnsureStatusOk()
	assert.NoError(t, resp.Err())
}

func testSend(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case MethodConnect, MethodOptions, MethodTrace:
			w.WriteHeader(http.StatusMethodNotAllowed)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	err :=
		Send(MethodConnect, ts.URL).
			EnsureStatus(http.StatusMethodNotAllowed).
			Verbose(ioutil.Discard, true) // cover resp.ContentLength == 0
	assert.NoError(t, err)

	resp :=
		Send(MethodOptions, ts.URL).
			EnsureStatus(http.StatusMethodNotAllowed)
	assert.NoError(t, resp.Err())

	resp =
		Send(MethodTrace, ts.URL).
			EnsureStatus(http.StatusMethodNotAllowed)
	assert.NoError(t, resp.Err())
}
