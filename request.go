package ghttp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"strings"
)

// Common HTTP methods.
const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
)

type (
	// Request wraps the raw HTTP request.
	Request struct {
		*http.Request
		retrier *Retrier
	}

	// RequestOption provides a convenient way to setup Request.
	RequestOption func(req *Request) error

	// BeforeRequestHook is an alias of RequestOption, but for specifying a before request hook.
	// Return a non-nil error to prevent the request.
	BeforeRequestHook = RequestOption
)

// NewRequest returns a new Request given a method, URL and optional request options.
func NewRequest(method string, url string, opts ...RequestOption) (*Request, error) {
	rawRequest, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, &Error{
			Op:  "http.NewRequest",
			Err: err,
		}
	}

	req := &Request{
		Request: rawRequest,
	}
	for _, opt := range opts {
		if err = opt(req); err != nil {
			break
		}
	}
	return req, err
}

// SetBody sets body for the HTTP request.
func (req *Request) SetBody(body io.Reader) *Request {
	req.Body = toReadCloser(body)
	if body != nil {
		switch v := body.(type) {
		case *bytes.Buffer:
			req.ContentLength = int64(v.Len())
			buf := v.Bytes()
			req.GetBody = func() (io.ReadCloser, error) {
				r := bytes.NewReader(buf)
				return ioutil.NopCloser(r), nil
			}
		case *bytes.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return ioutil.NopCloser(&r), nil
			}
		case *strings.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return ioutil.NopCloser(&r), nil
			}
		default:
			// This is where we'd set it to -1 (at least
			// if body != NoBody) to mean unknown, but
			// that broke people during the Go 1.8 testing
			// period. People depend on it being 0 I
			// guess. Maybe retry later. See Issue 18117.
		}
		// For client requests, Request.ContentLength of 0
		// means either actually 0, or unknown. The only way
		// to explicitly say that the ContentLength is zero is
		// to set the Body to nil. But turns out too much code
		// depends on NewRequest returning a non-nil Body,
		// so we use a well-known ReadCloser variable instead
		// and have the http package also treat that sentinel
		// variable to mean explicitly zero.
		if req.GetBody != nil && req.ContentLength == 0 {
			req.Body = http.NoBody
			req.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
		}
	}

	return req
}

// SetHost sets Host header value for the HTTP request.
func (req *Request) SetHost(host string) *Request {
	req.Host = host
	return req
}

// SetHeaders sets headers for the HTTP request.
// It replaces any existing values.
func (req *Request) SetHeaders(headers Headers) *Request {
	for k, vs := range headers.Decode() {
		k = http.CanonicalHeaderKey(k)
		if k == "Host" && len(vs) > 0 {
			req.SetHost(vs[0])
		} else {
			req.Header[k] = vs
		}
	}
	return req
}

// SetContentType sets Content-Type header value for the HTTP request.
func (req *Request) SetContentType(contentType string) *Request {
	req.Header.Set("Content-Type", contentType)
	return req
}

// SetUserAgent sets User-Agent header value for the HTTP request.
func (req *Request) SetUserAgent(userAgent string) *Request {
	req.Header.Set("User-Agent", userAgent)
	return req
}

// SetOrigin sets Origin header value for the HTTP request.
func (req *Request) SetOrigin(origin string) *Request {
	req.Header.Set("Origin", origin)
	return req
}

// SetReferer sets Referer header value for the HTTP request.
func (req *Request) SetReferer(referer string) *Request {
	req.Header.Set("Referer", referer)
	return req
}

// SetQuery sets query parameters for the HTTP request.
// It replaces any existing values.
func (req *Request) SetQuery(params Params) *Request {
	query := req.URL.Query()
	for k, vs := range params.Decode() {
		query[k] = vs
	}
	req.URL.RawQuery = query.Encode()
	return req
}

// SetContent sets bytes payload for the HTTP request.
func (req *Request) SetContent(content []byte) *Request {
	req.SetBody(bytes.NewBuffer(content))
	return req
}

