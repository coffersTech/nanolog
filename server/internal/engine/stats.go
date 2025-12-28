package engine

import (
	"os"
	"path/filepath"
)

// SystemStats contains high-level system metrics.
type SystemStats struct {
	IngestionRate float64        `json:"ingestion_rate"` // logs/sec
	TotalLogs     int64          `json:"total_logs"`     // total count
	DiskUsage     int64          `json:"disk_usage"`     // bytes
	LevelDist     map[string]int `json:"level_dist"`     // e.g. "INFO": 100
	TopServices   map[string]int `json:"top_services"`   // e.g. "order-svc": 50
}

// GetStats aggregates current system statistics.
func (qe *QueryEngine) GetStats() SystemStats {
	stats := SystemStats{}

	// 1. Ingestion Rate (from MemTable)
	stats.IngestionRate = qe.mt.GetIngestionRate()

	// 2. Disk Usage
	var size int64
	_ = filepath.Walk(qe.dataDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	stats.DiskUsage = size

	// 3. MemTable Stats (Live)
	// We need to iterate MemTable to get Top Services and Level Distribution.
	// We use direct column access strictly locked.
	qe.mt.mu.RLock()
	defer qe.mt.mu.RUnlock()

	svcCounts := make(map[string]int)
	lvlCounts := make(map[string]int)

	rowCount := len(qe.mt.TsCol)
	stats.TotalLogs = int64(rowCount)

	// Estimate total logs from disk size (simple heuristic for v1)
	if size > 0 {
		// Assuming roughly 50 bytes per compressed log
		stats.TotalLogs += size / 50
	}

	for i := 0; i < rowCount; i++ {
		// Service
		svc := qe.mt.SvcCol[i]
		svcCounts[svc]++

		// Level
		lvl := qe.mt.LvlCol[i]
		lvlStr := "UNKNOWN"
		switch lvl {
		case LevelDebug:
			lvlStr = "DEBUG"
		case LevelInfo:
			lvlStr = "INFO"
		case LevelWarn:
			lvlStr = "WARN"
		case LevelError:
			lvlStr = "ERROR"
		case LevelFatal:
			lvlStr = "FATAL"
		}
		lvlCounts[lvlStr]++
	}

	stats.LevelDist = lvlCounts

	// Top Services (Sort and trim)
	// We will return the map here, but for "Top" usually implies a list.
	// However, the requirement says `TopServices map[string]int`.
	// If the frontend expects a map, we pass the map.
	// But usually "Top" means filtered.
	// Let's implement full map for now, or filter if map gets too huge?
	// The requirement `TopServices map[string]int` suggests the API returns a map.
	// If we want "Top N", we calculate it.
	// But `map` in JSON is unordered.
	// Wait, the previous `TopServices` in `http.go` was `[]ServiceStat`.
	// The USER requirement for `stats.go` specified `TopServices map[string]int`.
	// "TopServices map[string]int `json:"top_services"`"
	// So I will return the map. It might contain ALL services.
	// For high cardinality, this is bad, but for V1 and "TopServices" name it's acceptable if N is small.
	// We can limit it to top 10 to be safe?
	// Since the type is map, we can't easily force order.
	// I will return the full map of services found in MemTable.
	// This is "Active Services" effectively.
	stats.TopServices = svcCounts

	return stats
}
