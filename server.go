package usrv

import (
	"sync"

	"golang.org/x/net/context"
)

// Endpoint handlers should match this signature.
type Handler func(req, res Message)

type serverEndpoint struct {
	name    string
	msgChan <-chan Message
	handler Handler
}

type Server struct {
	ctx         context.Context
	ctxCancelFn context.CancelFunc

	endpoints   []serverEndpoint
	epWaitGroup sync.WaitGroup

	transport Transport

	service string
}

func NewServer(service string, transport Transport) *Server {
	ctx, cancelFn := context.WithCancel(context.Background())
	return &Server{
		ctx:         ctx,
		ctxCancelFn: cancelFn,
		endpoints:   make([]serverEndpoint, 0),
		transport:   transport,
		service:     service,
	}
}

// Bind endpoint.
func (srv *Server) Handle(endpoint string, handler Handler) error {
	for _, existing := range srv.endpoints {
		if existing.name == endpoint {
			return ErrEndpointAlreadyBound
		}
	}

	// Try to bind
	msgChan, err := srv.transport.Bind(srv.service, endpoint)
	if err != nil {
		return err
	}

	srv.endpoints = append(srv.endpoints, serverEndpoint{
		name:    endpoint,
		msgChan: msgChan,
		handler: handler,
	})

	return nil
}

// Listen for incoming messages and dispatch them to the registered endpoints.
func (srv *Server) Listen() error {
	if len(srv.endpoints) == 0 {
		return ErrNoEndpointsBound
	}

	// Start a go-routine to handle incoming messages
	for _, endpoint := range srv.endpoints {
		srv.epWaitGroup.Add(1)
		go srv.serve(endpoint)
	}

	return nil
}

// Shut down the server.
func (srv *Server) Close() error {
	// Cancel server context and wait till all endpoint handler go-routines terminate.
	srv.ctxCancelFn()
	srv.epWaitGroup.Wait()

	return nil
}

// Serve an endpoint. This method will dequeue messages from the endpoint msg
// channel and spawn a go-routine to handle the request. The endpoint handler
// will exit if the server context is somehow terminated.
func (srv *Server) serve(endpoint serverEndpoint) {
	defer srv.epWaitGroup.Done()
	for {
		select {
		case <-srv.ctx.Done():
			return
		case msg := <-endpoint.msgChan:
			go func(req Message) {
				res := srv.transport.ReplyTo(req)
				endpoint.handler(req, res)
				srv.transport.Send(res, false)

			}(msg)
		}
	}

}
