package ghttp

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	neturl "net/url"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

const (
	defaultTimeout = 120 * time.Second
)

var (
	// GlobalClient is a global Client used by the global functions such as Get, Post, etc.
	GlobalClient = New()
)

type (
	// Client wraps the raw HTTP client.
	// Do not modify the client across Goroutines!
	// You should reuse it as possible after initialized.
	Client struct {
		*http.Client
		limiter            Limiter
		beforeRequestHooks []BeforeRequestHook
		afterResponseHooks []AfterResponseHook
	}
)

// New returns a new Client with a preset HTTP client.
func New() *Client {
	client := &http.Client{
		Transport: DefaultTransport(),
		Timeout:   defaultTimeout,
	}
	return NewWithHTTPClient(client)
}

// NewWithHTTPClient returns a new Client given an *http.Client.
func NewWithHTTPClient(client *http.Client) *Client {
	return &Client{
		Client: client,
	}
}

// UseRateLimiter specifies a rate-limiter for c to limit outbound requests.
func (c *Client) UseRateLimiter(limiter Limiter) *Client {
	c.limiter = limiter
	return c
}

// SetTransport sets transport of the HTTP client.
func (c *Client) SetTransport(transport http.RoundTripper) *Client {
	c.Transport = transport
	return c
}

// SetRedirect sets policy of the HTTP client for handling redirects.
func (c *Client) SetRedirect(policy func(req *http.Request, via []*http.Request) error) *Client {
	c.CheckRedirect = policy
	return c
}

func disableRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// DisableRedirect makes the HTTP client not follow redirects.
func (c *Client) DisableRedirect() *Client {
	return c.SetRedirect(disableRedirect)
}

// EnableSession makes c persist cookies across requests given an optional cookie jar.
// If the jar not specified, ghttp will constructor one provided by net/http/cookiejar package.
func (c *Client) EnableSession(jar ...http.CookieJar) *Client {
	var _jar http.CookieJar
	if len(jar) > 0 {
		_jar = jar[0]
	} else {
		_jar, _ = cookiejar.New(&cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		})
	}
	c.Jar = _jar
	return c
}

// SetTimeout sets timeout of the HTTP client, default is 120s.
func (c *Client) SetTimeout(timeout time.Duration) *Client {
	c.Timeout = timeout
	return c
}

// SetProxy sets proxy of the HTTP client.
func (c *Client) SetProxy(proxy func(*http.Request) (*neturl.URL, error)) *Client {
	if t, ok := c.Transport.(*http.Transport); ok && t != nil {
		t.Proxy = proxy
	}
	return c
}

// SetProxyFromURL sets proxy of the HTTP client from a URL.
func (c *Client) SetProxyFromURL(url string) *Client {
	if fixedURL, err := neturl.Parse(url); err == nil {
		c.SetProxy(http.ProxyURL(fixedURL))
	}
	return c
}

// DisableProxy makes the HTTP client not use proxy.
func (c *Client) DisableProxy() *Client {
	return c.SetProxy(nil)
}

// SetTLSClientConfig sets TLS configuration of the HTTP client.
func (c *Client) SetTLSClientConfig(config *tls.Config) *Client {
	if t, ok := c.Transport.(*http.Transport); ok && t != nil {
		t.TLSClientConfig = config
	}
	return c
}

// AppendClientCerts appends client certificates to the HTTP client.
func (c *Client) AppendClientCerts(certs ...tls.Certificate) *Client {
	if t, ok := c.Transport.(*http.Transport); ok && t != nil {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		t.TLSClientConfig.Certificates = append(t.TLSClientConfig.Certificates, certs...)
	}
	return c
}

// AppendRootCerts appends root certificates from a pem file to the HTTP client.
// If there is an error while reading pemFile, it will panic.
func (c *Client) AppendRootCerts(pemFile string) *Client {
	if t, ok := c.Transport.(*http.Transport); ok && t != nil {
		pemCerts, err := ioutil.ReadFile(pemFile)
		if err != nil {
			panic(err)
		}

		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		if t.TLSClientConfig.RootCAs == nil {
			t.TLSClientConfig.RootCAs = x509.NewCertPool()
		}
		t.TLSClientConfig.RootCAs.AppendCertsFromPEM(pemCerts)
	}
	return c
}

// DisableVerify makes the HTTP client not verify the server's TLS certificate.
func (c *Client) DisableVerify() *Client {
	if t, ok := c.Transport.(*http.Transport); ok && t != nil {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		t.TLSClientConfig.InsecureSkipVerify = true
	}
	return c
}

// SetCookies sets cookies to cookie jar for the given URL.
// If the cookie jar is nil, it will panic.
func (c *Client) SetCookies(url string, cookies ...*http.Cookie) *Client {
	if c.Jar == nil {
		panic(ErrNilCookieJar)
	}

	if u, err := neturl.Parse(url); err == nil {
		c.Jar.SetCookies(u, cookies)
	}
	return c
}

// OnBeforeRequest appends request hooks into the before request chain.
func (c *Client) OnBeforeRequest(hooks ...BeforeRequestHook) *Client {
	c.beforeRequestHooks = append(c.beforeRequestHooks, hooks...)
	return c
}

// OnAfterResponse appends response hooks into the after response chain.
func (c *Client) OnAfterResponse(hooks ...AfterResponseHook) *Client {
	c.afterResponseHooks = append(c.afterResponseHooks, hooks...)
	return c
}

