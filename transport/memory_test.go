package transport

import (
	"bytes"
	"fmt"
	"time"

	"github.com/achilleasa/usrv"
)
import "testing"

func TestMemoryTransport(t *testing.T) {
	tr := NewInMemory()
	defer tr.Close()

	reqChan, err := tr.Bind("srv", "ep1")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		select {
		case reqMsg := <-reqChan:
			propVal := reqMsg.Property().Get("foo")
			if propVal != "bar" {
				t.Fatalf("Expected property 'foo' to have value 'bar'; got %s", propVal)
			}

			resMsg := tr.ReplyTo(reqMsg)
			resMsg.SetContent([]byte("OK"), nil)
			tr.Send(resMsg, 0, false)
		}
	}()

	reqMsg := tr.MessageTo("test", "srv", "ep1")
	reqMsg.Property().Set("foo", "bar")
	resChan := tr.Send(reqMsg, 0, true)

	resMsg := <-resChan
	content, err := resMsg.Content()
	if err != nil {
		t.Fatal(err)
	}
	exp := "OK"
	if !bytes.Equal([]byte(exp), content) {
		t.Fatalf("Expected response to be %s; got %s\n", exp, string(content))
	}
	if reqMsg.CorrelationId() != resMsg.CorrelationId() {
		t.Fatalf("Expected res msg corellation id to be %s; got %s", reqMsg.CorrelationId(), resMsg.CorrelationId())
	}
}

func TestMemoryTransportTimeouts(t *testing.T) {
	tr := NewInMemory()
	defer tr.Close()

	reqChan, err := tr.Bind("srv", "ep1")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		select {
		case <-reqChan:
		}
	}()

	reqMsg := tr.MessageTo("test", "srv", "ep1")
	resChan := tr.Send(reqMsg, 1*time.Millisecond, true)

	resMsg := <-resChan
	content, err := resMsg.Content()
	if err != usrv.ErrServiceUnavailable {
		t.Fatalf("Expected to get ErrServiceUnavailable; got %v", err)
	}
	if content != nil {
		t.Fatalf("Expected content to be nil; got %v", content)
	}
}

func TestMemoryTransportError(t *testing.T) {
	tr := NewInMemory()
	tr.SetLogger(usrv.NullLogger)
	defer tr.Close()

	reqChan, err := tr.Bind("srv", "ep1")
	if err != nil {
		t.Fatal(err)
	}

	expErr := fmt.Errorf("An error")

	go func() {
		select {
		case msg := <-reqChan:
			res := tr.ReplyTo(msg)
			res.SetContent(nil, expErr)
			tr.Send(res, 0, false)
		}
	}()

	reqMsg := tr.MessageTo("test", "srv", "ep1")
	resChan := tr.Send(reqMsg, 0, true)

	resMsg := <-resChan
	content, err := resMsg.Content()
	if err.Error() != expErr.Error() {
		t.Fatalf("Expected to get error %v; got %v", expErr, err)
	}
	if content != nil {
		t.Fatalf("Expected content to be nil; got %v", content)
	}
}

func TestMemoryTransportUnknownEndpoint(t *testing.T) {
	tr := NewInMemory()
	defer tr.Close()

	reqMsg := tr.MessageTo("test", "srv", "ep1")
	resChan := tr.Send(reqMsg, 0, true)

	resMsg := <-resChan
	content, err := resMsg.Content()
	if err != usrv.ErrServiceUnavailable {
		t.Fatalf("Expected to get ErrServiceUnavailable; got %v", err)
	}
	if content != nil {
		t.Fatalf("Expected content to be nil; got %v", content)
	}
}

func TestMemoryTransportConfig(t *testing.T) {
	tr := NewInMemory()
	defer tr.Close()

	err := tr.Config(map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatal(err)
	}
}
