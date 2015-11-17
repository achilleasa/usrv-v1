package usrv

// The Logger interface must be implemented by any
// logger that can be plugged into usrv.
//
// This interface has been copied from logxi (https://github.com/mgutz/logxi)
type Logger interface {
	Trace(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{}) error
	Error(msg string, args ...interface{}) error
	Fatal(msg string, args ...interface{})
}

// The NullLogger discards any log message
var NullLogger = nullLogger{}

type nullLogger struct{}

func (l nullLogger) Trace(msg string, args ...interface{}) {}

func (l nullLogger) Debug(msg string, args ...interface{}) {}

func (l nullLogger) Info(msg string, args ...interface{}) {}

func (l nullLogger) Warn(msg string, args ...interface{}) error { return nil }

func (l nullLogger) Error(msg string, args ...interface{}) error { return nil }

func (l nullLogger) Fatal(msg string, args ...interface{}) {}
