package usrvtest

import "github.com/achilleasa/usrv"

type Message struct {
	// From
	F string

	// To
	T string

	// CorrelationID
	C string

	// Properties
	P usrv.Property

	// Content
	Cont []byte

	// Error
	Err error
}

func (m *Message) From() string                 { return m.F }
func (m *Message) To() string                   { return m.T }
func (m *Message) Property() usrv.Property      { return m.P }
func (m *Message) CorrelationId() string        { return m.C }
func (m *Message) Content() ([]byte, error)     { return m.Cont, m.Err }
func (m *Message) SetContent(c []byte, e error) { m.Cont, m.Err = c, e }
