package usrv

import "errors"

var (
	ErrEndpointAlreadyBound = errors.New("Endpoint already bound")
	ErrNoEndpointsBound     = errors.New("No endpoints bound")
	ErrServiceUnavailable   = errors.New("Service unavailable")
)
