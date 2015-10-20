package transport

import (
	"fmt"

	"github.com/achilleasa/usrv"
)

// The internal message type used by the http transport.
type memMessage struct {
	from     string
	to       string
	property usrv.Property
	content  []byte
	err      error

	isReply   bool
	replyChan chan usrv.Message
}

func (m *memMessage) From() string {
	return m.from
}
func (m *memMessage) To() string {
	return m.to
}
func (m *memMessage) Property() usrv.Property {
	return m.property
}
func (m *memMessage) Content() ([]byte, error) {
	return m.content, m.err
}
func (m *memMessage) SetContent(content []byte, err error) {
	m.content, m.err = content, err
}

type inMemTransport struct {
	msgChans map[string]chan usrv.Message
}

func NewInMemory() *inMemTransport {
	return &inMemTransport{
		msgChans: make(map[string]chan usrv.Message, 0),
	}
}

func (t *inMemTransport) Close() error {
	return nil
}
func (t *inMemTransport) Bind(service string, endpoint string) (<-chan usrv.Message, error) {
	fullPath := fmt.Sprintf("%s.%s", service, endpoint)
	t.msgChans[fullPath] = make(chan usrv.Message, 0)
	return t.msgChans[fullPath], nil
}

func (t *inMemTransport) Send(m usrv.Message, expectReply bool) <-chan usrv.Message {
	msg, ok := m.(*memMessage)
	if !ok {
		panic("Unsupported message type")
	}

	if msg.isReply {
		msg.replyChan <- msg
		close(msg.replyChan)
		return nil
	}

	// Try to match endpoint
	msgChan, found := t.msgChans[msg.to]
	if !found {
		panic("Endpoint not found")
	}

	resChan := make(chan usrv.Message, 0)
	go func() {
		// Simulate async request
		reqMsg := &memMessage{
			from:     msg.from,
			to:       msg.to,
			property: make(usrv.Property, 0),
			content:  msg.content,
			// Reply Channel
			replyChan: make(chan usrv.Message, 0),
		}
		for k, v := range msg.property {
			reqMsg.property[k] = v
		}

		// Send to the bound endpoint listener and wait for reply
		msgChan <- reqMsg
		resMsg := <-reqMsg.replyChan

		// Just pipe the res msg to the res chan
		resChan <- resMsg
		close(resChan)
	}()

	return resChan
}

// Create a message to be delivered to a target endpoint
func (t *inMemTransport) MessageTo(from string, toService string, toEndpoint string) usrv.Message {
	return &memMessage{
		from:     from,
		to:       fmt.Sprintf("%s.%s", toService, toEndpoint),
		property: make(usrv.Property, 0),
	}
}

func (t *inMemTransport) ReplyTo(msg usrv.Message) usrv.Message {
	reqMsg, ok := msg.(*memMessage)
	if !ok {
		panic("Unsupported message type")
	}

	return &memMessage{
		from:     reqMsg.To(),
		to:       reqMsg.From(),
		property: make(usrv.Property, 0),
		// Copy reply channel from req msg
		replyChan: reqMsg.replyChan,
		isReply:   true,
	}
}
