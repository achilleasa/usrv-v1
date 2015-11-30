package middleware

import (
	"fmt"
	"testing"

	"github.com/achilleasa/usrv"
	"github.com/achilleasa/usrv/usrvtest"
)

func TestLogMiddlewareOnSuccess(t *testing.T) {
	handler := func(req usrv.Message, res usrv.Message) {
		res.SetContent([]byte("OK"), nil)
	}

	req := &usrvtest.Message{
		F:    "api",
		T:    "service",
		C:    "123",
		Cont: []byte("REQ"),
	}

	res := &usrvtest.Message{
		F: "service",
		T: "api",
		C: "123",
	}

	logger := &usrvtest.Logger{}
	logHandler := LogRequest(logger, handler)
	logHandler(req, res)

	if len(logger.Entries) != 1 {
		t.Fatalf("Expected to log 1 entry; got %d", len(logger.Entries))
	}
	entry := logger.Entries[0]

	if entry.Message != "Processed request" {
		t.Fatalf("Expected log entry message to be 'Processed request'; got %s", entry.Message)
	}
	if entry.Level != "info" {
		t.Fatalf("Expected log entry level to be 'info'; got %s", entry.Level)
	}

	exp := map[string]interface{}{
		"from":           req.From(),
		"to":             req.To(),
		"correlation_id": req.CorrelationId(),
		"req_len":        len(req.Cont),
		"res_len":        len(res.Cont),
	}
	for k, v := range exp {
		if entry.Context[k] != v {
			t.Fatalf("Expected logger key '%s' to contain value %v; got %v", k, v, entry.Context[k])
		}
	}

	procTime, ok := entry.Context["time"].(int64)
	if !ok || procTime <= 0 {
		t.Fatalf("Expected logger key 'time' entry to contain a non-zero value; got %v", procTime)
	}
}

func TestLogMiddlewareOnError(t *testing.T) {
	expError := fmt.Errorf("Error")
	handler := func(req usrv.Message, res usrv.Message) {
		res.SetContent(nil, expError)
	}

	req := &usrvtest.Message{
		F:    "api",
		T:    "service",
		C:    "123",
		Cont: []byte("REQ"),
	}

	res := &usrvtest.Message{
		F: "service",
		T: "api",
		C: "123",
	}

	logger := &usrvtest.Logger{}
	logHandler := LogRequest(logger, handler)
	logHandler(req, res)

	if len(logger.Entries) != 1 {
		t.Fatalf("Expected to log 1 entry; got %d", len(logger.Entries))
	}
	entry := logger.Entries[0]

	if entry.Message != "Request failed" {
		t.Fatalf("Expected log entry message to be 'Request failed'; got %s", entry.Message)
	}
	if entry.Level != "error" {
		t.Fatalf("Expected log entry level to be 'error'; got %s", entry.Level)
	}

	exp := map[string]interface{}{
		"from":           req.From(),
		"to":             req.To(),
		"correlation_id": req.CorrelationId(),
		"req_len":        len(req.Cont),
		"error":          expError,
	}
	for k, v := range exp {
		if entry.Context[k] != v {
			t.Fatalf("Expected logger key '%s' to contain value %v; got %v", k, v, entry.Context[k])
		}
	}

	procTime, ok := entry.Context["time"].(int64)
	if !ok || procTime <= 0 {
		t.Fatalf("Expected logger key 'time' entry to contain a non-zero value; got %v", procTime)
	}
}
