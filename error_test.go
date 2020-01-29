package ghttp

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	_, err := NewRequest(MethodPost, "https://httpbin.org/post",
		WithJSON(map[string]interface{}{
			"num": math.Inf(1),
		}, true),
	)
	e, ok := err.(*Error)
	if assert.True(t, ok) {
		assert.Contains(t, e.Error(), "ghttp [Request.SetJSON]")
		_, ok := e.Unwrap().(*json.UnsupportedValueError)
		assert.True(t, ok)
	}
}
