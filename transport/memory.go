package transport

import (
	"fmt"
	"time"

	"github.com/achilleasa/usrv"
	"github.com/pborman/uuid"
)

// The internal message type used by the http transport.
type memMessage struct {
	from          string
	to            string
	correlationId string
	property      usrv.Property
	content       []byte
	err           error

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
func (m *memMessage) CorrelationId() string {
	return m.correlationId
}
func (m *memMessage) Content() ([]byte, error) {
	return m.content, m.err
}
func (m *memMessage) SetContent(content []byte, err error) {
	m.content, m.err = content, err
}

type InMemTransport struct {
	logger   usrv.Logger
	msgChans map[string]chan usrv.Message
}

func NewInMemory() *InMemTransport {
	return &InMemTransport{
		logger:   usrv.NullLogger,
		msgChans: make(map[string]chan usrv.Message, 0),
	}
}

func (t *InMemTransport) SetLogger(logger usrv.Logger) {
	t.logger = logger
}

func (t *InMemTransport) Config(params map[string]string) error {
	return nil
}

func (t *InMemTransport) Close() error {
	return nil
}
func (t *InMemTransport) Bind(service string, endpoint string) (<-chan usrv.Message, error) {
	fullPath := fmt.Sprintf("%s.%s", service, endpoint)
	t.msgChans[fullPath] = make(chan usrv.Message, 0)
	return t.msgChans[fullPath], nil
}

func (t *InMemTransport) Send(m usrv.Message, timeout time.Duration, expectReply bool) <-chan usrv.Message {
	msg, ok := m.(*memMessage)
	if !ok {
		panic("Unsupported message type")
	}

	if msg.isReply {
		msg.replyChan <- msg
		close(msg.replyChan)
		return nil
	}

	resChan := make(chan usrv.Message, 0)
	go func() {
		// Simulate async request
		reqMsg := &memMessage{
			from:          msg.from,
			to:            msg.to,
			property:      make(usrv.Property, 0),
			correlationId: msg.correlationId,
			content:       msg.content,
			// Reply Channel
			replyChan: make(chan usrv.Message, 0),
		}
		for k, v := range msg.property {
			reqMsg.property[k] = v
		}

		var resMsg usrv.Message

		// Try to match endpoint
		msgChan, found := t.msgChans[msg.to]
		if !found {
			t.logger.Error(
				"Unknown destination",
				"from", msg.from,
				"to", msg.to,
			)
			resMsg = t.ReplyTo(reqMsg)
			resMsg.SetContent(nil, usrv.ErrServiceUnavailable)
		} else {

			var timeoutChan <-chan time.Time
			if timeout > 0 {
				timeoutChan = time.After(timeout)
			}

			// Send to the bound endpoint listener and wait for reply
			msgChan <- reqMsg

			select {
			case resMsg = <-reqMsg.replyChan:
			case <-timeoutChan:
				resMsg = t.ReplyTo(reqMsg)
				resMsg.SetContent(nil, usrv.ErrTimeout)
			}
		}

		// Just pipe the res msg to the res chan
		resChan <- resMsg
		close(resChan)
	}()

	return resChan
}

// Create a message to be delivered to a target endpoint
func (t *InMemTransport) MessageTo(from string, toService string, toEndpoint string) usrv.Message {
	return &memMessage{
		from:          from,
		to:            fmt.Sprintf("%s.%s", toService, toEndpoint),
		property:      make(usrv.Property, 0),
		correlationId: uuid.New(),
	}
}

func (t *InMemTransport) ReplyTo(msg usrv.Message) usrv.Message {
	reqMsg, ok := msg.(*memMessage)
	if !ok {
		panic("Unsupported message type")
	}

	return &memMessage{
		from:          reqMsg.To(),
		to:            reqMsg.From(),
		property:      make(usrv.Property, 0),
		correlationId: reqMsg.CorrelationId(),
		// Copy reply channel from req msg
		replyChan: reqMsg.replyChan,
		isReply:   true,
	}
}
