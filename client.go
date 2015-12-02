package usrv

import "time"

type Client struct {
	service   string
	transport Transport
}

func NewClient(service string, transport Transport) *Client {
	return &Client{
		service:   service,
		transport: transport,
	}
}

func (c *Client) NewMessage(from string, toEndpoint string) Message {
	return c.transport.MessageTo(from, c.service, toEndpoint)
}

func (c *Client) Send(msg Message, timeout time.Duration) <-chan Message {
	return c.transport.Send(msg, timeout, true)
}
