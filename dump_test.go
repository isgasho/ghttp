package ghttp

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	errRead = 1 << iota
	errClose
)

var (
	errPermissionDenied = errors.New("permission denied")
)

type (
	dummyBody struct {
		s       string
		i       int64
		errFlag int
	}
)

func (db *dummyBody) Read(b []byte) (n int, err error) {
	if db.errFlag&errRead != 0 {
		return 0, errPermissionDenied
	}

	if db.i >= int64(len(db.s)) {
		return 0, io.EOF
	}

	n = copy(b, db.s[db.i:])
	db.i += int64(n)
	return
}

func (db *dummyBody) Close() error {
	if db.errFlag&errClose != 0 {
		return errPermissionDenied
	}

	return nil
}

func TestRequest_Dump(t *testing.T) {
	req, err := NewRequest(MethodPost, "https://httpbin.org/post",
		WithQuery(Params{
			"k1": "v1",
			"k2": "v2",
		}),
		WithForm(Form{
			"k3": "v3",
			"k4": "v4",
		}),
	)
	require.NoError(t, err)

	_, err = req.Dump(true)
	assert.NoError(t, err)
}
