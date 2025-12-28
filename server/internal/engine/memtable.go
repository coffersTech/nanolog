package engine

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	LevelDebug   = 0
	LevelInfo    = 1
	LevelWarn    = 2
	LevelError   = 3
	LevelFatal   = 4
	LevelUnknown = 255
)

// MemTable stores logs in columnar format.
// Columns are exported for access by storage package.
type MemTable struct {
	mu sync.RWMutex

	// Exported Columns
	TsCol   []int64  // Timestamp
	LvlCol  []uint8  // Level (dictionary encoded)
	SvcCol  []string // Service name
	HostCol []string // Hostname/IP
	MsgCol  []string // Message content

	// Metadata
	size int64 // Estimated memory usage in bytes

	// Stats
	writeCounter int64   // Atomic counter for ingestion
	currentRate  float64 // Logs per second
}

// NewMemTable initializes MemTable with pre-allocated capacity.
func NewMemTable() *MemTable {
	cap := 4096
	return &MemTable{
		TsCol:   make([]int64, 0, cap),
		LvlCol:  make([]uint8, 0, cap),
		SvcCol:  make([]string, 0, cap),
		HostCol: make([]string, 0, cap),
		MsgCol:  make([]string, 0, cap),
		size:    0,
	}
}

// Append adds a log entry.
func (mt *MemTable) Append(ts int64, level string, service string, host string, msg string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.TsCol = append(mt.TsCol, ts)
	lvl := encodeLevel(level)
	mt.LvlCol = append(mt.LvlCol, lvl)
	mt.SvcCol = append(mt.SvcCol, service)
	mt.HostCol = append(mt.HostCol, host)
	mt.MsgCol = append(mt.MsgCol, msg)

	// Update size estimate
	mt.size += 8 + 1 + int64(len(service)+16) + int64(len(host)+16) + int64(len(msg)+16)

	// Update stats counter
	atomic.AddInt64(&mt.writeCounter, 1)
}

// Size returns the estimated memory usage in bytes.
func (mt *MemTable) Size() int64 {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.size
}

// Len returns the number of rows.
func (mt *MemTable) Len() int {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return len(mt.TsCol)
}

// Reset clears all column data for memory reuse.
func (mt *MemTable) Reset() {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.TsCol = mt.TsCol[:0]
	mt.LvlCol = mt.LvlCol[:0]
	mt.SvcCol = mt.SvcCol[:0]
	mt.HostCol = mt.HostCol[:0]
	mt.MsgCol = mt.MsgCol[:0]
	mt.size = 0
}

// MinTimestamp returns the minimum timestamp (first element).
func (mt *MemTable) MinTimestamp() int64 {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	if len(mt.TsCol) == 0 {
		return 0
	}
	return mt.TsCol[0]
}

// MaxTimestamp returns the maximum timestamp (last element).
func (mt *MemTable) MaxTimestamp() int64 {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	if len(mt.TsCol) == 0 {
		return 0
	}
	return mt.TsCol[len(mt.TsCol)-1]
}

// Search filters in-memory logs based on criteria, newest first.
func (mt *MemTable) Search(filter Filter, limit int) []LogRow {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	var result []LogRow
	rowCount := len(mt.TsCol)

	// Scan backwards (newest first)
	for i := rowCount - 1; i >= 0; i-- {
		if len(result) >= limit {
			break
		}

		ts := mt.TsCol[i]
		if filter.MinTime > 0 && ts < filter.MinTime {
			continue
		}
		if filter.MaxTime > 0 && ts > filter.MaxTime {
			continue
		}

		lvl := mt.LvlCol[i]
		if filter.Level > 0 && lvl != filter.Level {
			continue
		}

		svc := mt.SvcCol[i]
		if filter.Service != "" && svc != filter.Service {
			continue
		}

		host := mt.HostCol[i]
		if filter.Host != "" && host != filter.Host {
			continue
		}

		msg := mt.MsgCol[i]
		if filter.Query != "" && !strings.Contains(msg, filter.Query) {
			continue
		}

		result = append(result, LogRow{
			Timestamp: ts,
			Level:     lvl,
			Service:   svc,
			Host:      host,
			Message:   msg,
		})
	}

	return result
}

func encodeLevel(l string) uint8 {
	switch strings.ToUpper(l) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "FATAL":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// StartStatsTicker starts a background ticker to calculate ingestion rate.
func (mt *MemTable) StartStatsTicker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			count := atomic.SwapInt64(&mt.writeCounter, 0)
			rate := float64(count) / interval.Seconds()
			mt.mu.Lock()
			mt.currentRate = rate
			mt.mu.Unlock()
		}
	}()
}

// GetIngestionRate returns the current ingestion rate (logs/sec).
func (mt *MemTable) GetIngestionRate() float64 {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.currentRate
}
