package ghttp

import (
	"context"
	"net/http"
	"regexp"

	"golang.org/x/time/rate"
)

type (
	// Limiter is the interface to define a rate-limiter for limiting outbound requests.
	Limiter interface {
		// Allow determines whether an outbound request should be limited or not.
		Allow(req *http.Request) bool

		// Wait blocks until the limiter permits one event to happen.
		// It must be concurrent-safe.
		Wait(ctx context.Context) error
	}

	regexpLimiter struct {
		rateLimiter *rate.Limiter
		urlPatterns []*regexp.Regexp
	}
)

// NewRegexpLimiter returns a new Limiter given a *rate.Limiter and a group of regular expressions(optional).
// When one or more urlPatterns are specified, only the request URL matches one of the patterns will be limited.
func NewRegexpLimiter(rateLimiter *rate.Limiter, urlPatterns ...*regexp.Regexp) Limiter {
	return &regexpLimiter{
		rateLimiter: rateLimiter,
		urlPatterns: urlPatterns,
	}
}

// Allow implements Limiter interface.
func (rl *regexpLimiter) Allow(req *http.Request) bool {
	if len(rl.urlPatterns) == 0 {
		return false
	}

	for _, pattern := range rl.urlPatterns {
		if pattern.MatchString(req.URL.String()) {
			return false
		}
	}

	return true
}

// Wait implements Limiter interface.
func (rl *regexpLimiter) Wait(ctx context.Context) error {
	return rl.rateLimiter.Wait(ctx)
}
