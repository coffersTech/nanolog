package engine

// LogRow represents a single log record (row-oriented view).
// Used when reading data from disk or returning query results.
type LogRow struct {
	Timestamp int64  `json:"timestamp"`
	Level     uint8  `json:"level"`
	Service   string `json:"service"`
	Host      string `json:"host"`
	Message   string `json:"message"`
}

// Filter defines criteria for log retrieval.
type Filter struct {
	MinTime int64  `json:"min_time"`
	MaxTime int64  `json:"max_time"`
	Level   uint8  `json:"level"`
	Service string `json:"service"`
	Host    string `json:"host"`
	Query   string `json:"q"` // Global keyword search in message
}
