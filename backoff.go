package ghttp

import (
	"math"
	"math/rand"
	"time"
)

type (
	// Backoff specifies the backoff of the retry policy. It is called
	// after a failing request to determine the amount of time
	// that should pass before trying again.
	Backoff interface {
		// WaitTime returns the wait time to sleep before retrying request.
		WaitTime(attemptNum int, resp *Response) time.Duration
	}

	constantBackoff struct {
		initialDuration time.Duration
		jitter          bool
	}

	exponentialBackoff struct {
		initialDuration time.Duration
		maxDuration     time.Duration
		jitter          bool
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewConstantBackoff provides a callback for the retry policy which
// will perform constant backoff with jitter based on initial duration.
func NewConstantBackoff(initialDuration time.Duration, jitter bool) Backoff {
	return &constantBackoff{
		initialDuration: initialDuration,
		jitter:          jitter,
	}
}

// WaitTime implements Backoff interface.
func (cb *constantBackoff) WaitTime(_ int, _ *Response) time.Duration {
	if !cb.jitter {
		return cb.initialDuration
	}

	return cb.initialDuration/2 + time.Duration(rand.Int63n(int64(cb.initialDuration)))
}

// NewExponentialBackoff provides a callback for the retry policy which
// will perform exponential backoff with jitter based on the attempt number and limited
// by the provided initial and maximum durations.
// See: https://aws.amazon.com/cn/blogs/architecture/exponential-backoff-and-jitter/
func NewExponentialBackoff(initialDuration, maxDuration time.Duration, jitter bool) Backoff {
	return &exponentialBackoff{
		initialDuration: initialDuration,
		maxDuration:     maxDuration,
		jitter:          jitter,
	}
}

// WaitTime implements Backoff interface.
func (eb *exponentialBackoff) WaitTime(attemptNum int, _ *Response) time.Duration {
	temp := math.Min(float64(eb.maxDuration), float64(eb.initialDuration)*math.Exp2(float64(attemptNum)))
	if !eb.jitter {
		return time.Duration(temp)
	}

	n := int64(temp / 2)
	return time.Duration(n + rand.Int63n(n))
}
