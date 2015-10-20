package transport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	httpPkg "net/http"
	"sync"

	"github.com/achilleasa/usrv"
)

var (
	httpClient = &httpPkg.Client{}
)

// The internal message type used by the http transport.
type httpMessage struct {
	from     string
	to       string
	property usrv.Property
	content  []byte
	err      error

	isReply   bool
	replyChan chan usrv.Message
}

func (m *httpMessage) From() string {
	return m.from
}
func (m *httpMessage) To() string {
	return m.to
}
func (m *httpMessage) Property() usrv.Property {
	return m.property
}
func (m *httpMessage) Content() ([]byte, error) {
	return m.content, m.err
}
func (m *httpMessage) SetContent(content []byte, err error) {
	m.content, m.err = content, err
}

type httpTransport struct {
	port     int
	msgChans map[string]chan usrv.Message

	// Are we listening for requests? Server-side only
	listening bool
	sync.Mutex
}

func NewHttp(port int) *httpTransport {
	return &httpTransport{
		port: port,

		msgChans: make(map[string]chan usrv.Message, 0),
	}
}

func (t *httpTransport) Close() error {
	return nil
}
func (t *httpTransport) Bind(service string, endpoint string) (<-chan usrv.Message, error) {
	err := t.listen()
	if err != nil {
		return nil, err
	}

	fullPath := fmt.Sprintf("%s/%s", service, endpoint)
	t.msgChans[fullPath] = make(chan usrv.Message, 0)
	return t.msgChans[fullPath], nil
}

func (t *httpTransport) Send(m usrv.Message, expectReply bool) <-chan usrv.Message {
	msg, ok := m.(*httpMessage)
	if !ok {
		panic("Unsupported message type")
	}

	if msg.isReply {
		msg.replyChan <- msg
		close(msg.replyChan)
		return nil
	}

	var body io.Reader
	content, _ := msg.Content()
	if content != nil {
		body = bytes.NewReader(content)
	}

	req, err := httpPkg.NewRequest("POST", "http://"+msg.to, body)
	if err != nil {
		panic(err)
	}
	if len(msg.property) > 0 {
		bytes, _ := json.Marshal(msg.property)
		req.Header.Set("X-Usrv-Properties", string(bytes))
	}
	req.Header.Set("Referer", msg.from)

	resChan := make(chan usrv.Message, 0)
	go func() {
		resMsg := &httpMessage{
			from:     msg.to,
			to:       msg.from,
			property: make(usrv.Property, 0),
		}

		defer func() {
			resChan <- resMsg
			close(resChan)
		}()

		res, err := httpClient.Do(req)
		if err != nil {
			resMsg.SetContent(nil, err)
			return
		}

		fmt.Printf("Res Headers: %#v\n", res.Header)

		// Parse property header and check for errors
		propHeader := res.Header.Get("X-Usrv-Properties")
		if propHeader != "" {
			json.Unmarshal([]byte(propHeader), &resMsg.property)

			errProp := resMsg.property.Get(usrv.PropertyHasError)
			if errProp != "" {
				res.Body.Close()
				resMsg.SetContent(nil, errors.New(errProp))
				return
			}
		}

		// Parse body
		content, err := ioutil.ReadAll(res.Body)
		if err != nil {
			resMsg.SetContent(nil, err)
			return
		}

		resMsg.SetContent(content, nil)
	}()

	return resChan
}

// Create a message to be delivered to a target endpoint
func (t *httpTransport) MessageTo(from string, toService string, toEndpoint string) usrv.Message {
	return &httpMessage{
		from:     from,
		to:       fmt.Sprintf("%s/%s", toService, toEndpoint),
		property: make(usrv.Property, 0),
	}
}

func (t *httpTransport) ReplyTo(msg usrv.Message) usrv.Message {
	reqMsg, ok := msg.(*httpMessage)
	if !ok {
		panic("Unsupported message type")
	}

	return &httpMessage{
		from:     reqMsg.To(),
		to:       reqMsg.From(),
		property: make(usrv.Property, 0),
		// Copy reply channel from req msg
		replyChan: reqMsg.replyChan,
		isReply:   true,
	}
}

// Handle incoming HTTP request. This method will attempt to match the host/path
// combination of incoming requests to a bound endpoint. If a match is found,
// the message will be sent to the matched endpoint's queue; otherwise a 404 will
// be returned.
func (t *httpTransport) handleRequest(w httpPkg.ResponseWriter, r *httpPkg.Request) {
	// Try to match endpoint
	endpoint := r.Host + r.URL.String()
	msgChan, found := t.msgChans[endpoint]
	if !found {
		httpPkg.NotFound(w, r)
		return
	}

	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	reqMsg := &httpMessage{
		from:     r.Referer(),
		to:       r.Host + r.URL.String(),
		property: make(usrv.Property, 0),
		content:  content,
		// Reply Channel
		replyChan: make(chan usrv.Message, 0),
	}

	// Parse properties
	propHeader := r.Header.Get("X-Usrv-Properties")
	if propHeader != "" {
		json.Unmarshal([]byte(propHeader), &reqMsg.property)
	}

	// Send to the bound endpoint listener and wait for reply
	msgChan <- reqMsg
	resMsg := <-reqMsg.replyChan

	content, err = resMsg.Content()
	if err != nil {
		resMsg.Property().Set(usrv.PropertyHasError, err.Error())
	}
	if len(resMsg.Property()) > 0 {
		bytes, _ := json.Marshal(resMsg.Property())
		w.Header().Set("X-Usrv-Properties", string(bytes))
	}
	w.Header().Set("Referer", reqMsg.To())
	w.Write(content)
}

// Ensure that the transport is listening for incoming connections.
func (t *httpTransport) listen() error {
	t.Lock()
	defer t.Unlock()

	if t.listening {
		return nil
	}

	go func() {
		err := httpPkg.ListenAndServe(
			fmt.Sprintf(":%d", t.port),
			httpPkg.HandlerFunc(t.handleRequest),
		)

		if err != nil {
			panic(err)
		}
	}()

	t.listening = true
	return nil
}
