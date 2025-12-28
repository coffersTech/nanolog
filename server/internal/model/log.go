package model

// LogRecord represents a structured log entry.
// This is the logical view of a log, used for higher-level processing
// before ingestion (if needed) or during query (reassembly).
type LogRecord struct {
	Timestamp  int64
	TraceID    string
	Level      int
	Service    string
	Host       string
	Message    string
	Attributes map[string]interface{}
}
