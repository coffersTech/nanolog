package engine

// LogRow represents a single log record (row-oriented view).
// Used when reading data from disk or returning query results.
type LogRow struct {
	Timestamp int64
	Level     uint8
	Service   string
	Message   string
}

// Filter defines criteria for log retrieval.
type Filter struct {
	MinTime int64
	MaxTime int64
	Level   uint8
	Service string
	Query   string // Global keyword search in message
}
