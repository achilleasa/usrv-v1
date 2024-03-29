package usrv

import "time"

type Transport interface {

	// Attach a logger to the transport.
	SetLogger(logger Logger)

	// Configure the transport.
	Config(params map[string]string) error

	// Close the transport.
	Close() error

	// Bind service endpoint. Returns a channel that emits incoming Messages to that endpoint
	Bind(service string, endpoint string) (<-chan Message, error)

	// Send a message.
	Send(message Message, timeout time.Duration, expectReply bool) <-chan Message

	// Create a message to be delivered to a target endpoint
	MessageTo(from string, toService string, toEndpoint string) Message

	// Create a message that serves as a reply to an incoming message
	ReplyTo(message Message) Message
}
