package ghttp

var (
	noRetry = &Retrier{
		maxAttempts: 1,
	}
)

type (
	// Retrier specifies the retry policy for handling retries.
	Retrier struct {
		maxAttempts int
		backoff     Backoff
		triggers    []func(resp *Response) bool
	}
)

// NewRetrier returns a new retrier given the max attempts, backoff and optional triggers.
// maxAttempts specifies the max attempts of the retry policy, 1 means no retries.
// triggers determines whether a request needs a retry or not(optional).
// If the triggers not specified, default is the response's error isn't nil.
func NewRetrier(maxAttempts int, backoff Backoff, triggers ...func(resp *Response) bool) *Retrier {
	return &Retrier{
		maxAttempts: maxAttempts,
		backoff:     backoff,
		triggers:    triggers,
	}
}

func (r *Retrier) on(resp *Response) bool {
	if len(r.triggers) == 0 {
		return false
	}

	for _, trigger := range r.triggers {
		if trigger(resp) {
			return true
		}
	}

	return false
}
