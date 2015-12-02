package transport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	httpPkg "net/http"
	"strconv"
	"sync"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/achilleasa/usrv"
)

var (
	defaultDialer = &net.Dialer{Timeout: 1000 * time.Millisecond}
	httpClient    = &httpPkg.Client{
		Transport: &httpPkg.Transport{
			Dial:  defaultDialer.Dial,
			Proxy: httpPkg.ProxyFromEnvironment,
		},
	}
)

// The DefaultTransport used by the http package implements this
// method but the Transport interface does not expose it. This
// is a hack-y way to access it.
type requestCanceler interface {
	CancelRequest(*httpPkg.Request)
}

// The internal message type used by the http transport.
type httpMessage struct {
	from          string
	to            string
	property      usrv.Property
	correlationId string
	content       []byte
	err           error

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
func (m *httpMessage) CorrelationId() string {
	return m.correlationId
}
func (m *httpMessage) Content() ([]byte, error) {
	return m.content, m.err
}
func (m *httpMessage) SetContent(content []byte, err error) {
	m.content, m.err = content, err
}

type HttpConfig map[string]string

func NewHttpConfig(serverPort int) HttpConfig {
	return HttpConfig{
		"port": fmt.Sprint(serverPort),
	}
}

type HttpTransport struct {
	logger   usrv.Logger
	port     int
	msgChans map[string]chan usrv.Message

	// Are we listening for requests? Server-side only
	listening bool
	sync.Mutex
}

func NewHttp() *HttpTransport {
	return &HttpTransport{
		logger:   usrv.NullLogger,
		port:     80,
		msgChans: make(map[string]chan usrv.Message, 0),
	}
}

func (t *HttpTransport) SetLogger(logger usrv.Logger) {
	t.logger = logger
}

func (t *HttpTransport) Config(params map[string]string) error {
	needsReset := false

	portVal, exists := params["port"]
	if exists {
		port, err := strconv.Atoi(portVal)
		if err != nil {
			return err
		}
		t.port = port
		needsReset = true
	}

	if needsReset {
		t.logger.Info("Configuration changed", "port", t.port)
		return t.listen()
	}

	return nil
}

func (t *HttpTransport) Close() error {
	return nil
}
func (t *HttpTransport) Bind(service string, endpoint string) (<-chan usrv.Message, error) {
	err := t.listen()
	if err != nil {
		return nil, err
	}

	fullPath := fmt.Sprintf("%s/%s", service, endpoint)
	t.msgChans[fullPath] = make(chan usrv.Message, 0)
	return t.msgChans[fullPath], nil
}

func (t *HttpTransport) Send(m usrv.Message, timeout time.Duration, expectReply bool) <-chan usrv.Message {
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
	req.Header.Set("X-Usrv-CorrelationId", msg.correlationId)
	req.Header.Set("Referer", msg.from)

	resChan := make(chan usrv.Message, 0)
	go func() {
		resMsg := &httpMessage{
			from:          msg.to,
			to:            msg.from,
			property:      make(usrv.Property, 0),
			correlationId: msg.correlationId,
		}

		defer func() {
			resChan <- resMsg
			close(resChan)
		}()

		// If a timeout is specified, set up a timer to cancel the request
		if timeout > 0 {
			time.AfterFunc(timeout, func() {
				if canceler, ok := httpClient.Transport.(requestCanceler); ok {
					canceler.CancelRequest(req)
				}
			})
		}

		res, err := httpClient.Do(req)
		if err != nil {
			t.logger.Error(
				"Http request failed",
				"from", msg.from,
				"to", msg.to,
				"err", err.Error(),
			)
			resMsg.SetContent(nil, usrv.ErrServiceUnavailable)
			return
		}

		// Handle non-2XX codes
		if res.StatusCode < 200 || res.StatusCode > 299 {
			t.logger.Error(
				"Http request failed",
				"from", msg.from,
				"to", msg.to,
				"status", res.StatusCode,
			)
			resMsg.SetContent(nil, usrv.ErrServiceUnavailable)
			return
		}

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
func (t *HttpTransport) MessageTo(from string, toService string, toEndpoint string) usrv.Message {
	return &httpMessage{
		from:          from,
		to:            fmt.Sprintf("%s/%s", toService, toEndpoint),
		property:      make(usrv.Property, 0),
		correlationId: uuid.New(),
	}
}

func (t *HttpTransport) ReplyTo(msg usrv.Message) usrv.Message {
	reqMsg, ok := msg.(*httpMessage)
	if !ok {
		panic("Unsupported message type")
	}

	return &httpMessage{
		from:          reqMsg.To(),
		to:            reqMsg.From(),
		property:      make(usrv.Property, 0),
		correlationId: reqMsg.CorrelationId(),
		// Copy reply channel from req msg
		replyChan: reqMsg.replyChan,
		isReply:   true,
	}
}

// Handle incoming HTTP request. This method will attempt to match the host/path
// combination of incoming requests to a bound endpoint. If a match is found,
// the message will be sent to the matched endpoint's queue; otherwise a 404 will
// be returned.
func (t *HttpTransport) handleRequest(w httpPkg.ResponseWriter, r *httpPkg.Request) {
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
		from:          r.Referer(),
		to:            r.Host + r.URL.String(),
		property:      make(usrv.Property, 0),
		correlationId: r.Header.Get("X-Usrv-CorrelationId"),
		content:       content,
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
	w.Header().Set("X-Usrv-CorrelationId", reqMsg.correlationId)
	w.Write(content)
}

// Ensure that the transport is listening for incoming connections.
func (t *HttpTransport) listen() error {
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
