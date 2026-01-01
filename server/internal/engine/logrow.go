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

// Getter methods to implement nanoql.LogRecord interface

func (r *LogRow) GetTimestamp() int64 { return r.Timestamp }
func (r *LogRow) GetLevel() uint8     { return r.Level }
func (r *LogRow) GetService() string  { return r.Service }
func (r *LogRow) GetHost() string     { return r.Host }
func (r *LogRow) GetMessage() string  { return r.Message }

// Filter defines criteria for log retrieval.
type Filter struct {
	MinTime int64  `json:"min_time"`
	MaxTime int64  `json:"max_time"`
	Level   uint8  `json:"level"`
	Service string `json:"service"`
	Host    string `json:"host"`
	Query   string `json:"q"` // NanoQL query string
}
