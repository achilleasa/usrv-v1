package usrvtest

import "testing"

func TestLogger(t *testing.T) {
	logger := &Logger{}

	levels := []string{
		"trace", "debug", "info", "warn", "error", "fatal",
	}
	args := []interface{}{
		"k1", "v1",
		"k2", "v2",
	}

	logger.Trace("Test", args...)
	logger.Debug("Test", args...)
	logger.Info("Test", args...)
	logger.Warn("Test", args...)
	logger.Error("Test", args...)
	logger.Fatal("Test", args...)

	if len(logger.Entries) != len(levels) {
		t.Fatalf("Expected logger to contain %d entries; got %d", len(levels), len(logger.Entries))
	}

	for idx, level := range levels {
		entry := logger.Entries[idx]
		if entry.Message != "Test" {
			t.Fatalf("[entry %d] Expected message to be 'Test'; got %s", idx, entry.Message)
		}
		if entry.Level != level {
			t.Fatalf("[entry %d] Expected level to be '%s'; got %s", idx, level, entry.Level)
		}
		for aidx := 0; aidx < len(args)/2; aidx += 2 {
			if entry.Context[args[aidx].(string)] != args[aidx+1] {
				t.Fatalf("[entry %d] Expected context entry with key '%s' to have value '%v' got '%v'", args[aidx], args[aidx+1], entry.Context[args[aidx].(string)])
			}
		}
	}
}
