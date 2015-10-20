package usrv

// Common message property names
const (
	PropertyHasError = "error"
)

type Property map[string]string

// Get a key from the property set.
func (p Property) Get(key string) string {
	return p[key]
}

// Set a property key.
func (p Property) Set(key string, value string) {
	p[key] = value
}

// Delete a propery key.
func (p Property) Del(key string) {
	delete(p, key)
}

// The message interface is implemented by all messages
// that can be passed through a transport.
type Message interface {
	From() string
	To() string
	Property() Property

	Content() ([]byte, error)
	SetContent([]byte, error)
}
