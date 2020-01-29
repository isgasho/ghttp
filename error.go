package ghttp

import (
	"errors"
	"fmt"
)

var (
	// ErrNilCookieJar can be used when the cookie jar is nil.
	ErrNilCookieJar = errors.New("ghttp: nil cookie jar")

	// ErrNoCookie can be used when a cookie not found in the HTTP response or cookie jar.
	ErrNoCookie = errors.New("ghttp: named cookie not present")
)

type (
	// Error records an error with more details to makes it more readable.
	Error struct {
		Op  string
		Err error
	}
)

// Error implements error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("ghttp [%s]: %s", e.Op, e.Err.Error())
}

// Unwrap unpacks and returns the wrapped err of e.
func (e *Error) Unwrap() error {
	return e.Err
}
