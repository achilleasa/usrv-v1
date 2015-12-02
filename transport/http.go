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

	"gopkg.in/tylerb/graceful.v1"

	"code.google.com/p/go-uuid/uuid"
	"github.com/achilleasa/usrv"
)

var (
	errListenerClosed = errors.New("Listener stopped")
	defaultDialer     = &net.Dialer{Timeout: 1000 * time.Millisecond}
	httpClient        = &httpPkg.Client{
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

func NewHttpsConfig(serverPort int, certFile, certKeyFile string) HttpConfig {
	return HttpConfig{
		"port":        fmt.Sprint(serverPort),
		"certFile":    certFile,
		"certKeyFile": certKeyFile,
	}
}

type HttpTransport struct {
	logger      usrv.Logger
	port        int
	certFile    string
	certKeyFile string
	msgChans    map[string]chan usrv.Message

	// The protocol for outgoing requests (http or https if TLS is enabled)
	protocol string

	server *graceful.Server

	// A mutex for synchronized access to the server instance
	sync.Mutex
}

func NewHttp() *HttpTransport {
	t := &HttpTransport{
		logger:   usrv.NullLogger,
		port:     80,
		protocol: "http://",
		msgChans: make(map[string]chan usrv.Message, 0),
	}
	return t
}

func (t *HttpTransport) SetLogger(logger usrv.Logger) {
	t.logger = logger
}

func (t *HttpTransport) Config(params map[string]string) error {
	needsReset := false
	t.certFile = ""
	t.certKeyFile = ""
	t.protocol = "http://"

	portVal, portDefined := params["port"]
	if portDefined {
		port, err := strconv.Atoi(portVal)
		if err != nil {
			return err
		}
		t.port = port
		needsReset = true
	}

	certFile := params["certFile"]
	certKeyFile := params["certKeyFile"]
	if certFile != "" && certKeyFile != "" {
		t.certFile = certFile
		t.certKeyFile = certKeyFile

		// If port is not defined, use 443 as the default
		if !portDefined {
			t.port = 443
		}

		t.protocol = "https://"
		needsReset = true
	}

	if needsReset {
		t.logger.Info("Configuration changed", "port", t.port, "protocol", t.protocol)
		return t.listen()
	}

	return nil
}

func (t *HttpTransport) Close() error {
	t.Lock()
	defer t.Unlock()

	if t.server == nil {
		return nil
	}

	// Close server and wait for connections to drain
	stopChan := t.server.StopChan()
	t.server.Stop(1 * time.Second)
	<-stopChan
	t.server = nil

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

	req, err := httpPkg.NewRequest("POST", t.protocol+msg.to, body)
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

	// Already listening
	if t.server != nil {
		return nil
	}

	addr := fmt.Sprintf(":%d", t.port)
	t.server = &graceful.Server{
		Server: &httpPkg.Server{
			Addr:    addr,
			Handler: httpPkg.HandlerFunc(t.handleRequest),
		},
	}

	var err error
	var listener net.Listener
	if t.certFile != "" && t.certKeyFile != "" {
		listener, err = t.server.ListenTLS(t.certFile, t.certKeyFile)
	} else {
		listener, err = net.Listen("tcp", addr)
	}

	if err != nil {
		return err
	}

	go func() {
		t.server.Serve(listener)
		t.logger.Error("Http server exited", "err", err)
	}()

	return nil
}