// Get makes a GET HTTP request.
func (c *Client) Get(url string, opts ...RequestOption) *Response {
	return c.Send(MethodGet, url, opts...)
}

// Head makes a HEAD HTTP request.
func (c *Client) Head(url string, opts ...RequestOption) *Response {
	return c.Send(MethodHead, url, opts...)
}

// Post makes a POST HTTP request.
func (c *Client) Post(url string, opts ...RequestOption) *Response {
	return c.Send(MethodPost, url, opts...)
}

// Put makes a PUT HTTP request.
func (c *Client) Put(url string, opts ...RequestOption) *Response {
	return GlobalClient.Send(MethodPut, url, opts...)
}

// Patch makes a PATCH HTTP request.
func (c *Client) Patch(url string, opts ...RequestOption) *Response {
	return c.Send(MethodPatch, url, opts...)
}

// Delete makes a DELETE HTTP request.
func (c *Client) Delete(url string, opts ...RequestOption) *Response {
	return c.Send(MethodDelete, url, opts...)
}

// Send makes an HTTP request using a specified method.
func (c *Client) Send(method string, url string, opts ...RequestOption) *Response {
	req, err := NewRequest(method, url, opts...)
	if err != nil {
		return &Response{err: err}
	}

	return c.Do(req)
}

// FilterCookies returns the cookies to send in a request for the given URL from cookie jar.
func (c *Client) FilterCookies(url string) ([]*http.Cookie, error) {
	if c.Jar == nil {
		return nil, ErrNilCookieJar
	}

	u, err := neturl.Parse(url)
	if err != nil {
		return nil, err
	}

	return c.Jar.Cookies(u), nil
}

// FilterCookie returns the named cookie to send in a request for the given URL from cookie jar.
func (c *Client) FilterCookie(url string, name string) (*http.Cookie, error) {
	cookies, err := c.FilterCookies(url)
	if err != nil {
		return nil, err
	}

	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie, nil
		}
	}

	return nil, ErrNoCookie
}

// Do sends a request and returns its  response.
func (c *Client) Do(req *Request) *Response {
	resp := new(Response)

	if err := c.onBeforeRequest(req); err != nil {
		resp.err = err
		return resp
	}

	c.doWithRetry(req, resp)
	c.onAfterResponse(resp)
	return resp
}

func (c *Client) onBeforeRequest(req *Request) error {
	var err error
	for _, hook := range c.beforeRequestHooks {
		if err = hook(req); err != nil {
			break
		}
	}
	return err
}

func (c *Client) doWithRetry(req *Request, resp *Response) {
	var err error
	if req.retrier == nil {
		req.retrier = noRetry
	} else if req.retrier.maxAttempts > 1 && req.Body != nil && req.GetBody == nil {
		var body *bytes.Buffer
		body, err = drainBody(req.Body)
		if err != nil {
			resp.err = err
			return
		}
		req.SetBody(body)
	}

	ctx := req.Request.Context()
	if c.limiter != nil && !c.limiter.Allow(req.Request) {
		if err = c.limiter.Wait(ctx); err != nil {
			resp.err = err
			return
		}
	}

	for i := 0; i < req.retrier.maxAttempts; i++ {
		resp.Response, resp.err = c.do(req.Request)
		if ctx.Err() != nil || i >= req.retrier.maxAttempts-1 || !req.retrier.on(resp) {
			return
		}

		if req.GetBody != nil {
			req.Body, _ = req.GetBody()
		}

		select {
		case <-time.After(req.retrier.backoff.WaitTime(i, resp)):
		case <-ctx.Done():
			resp.err = ctx.Err()
			return
		}
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.Client.Do(req)
	if err != nil {
		return resp, err
	}

	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") &&
		resp.ContentLength != 0 {
		if _, ok := resp.Body.(*gzip.Reader); !ok {
			body, err := gzip.NewReader(resp.Body)
			resp.Body.Close()
			resp.Body = body
			return resp, err
		}
	}

	return resp, nil
}

func (c *Client) onAfterResponse(resp *Response) {
	if resp.err != nil {
		return
	}

	var err error
	for _, hook := range c.afterResponseHooks {
		if err = hook(resp); err != nil {
			resp.err = err
			return
		}
	}
}

// Get makes a GET HTTP request.
func Get(url string, opts ...RequestOption) *Response {
	return GlobalClient.Get(url, opts...)
}

// Head makes a HEAD HTTP request.
func Head(url string, opts ...RequestOption) *Response {
	return GlobalClient.Head(url, opts...)
}

// Post makes a POST HTTP request.
func Post(url string, opts ...RequestOption) *Response {
	return GlobalClient.Post(url, opts...)
}

// Put makes a PUT HTTP request.
func Put(url string, opts ...RequestOption) *Response {
	return GlobalClient.Put(url, opts...)
}

// Patch makes a PATCH HTTP request.
func Patch(url string, opts ...RequestOption) *Response {
	return GlobalClient.Patch(url, opts...)
}

// Delete makes a DELETE HTTP request.
func Delete(url string, opts ...RequestOption) *Response {
	return GlobalClient.Delete(url, opts...)
}

// Send makes an HTTP request using a specified method.
func Send(method string, url string, opts ...RequestOption) *Response {
	return GlobalClient.Send(method, url, opts...)
}
