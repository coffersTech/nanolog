package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// PersistentStats holds cumulative statistics that survive restarts.
type PersistentStats struct {
	TotalLogs     int64            `json:"total_logs"`
	TotalBytes    int64            `json:"total_bytes"`
	LevelCounts   map[int]int64    `json:"level_counts"`   // Level (uint8) -> count
	ServiceCounts map[string]int64 `json:"service_counts"` // Service name -> count
}

// SystemStats contains high-level system metrics for API response.
type SystemStats struct {
	IngestionRate float64        `json:"ingestion_rate"` // logs/sec
	TotalLogs     int64          `json:"total_logs"`     // total count
	DiskUsage     int64          `json:"disk_usage"`     // bytes
	LevelDist     map[string]int `json:"level_dist"`     // e.g. "INFO": 100
	TopServices   map[string]int `json:"top_services"`   // e.g. "order-svc": 50
}

// statsFileName is the filename for persisted stats
const statsFileName = ".nanolog.stats"

// loadPersistentStats reads stats from disk.
func loadPersistentStats(dataDir string) PersistentStats {
	stats := PersistentStats{
		LevelCounts:   make(map[int]int64),
		ServiceCounts: make(map[string]int64),
	}

	path := filepath.Join(dataDir, statsFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist or can't be read, return empty stats
		return stats
	}

	if err := json.Unmarshal(data, &stats); err != nil {
		// Corrupted file, return empty stats
		return stats
	}

	// Ensure maps are initialized
	if stats.LevelCounts == nil {
		stats.LevelCounts = make(map[int]int64)
	}
	if stats.ServiceCounts == nil {
		stats.ServiceCounts = make(map[string]int64)
	}

	return stats
}

// savePersistentStats writes stats to disk atomically.
func savePersistentStats(dataDir string, stats PersistentStats) error {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dataDir, statsFileName)
	tmpPath := path + ".tmp"

	// Write to temp file first
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, path)
}
func (qe *QueryEngine) GetStats() SystemStats {
	// 1. Get snapshot of current MemTable stats
	qe.mu.RLock()
	mt := qe.mt
	qe.mu.RUnlock()
	memStats := mt.GetStats()

	// 2. Get persistent stats under lock
	qe.statsLock.RLock()
	diskStats := qe.globalStats
	qe.statsLock.RUnlock()

	// 3. Merge results
	stats := SystemStats{
		IngestionRate: mt.GetIngestionRate(),
		TotalLogs:     diskStats.TotalLogs + int64(memStats.RowCount),
		LevelDist:     make(map[string]int),
		TopServices:   make(map[string]int),
	}

	// Merge Level Distributions
	for lvl, count := range diskStats.LevelCounts {
		lvlStr := levelIntToString(lvl)
		stats.LevelDist[lvlStr] += int(count)
	}
	for lvl, count := range memStats.LevelCounts {
		lvlStr := levelIntToString(lvl)
		stats.LevelDist[lvlStr] += int(count)
	}

	// Merge Service Counters
	for svc, count := range diskStats.ServiceCounts {
		stats.TopServices[svc] += int(count)
	}
	for svc, count := range memStats.ServiceCounts {
		stats.TopServices[svc] += int(count)
	}

	// 4. Calculate actual Disk Usage
	var size int64
	_ = filepath.Walk(qe.dataDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	stats.DiskUsage = size

	return stats
}

// levelIntToString converts level int to string
func levelIntToString(lvl int) string {
	switch uint8(lvl) {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
