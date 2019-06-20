package types

import (
	"os"

	"github.com/meinside/loggly-go"
)

// LogMessage for embedding into log message types
type LogMessage struct {
	Hostname  string `json:"hostname,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// MarkTimestamp marks `Timestamp` as current time
func (m *LogMessage) MarkTimestamp() {
	m.Hostname, _ = os.Hostname()
	_, m.Timestamp = loggly.Timestamp()
}
