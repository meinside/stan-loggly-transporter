package types

import (
	"github.com/meinside/loggly-go"
)

// LogMessage for embedding into log message types
type LogMessage struct {
	Timestamp string `json:"timestamp,omitempty"`
}

// MarkTimestamp marks `Timestamp` as current time
func (m *LogMessage) MarkTimestamp() {
	_, m.Timestamp = loggly.Timestamp()
}
