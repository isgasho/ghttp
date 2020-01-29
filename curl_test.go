package ghttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenCURLCommand(t *testing.T) {
	req, err := NewRequest(MethodPost, "https://httpbin.org/post?k1=v1",
		WithQuery(Params{
			"k2": "v2",
		}),
		WithForm(Form{
			"k3": "v3",
			"k4": "v4",
		}),
		WithHeaders(Headers{
			"User-Agent": "Go-http-client",
		}),
		WithHost("google.com"),
	)
	require.NoError(t, err)

	req.TransferEncoding = []string{"chunked"}
	req.Close = true
	want := "curl -v -X 'POST' -d 'k3=v3&k4=v4' -H 'Host: google.com' -H 'Transfer-Encoding: chunked' -H 'Connection: close' -H 'Content-Type: application/x-www-form-urlencoded' -H 'User-Agent: Go-http-client' 'https://httpbin.org/post?k1=v1&k2=v2'"
	cmd, err := req.Export()
	if assert.NoError(t, err) {
		assert.Equal(t, want, cmd)
	}
}
