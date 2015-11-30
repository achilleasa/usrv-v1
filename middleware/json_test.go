package middleware

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/achilleasa/usrv/usrvtest"
)

func TestJsonHandler(t *testing.T) {
	type request struct {
		A string
	}
	type response struct {
		B int
	}

	handler := JsonHandler(func(req *request, res *response) error {
		if req.A != "42" {
			t.Fatalf("Expected req.A to be '42'; got %s", req.A)
		}
		res.B = 42
		return nil
	}, false)

	reqMsg := &usrvtest.Message{
		Cont: []byte(`{"A":"42"}`),
	}
	resMsg := &usrvtest.Message{}

	handler(reqMsg, resMsg)

	if !bytes.Equal(resMsg.Cont, []byte(`{"B":42}`)) {
		t.Fatalf("Response message has unexpected content: %s", string(resMsg.Cont))
	}
}

func TestJsonHandlerErrorWrapping(t *testing.T) {
	type request struct {
		A string
	}
	type response struct {
		B int
	}

	handler := JsonHandler(func(req *request, res *response) error {
		if req.A == "0" {
			return fmt.Errorf("This is not the answer you are looking for")
		} else if req.A == "0/0" {
			panic(fmt.Errorf("Divide by zero"))
		}
		panic("42 is the ultimate answer")
	}, true)

	resMsg := &usrvtest.Message{}

	// Test normal error wrapping
	reqMsg := &usrvtest.Message{
		Cont: []byte(`{"A":"0"}`),
	}
	handler(reqMsg, resMsg)

	if resMsg.Err == nil || resMsg.Err.Error() != "This is not the answer you are looking for" {
		t.Fatalf("Response message has unexpected error: %v", resMsg.Err)
	}

	// Test panic wrapping when panic is invoked with an error
	reqMsg = &usrvtest.Message{
		Cont: []byte(`{"A":"0/0"}`),
	}
	handler(reqMsg, resMsg)

	if resMsg.Err == nil || resMsg.Err.Error() != "Divide by zero" {
		t.Fatalf("Response message has unexpected error: %v", resMsg.Err)
	}

	// Test panic wrapping when panic is invoked with a non-error
	reqMsg = &usrvtest.Message{
		Cont: []byte(`{"A":"42"}`),
	}
	handler(reqMsg, resMsg)

	if resMsg.Err == nil || resMsg.Err.Error() != "42 is the ultimate answer" {
		t.Fatalf("Response message has unexpected error: %v", resMsg.Err)
	}
}

func TestJsonHandlerPanics(t *testing.T) {

	spec := []interface{}{
		func(a, b, c int) {},
		func(a *int, b int) {},
		func(a string, b *[]byte) {},
		func(a, b *string) (int, int) { return 42, 42 },
		func(a, b *string) string { return "" },
	}

	for idx, fn := range spec {
		var didPanic bool
		catchPanicInJsonHandler(fn, &didPanic)

		if !didPanic {
			t.Fatalf("[fn #%d] Expected JsonHandler to panic", idx)
		}
	}
}

func catchPanicInJsonHandler(handler interface{}, didPanic *bool) {
	*didPanic = false
	defer func() {
		err := recover()
		if err != nil && err == "Argument signature must be a function receiving two pointer arguments to the request and response structs and return error" {
			*didPanic = true
		}
	}()

	JsonHandler(handler, false)
}
