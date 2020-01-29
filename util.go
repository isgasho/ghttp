package ghttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"sync"
	"unsafe"
)

var (
	bufPool = &sync.Pool{New: func() interface{} { return &bytes.Buffer{} }}
)

type (
	// H is a shortcut for map[string]interface{}, used for JSON unmarshalling.
	// Do not use it for other purposes!
	H map[string]interface{}

	// Number is a shortcut for float64.
	Number float64
)

func acquireBuffer() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func releaseBuffer(buf *bytes.Buffer) {
	if buf != nil {
		buf.Reset()
		bufPool.Put(buf)
	}
}

// Float64 converts n to a float64.
func (n Number) Float64() float64 {
	return float64(n)
}

// Float32 converts n to a float32.
func (n Number) Float32() float32 {
	return float32(n)
}

// Int converts n to an int.
func (n Number) Int() int {
	return int(n)
}

// Int64 converts n to an int64.
func (n Number) Int64() int64 {
	return int64(n)
}

// Int32 converts n to an int32.
func (n Number) Int32() int32 {
	return int32(n)
}

// Int16 converts n to an int16.
func (n Number) Int16() int16 {
	return int16(n)
}

// Int8 converts n to an int8.
func (n Number) Int8() int8 {
	return int8(n)
}

// Uint converts n to a uint.
func (n Number) Uint() uint {
	return uint(n)
}

// Uint64 converts n to a uint64.
func (n Number) Uint64() uint64 {
	return uint64(n)
}

// Uint32 converts n to a uint32.
func (n Number) Uint32() uint32 {
	return uint32(n)
}

// Uint16 converts n to a uint16.
func (n Number) Uint16() uint16 {
	return uint16(n)
}

// Uint8 converts n to a uint8.
func (n Number) Uint8() uint8 {
	return uint8(n)
}

// String converts n to a string.
func (n Number) String() string {
	return strconv.FormatFloat(n.Float64(), 'f', -1, 64)
}

// Has checks if a key exists.
func (h H) Has(key string) bool {
	_, ok := h[key]
	return ok
}

// Get gets the interface{} value associated with key.
func (h H) Get(key string) interface{} {
	v, _ := h[key]
	return v
}

// GetH gets the H value associated with key.
func (h H) GetH(key string) H {
	v, _ := h[key].(map[string]interface{})
	return v
}

// GetStringDefault gets the string value associated with key.
// The defaultValue is returned if the key not exists.
func (h H) GetStringDefault(key string, defaultValue string) string {
	v, ok := h[key].(string)
	if !ok {
		return defaultValue
	}

	return v
}

// GetString gets the string value associated with key.
// The zero value is returned if the key not exists.
func (h H) GetString(key string) string {
	return h.GetStringDefault(key, "")
}

// GetBoolDefault gets the bool value associated with key.
// The defaultValue is returned if the key not exists.
func (h H) GetBoolDefault(key string, defaultValue bool) bool {
	v, ok := h[key].(bool)
	if !ok {
		return defaultValue
	}

	return v
}

// GetBool gets the bool value associated with key.
// The zero value is returned if the key not exists.
func (h H) GetBool(key string) bool {
	return h.GetBoolDefault(key, false)
}

// GetNumberDefault gets the Number value associated with key.
// The defaultValue is returned if the key not exists.
func (h H) GetNumberDefault(key string, defaultValue Number) Number {
	v, ok := h[key].(float64)
	if !ok {
		return defaultValue
	}

	return Number(v)
}

// GetNumber gets the Number value associated with key.
// The zero value is returned if the key not exists.
func (h H) GetNumber(key string) Number {
	return h.GetNumberDefault(key, 0)
}

// GetSlice gets the []interface{} value associated with key.
func (h H) GetSlice(key string) []interface{} {
	v, _ := h[key].([]interface{})
	return v
}

// GetHSlice gets the []H value associated with key.
func (h H) GetHSlice(key string) []H {
	v := h.GetSlice(key)
	vs := make([]H, 0, len(v))
	for _, vv := range v {
		if vv, ok := vv.(map[string]interface{}); ok {
			vs = append(vs, vv)
		}
	}
	return vs
}

// GetStringSlice gets the []string value associated with key.
func (h H) GetStringSlice(key string) []string {
	v := h.GetSlice(key)
	vs := make([]string, 0, len(v))
	for _, vv := range v {
		if vv, ok := vv.(string); ok {
			vs = append(vs, vv)
		}
	}
	return vs
}

// GetBoolSlice gets the []bool value associated with key.
func (h H) GetBoolSlice(key string) []bool {
	v := h.GetSlice(key)
	vs := make([]bool, 0, len(v))
	for _, vv := range v {
		if vv, ok := vv.(bool); ok {
			vs = append(vs, vv)
		}
	}
	return vs
}

// GetNumberSlice gets the []Number value associated with key.
func (h H) GetNumberSlice(key string) []Number {
	v := h.GetSlice(key)
	vs := make([]Number, 0, len(v))
	for _, vv := range v {
		if vv, ok := vv.(float64); ok {
			vs = append(vs, Number(vv))
		}
	}
	return vs
}

// Decode encodes h to JSON and then decodes to the output structure.
// output must be a pointer.
func (h H) Decode(output interface{}) error {
	b, err := jsonMarshal(h, "", "", false)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, output)
}

// String returns the JSON-encoded text representation of h.
func (h H) String() string {
	return toJSON(h, "", "\t", false)
}

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func toString(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	case []byte:
		return b2s(v)
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	}

	panic(fmt.Errorf("ghttp: unexpected value %#v of type %T", v, v))
}

var jsonSuffix = []byte{'\n'}

func jsonMarshal(v interface{}, prefix string, indent string, escapeHTML bool) ([]byte, error) {
	buf := acquireBuffer()
	defer releaseBuffer(buf)

	encoder := json.NewEncoder(buf)
	encoder.SetIndent(prefix, indent)
	encoder.SetEscapeHTML(escapeHTML)
	err := encoder.Encode(v)
	return bytes.TrimSuffix(buf.Bytes(), jsonSuffix), err
}

func toJSON(v interface{}, prefix string, indent string, escapeHTML bool) string {
	b, err := jsonMarshal(v, prefix, indent, escapeHTML)
	if err != nil {
		return "{}"
	}

	return b2s(b)
}

func toReadCloser(r io.Reader) io.ReadCloser {
	rc, ok := r.(io.ReadCloser)
	if !ok && r != nil {
		rc = ioutil.NopCloser(r)
	}
	return rc
}
