package usrvtest

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/achilleasa/usrv"
)

func TestMessage(t *testing.T) {
	msg := Message{
		F: "from",
		T: "to",
		C: "correlationId",
		P: usrv.Property{"foo": "bar"},
	}

	expContent := []byte("Test")
	expError := fmt.Errorf("An error")
	msg.SetContent(expContent, expError)

	if msg.From() != msg.F {
		t.Fatalf("Expected msg.From() to return '%s'; got %v", msg.F, msg.From())
	}

	if msg.To() != msg.T {
		t.Fatalf("Expected msg.To() to return '%s'; got %v", msg.T, msg.To())
	}

	if msg.CorrelationId() != msg.C {
		t.Fatalf("Expected msg.CorrelationId() to return '%s'; got %v", msg.C, msg.CorrelationId())
	}

	if !reflect.DeepEqual(msg.P, msg.Property()) {
		t.Fatalf("Expected msg.Property() to return '%v'; got %v", msg.P, msg.Property())
	}

	content, err := msg.Content()
	if !reflect.DeepEqual(content, msg.Cont) {
		t.Fatalf("Expected content to equal %v; got %v", msg.Cont, content)
	}

	if err != msg.Err {
		t.Fatalf("Expected error to equal %v; got %v", msg.Err, err)
	}
}
