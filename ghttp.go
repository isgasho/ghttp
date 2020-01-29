package ghttp

import (
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type (
	// Values maps a string key to an interface{} type value,
	// It's typically used for request query parameters, form data or headers,
	// besides string and []string, its value also supports []byte, bool and number,
	// ghttp will convert to string automatically.
	Values map[string]interface{}

	// Params is an alias of Values, used for for request query parameters.
	Params = Values

	// Form is an alias of Values, used for request form data.
	Form = Values

	// Headers is an alias of Values, used for request headers.
	Headers = Values

	// Cookies is a shortcut for map[string]string, used for request cookies.
	Cookies map[string]string

	// Files maps a string key to a *File type value, used for files of multipart payload.
	Files map[string]*File

	// File specifies a file to upload.
	// Note: To upload a file its Filename field must be specified, otherwise ghttp will use "file" as default.
	// If you don't specify the MIME field, ghttp will detect automatically using http.DetectContentType.
	File struct {
		Body     io.ReadCloser
		Filename string
		MIME     string
	}
)

// Set sets the key to value. It replaces any existing values.
func (v Values) Set(key string, value interface{}) Values {
	v[key] = value
	return v
}

func translate(v interface{}) (vs []string) {
	defer func() {
		if err := recover(); err != nil {
			log.Print(err)
		}
	}()

	switch v := v.(type) {
	case []string:
		vs = make([]string, len(v))
		copy(vs, v)
	default:
		vs = []string{toString(v)}
	}

	return
}

// Decode translates v and returns the equivalent request query parameters, form data or headers.
// It ignores any unexpected key-value pairs.
func (v Values) Decode() map[string][]string {
	vv := make(map[string][]string, len(v))
	for key, value := range v {
		if vs := translate(value); len(vs) > 0 {
			vv[key] = vs
		}
	}
	return vv
}

// URLEncode encodes v into URL form sorted by key if v is considered as request query parameters or form data.
// It ignores any unexpected key-value pairs.
func (v Values) URLEncode(escaped bool) string {
	vv := v.Decode()
	keys := make([]string, 0, len(vv))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		vs := vv[k]
		for _, v := range vs {
			if sb.Len() > 0 {
				sb.WriteByte('&')
			}

			if escaped {
				k = neturl.QueryEscape(k)
				v = neturl.QueryEscape(v)
			}

			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(v)
		}
	}
	return sb.String()
}

// Marshal returns the JSON encoding of v.
func (v Values) Marshal() string {
	return toJSON(v, "", "", false)
}

// Set sets the key to value. It replaces any existing values.
func (c Cookies) Set(key string, value string) Cookies {
	c[key] = value
	return c
}

// Decode translates c and returns the equivalent request cookies.
func (c Cookies) Decode() []*http.Cookie {
	cookies := make([]*http.Cookie, 0, len(c))
	for k, v := range c {
		cookies = append(cookies, &http.Cookie{
			Name:  k,
			Value: v,
		})
	}
	return cookies
}

// FileFromReader constructors a new *File from a reader.
func FileFromReader(body io.Reader) *File {
	return &File{
		Body: toReadCloser(body),
	}
}

// SetFilename specifies the filename of f.
func (f *File) SetFilename(filename string) *File {
	f.Filename = filename
	return f
}

// SetMIME specifies the mime of f.
func (f *File) SetMIME(mime string) *File {
	f.MIME = mime
	return f
}

// Read implements Reader interface.
func (f *File) Read(b []byte) (int, error) {
	return f.Body.Read(b)
}

// Close implements Closer interface.
func (f *File) Close() error {
	return f.Body.Close()
}

// Open opens the named file and returns a *File with filename specified.
func Open(filename string) (*File, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return FileFromReader(file).SetFilename(filepath.Base(filename)), nil
}

// MustOpen opens the named file and returns a *File with filename specified.
// If there is an error, it will panic.
func MustOpen(filename string) *File {
	file, err := Open(filename)
	if err != nil {
		panic(err)
	}

	return file
}
