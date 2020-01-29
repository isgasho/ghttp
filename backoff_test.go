package ghttp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConstantBackoff_WaitTime(t *testing.T) {
	const (
		initialWaitTime = 100 * time.Millisecond
	)

	backoff := NewConstantBackoff(initialWaitTime, false)
	for i := 0; i < 10; i++ {
		assert.Equal(t, initialWaitTime, backoff.WaitTime(i, nil))
	}

	backoff = NewConstantBackoff(initialWaitTime, true)
	for i := 0; i < 10; i++ {
		assert.GreaterOrEqual(t, int64(backoff.WaitTime(i, nil)), int64(initialWaitTime/2))
		assert.LessOrEqual(t, int64(backoff.WaitTime(i, nil)), int64(initialWaitTime/2+initialWaitTime))
	}
}

func TestExponentialBackoff_WaitTime(t *testing.T) {
	const (
		initialWaitTime = 1 * time.Second
		maxWaitTime     = 30 * time.Second
	)

	backoff := NewExponentialBackoff(initialWaitTime, maxWaitTime, false)
	for i := 0; i < 10; i++ {
		assert.GreaterOrEqual(t, int64(backoff.WaitTime(i, nil)), int64(initialWaitTime/2))
		assert.LessOrEqual(t, int64(backoff.WaitTime(i, nil)), int64(maxWaitTime))
	}

	backoff = NewExponentialBackoff(initialWaitTime, maxWaitTime, true)
	for i := 0; i < 10; i++ {
		assert.GreaterOrEqual(t, int64(backoff.WaitTime(i, nil)), int64(initialWaitTime/2))
		assert.LessOrEqual(t, int64(backoff.WaitTime(i, nil)), int64(maxWaitTime))
	}
}