// SetText sets plain text payload for the HTTP request.
func (req *Request) SetText(text string) *Request {
	req.SetContentType("text/plain; charset=utf-8")
	req.SetBody(bytes.NewBufferString(text))
	return req
}

// SetForm sets form payload for the HTTP request.
func (req *Request) SetForm(form Form) *Request {
	req.SetContentType("application/x-www-form-urlencoded")
	req.SetBody(strings.NewReader(form.URLEncode(true)))
	return req
}

// SetJSON sets JSON payload for the HTTP request.
func (req *Request) SetJSON(data interface{}, escapeHTML bool) error {
	b, err := jsonMarshal(data, "", "", escapeHTML)
	if err != nil {
		return &Error{
			Op:  "Request.SetJSON",
			Err: err,
		}
	}

	req.SetContentType("application/json")
	req.SetBody(bytes.NewReader(b))
	return nil
}

// SetXML sets XML payload for the HTTP request.
func (req *Request) SetXML(data interface{}) error {
	b, err := xml.Marshal(data)
	if err != nil {
		return &Error{
			Op:  "Request.SetXML",
			Err: err,
		}
	}

	req.SetContentType("application/xml")
	req.SetBody(bytes.NewReader(b))
	return nil
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func setMultipartFiles(mw *multipart.Writer, files Files) error {
	const (
		fileFormat      = `form-data; name="%s"; filename="%s"`
		defaultFilename = "file"
	)

	var (
		part io.Writer
		err  error
	)
	for k, v := range files {
		filename := v.Filename
		if filename == "" {
			filename = defaultFilename
		}

		r := bufio.NewReader(v)
		mime := v.MIME
		if mime == "" {
			data, _ := r.Peek(512)
			mime = http.DetectContentType(data)
		}

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition",
			fmt.Sprintf(fileFormat, escapeQuotes(k), escapeQuotes(filename)))
		h.Set("Content-Type", mime)
		part, err = mw.CreatePart(h)
		if err != nil {
			return fmt.Errorf("can't create multipart section for form (%s=@%s): %s", k, filename, err.Error())
		}

		_, err = io.Copy(part, r)
		if err != nil {
			return fmt.Errorf("can't read file of form (%s=@%s): %s", k, filename, err.Error())
		}

		v.Close()
	}

	return nil
}

func setMultipartForm(mw *multipart.Writer, form Form) {
	for k, vs := range form.Decode() {
		for _, v := range vs {
			mw.WriteField(k, v)
		}
	}
}

// SetMultipart sets multipart payload for the HTTP request.
func (req *Request) SetMultipart(files Files, form Form) *Request {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		defer mw.Close()

		err := setMultipartFiles(mw, files)
		if err != nil {
			log.Printf("ghttp [Request.SetMultipart]: %s", err.Error())
			return
		}

		if len(form) > 0 {
			setMultipartForm(mw, form)
		}
	}()

	req.SetContentType(mw.FormDataContentType())
	req.SetBody(pr)
	return req
}

