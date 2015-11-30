package usrvtest

type LogEntry struct {
	Level   string
	Message string
	Context map[string]interface{}
}

type Logger struct {
	Entries []LogEntry
}

func (l *Logger) log(level string, msg string, args ...interface{}) {
	entry := LogEntry{
		Level:   level,
		Message: msg,
		Context: make(map[string]interface{}, 0),
	}

	for idx := 0; idx < len(args); idx += 2 {
		entry.Context[args[idx].(string)] = args[idx+1]
	}
	l.Entries = append(l.Entries, entry)
}

func (l *Logger) Trace(msg string, args ...interface{}) { l.log("trace", msg, args...) }

func (l *Logger) Debug(msg string, args ...interface{}) { l.log("debug", msg, args...) }

func (l *Logger) Info(msg string, args ...interface{}) { l.log("info", msg, args...) }

func (l *Logger) Warn(msg string, args ...interface{}) error {
	l.log("warn", msg, args...)
	return nil
}

func (l *Logger) Error(msg string, args ...interface{}) error {
	l.log("error", msg, args...)
	return nil
}

func (l *Logger) Fatal(msg string, args ...interface{}) { l.log("fatal", msg, args...) }
