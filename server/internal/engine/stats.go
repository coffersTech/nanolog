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

// GetStats aggregates current system statistics from cache and MemTable.
func (qe *QueryEngine) GetStats() SystemStats {
	stats := SystemStats{
		LevelDist:   make(map[string]int),
		TopServices: make(map[string]int),
	}

	// 1. Ingestion Rate (from MemTable)
	stats.IngestionRate = qe.mt.GetIngestionRate()

	// 2. Aggregate from Cache (Persisted)
	qe.mu.RLock()
	var totalPersistedLogs int64
	for _, fStats := range qe.statsCache {
		totalPersistedLogs += fStats.TotalLogs
		for lvl, count := range fStats.LevelDist {
			stats.LevelDist[lvl] += count
		}
		for svc, count := range fStats.TopServices {
			stats.TopServices[svc] += count
		}
	}
	qe.mu.RUnlock()

	// 3. Disk Usage
	var size int64
	_ = filepath.Walk(qe.dataDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	stats.DiskUsage = size

	// 4. MemTable Stats (Live)
	qe.mt.mu.RLock()
	defer qe.mt.mu.RUnlock()

	rowCount := len(qe.mt.TsCol)
	stats.TotalLogs = totalPersistedLogs + int64(rowCount)

	for i := 0; i < rowCount; i++ {
		// Service
		svc := qe.mt.SvcCol[i]
		stats.TopServices[svc]++

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
		stats.LevelDist[lvlStr]++
	}

	return stats
}
