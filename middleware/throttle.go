package middleware

import (
	"time"

	"github.com/achilleasa/usrv"
)

// Throttle incoming requests so that only maxConcurrent requests can be
// executed in parallel.  If a non-zero timeout is specified and a pending request
// cannot be serviced within the specified timeout, it will be aborted with
// ErrTimeout.
func Throttle(maxConcurrent int, timeout time.Duration, handler usrv.Handler) usrv.Handler {

	if maxConcurrent <= 0 {
		panic("maxConcurrent should be > 0")
	}

	// Allocate a buffered channel and pre-fill it with tokens
	tokens := make(chan struct{}, maxConcurrent)
	for i := 0; i < maxConcurrent; i++ {
		tokens <- struct{}{}
	}

	return func(req, res usrv.Message) {

		var timeoutChan <-chan time.Time
		if timeout > 0 {
			timeoutChan = time.After(timeout)
		}

		select {
		case <-tokens:
			defer func() {
				tokens <- struct{}{}
			}()

			handler(req, res)
		case <-timeoutChan:
			res.SetContent(nil, usrv.ErrTimeout)
		}

	}
}