// SetCookies sets cookies for the HTTP request.
func (req *Request) SetCookies(cookies Cookies) *Request {
	for _, c := range cookies.Decode() {
		req.AddCookie(c)
	}
	return req
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// SetBasicAuth sets basic authentication for the HTTP request.
func (req *Request) SetBasicAuth(username string, password string) *Request {
	req.Header.Set("Authorization", "Basic "+basicAuth(username, password))
	return req
}

// SetBearerToken sets bearer token for the HTTP request.
func (req *Request) SetBearerToken(token string) *Request {
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// SetContext sets context for the HTTP request.
func (req *Request) SetContext(ctx context.Context) *Request {
	req.Request = req.Request.WithContext(ctx)
	return req
}

// SetRetry specifies the retrier for handling retries.
func (req *Request) SetRetry(retrier *Retrier) *Request {
	req.retrier = retrier
	return req
}

// Dump returns the HTTP/1.x wire representation of req.
func (req *Request) Dump(withBody bool) ([]byte, error) {
	return httputil.DumpRequestOut(req.Request, withBody)
}

// Export converts req to CURL command line.
func (req *Request) Export() (string, error) {
	return GenCURLCommand(req.Request)
}

// WithBody is a request option to set body for the HTTP request.
func WithBody(body io.Reader) RequestOption {
	return func(req *Request) error {
		req.SetBody(body)
		return nil
	}
}

// WithHost is a request option to set Host header value for the HTTP request.
func WithHost(host string) RequestOption {
	return func(req *Request) error {
		req.SetHost(host)
		return nil
	}
}

// WithHeaders is a request option to set headers for the HTTP request.
// It replaces any existing values.
func WithHeaders(headers Headers) RequestOption {
	return func(req *Request) error {
		req.SetHeaders(headers)
		return nil
	}
}

// WithContentType is a request option to set Content-Type header value for the HTTP request.
func WithContentType(contentType string) RequestOption {
	return func(req *Request) error {
		req.SetContentType(contentType)
		return nil
	}
}

// WithUserAgent is a request option to set User-Agent header value for the HTTP request.
func WithUserAgent(userAgent string) RequestOption {
	return func(req *Request) error {
		req.SetUserAgent(userAgent)
		return nil
	}
}

// WithOrigin is a request option to set Origin header value for the HTTP request.
func WithOrigin(origin string) RequestOption {
	return func(req *Request) error {
		req.SetOrigin(origin)
		return nil
	}
}

// WithReferer is a request option to set Referer header value for the HTTP request.
func WithReferer(referer string) RequestOption {
	return func(req *Request) error {
		req.SetReferer(referer)
		return nil
	}
}

// WithQuery is a request option to set query parameters for the HTTP request.
// It replaces any existing values.
func WithQuery(params Params) RequestOption {
	return func(req *Request) error {
		req.SetQuery(params)
		return nil
	}
}

// WithContent is a request option to set bytes payload for the HTTP request.
func WithContent(content []byte) RequestOption {
	return func(req *Request) error {
		req.SetContent(content)
		return nil
	}
}

// WithText is a request option to set plain text payload for the HTTP request.
func WithText(text string) RequestOption {
	return func(req *Request) error {
		req.SetText(text)
		return nil
	}
}

// WithForm is a request option to set form payload for the HTTP request.
func WithForm(form Form) RequestOption {
	return func(req *Request) error {
		req.SetForm(form)
		return nil
	}
}

// WithJSON is a request option to set JSON payload for the HTTP request.
func WithJSON(data interface{}, escapeHTML bool) RequestOption {
	return func(req *Request) error {
		return req.SetJSON(data, escapeHTML)
	}
}

// WithXML is a request option to set XML payload for the HTTP request.
func WithXML(data interface{}) RequestOption {
	return func(req *Request) error {
		return req.SetXML(data)
	}
}

// WithMultipart is a request option sets multipart payload for the HTTP request.
func WithMultipart(files Files, form Form) RequestOption {
	return func(req *Request) error {
		req.SetMultipart(files, form)
		return nil
	}
}

// WithCookies is a request option to set cookies for the HTTP request.
func WithCookies(cookies Cookies) RequestOption {
	return func(req *Request) error {
		req.SetCookies(cookies)
		return nil
	}
}

// WithBasicAuth is a request option to set basic authentication for the HTTP request.
func WithBasicAuth(username string, password string) RequestOption {
	return func(req *Request) error {
		req.SetBasicAuth(username, password)
		return nil
	}
}

// WithBearerToken is a request option to set bearer token for the HTTP request.
func WithBearerToken(token string) RequestOption {
	return func(req *Request) error {
		req.SetBearerToken(token)
		return nil
	}
}

// WithContext is a request option to set context for the HTTP request.
func WithContext(ctx context.Context) RequestOption {
	return func(req *Request) error {
		req.SetContext(ctx)
		return nil
	}
}

// WithRetry is a request option to specify the retrier for handling retries.
func WithRetry(retrier *Retrier) RequestOption {
	return func(req *Request) error {
		req.SetRetry(retrier)
		return nil
	}
}
