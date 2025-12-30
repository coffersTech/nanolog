package engine

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SnapshotReaderFunc is a function type for reading .nano files with filtering.
type SnapshotReaderFunc func(filename string, filter Filter) ([]LogRow, error)

// SnapshotWriterFunc is a function type for writing MemTable to a .nano file.
type SnapshotWriterFunc func(path string, mt *MemTable) error

// QueryEngine handles query execution and data lifecycle across persisted data.
type QueryEngine struct {
	dataDir    string
	mt         *MemTable
	readerFunc SnapshotReaderFunc
	writerFunc SnapshotWriterFunc
	Retention  time.Duration

	// Configuration
	MaxTableSize int64

	// Stats Cache
	statsCache map[string]SystemStats
	mu         sync.RWMutex

	// WAL for crash recovery
	wal *WAL
}

// NewQueryEngine creates a new QueryEngine and initializes the stats cache.
func NewQueryEngine(dataDir string, mt *MemTable, readerFunc SnapshotReaderFunc, writerFunc SnapshotWriterFunc, retention time.Duration) *QueryEngine {
	// Initialize WAL
	walPath := filepath.Join(dataDir, "wal.log")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("Warning: failed to create data dir for WAL: %v", err)
	}

	wal, err := OpenWAL(walPath)
	if err != nil {
		log.Printf("Warning: failed to open WAL: %v", err)
	}

	qe := &QueryEngine{
		dataDir:      dataDir,
		mt:           mt,
		readerFunc:   readerFunc,
		writerFunc:   writerFunc,
		Retention:    retention,
		MaxTableSize: 64 * 1024 * 1024, // 64MB Default
		statsCache:   make(map[string]SystemStats),
		wal:          wal,
	}

	// Crash Recovery: Replay WAL if it has data
	if wal != nil {
		recoveredRows, err := wal.Replay()
		if err == nil && len(recoveredRows) > 0 {
			log.Printf("Crash recovery: replaying %d logs from WAL...", len(recoveredRows))
			for _, row := range recoveredRows {
				// Re-append to current MemTable
				// Note: We avoid calling qe.Ingest here to prevent re-writing to WAL
				qe.mt.Append(row.Timestamp, DecodeLevel(row.Level), row.Service, row.Host, row.Message)
			}
		} else if err != nil {
			log.Printf("WAL replay warning: %v", err)
		}
	}

	// Initial cache population
	qe.loadStatsCache()

	return qe
}

// Flush writes the current MemTable to disk and resets it.
func (qe *QueryEngine) Flush() error {
	if qe.mt.Len() == 0 {
		return nil
	}

	// Ensure data directory exists
	if err := os.MkdirAll(qe.dataDir, 0755); err != nil {
		return err
	}

	minTs := qe.mt.MinTimestamp()
	maxTs := qe.mt.MaxTimestamp()
	filename := fmt.Sprintf("log_%d_%d.nano", minTs, maxTs)
	path := filepath.Join(qe.dataDir, filename)

	// Compute stats before reset
	rows := qe.mt.Search(Filter{}, -1)
	fStats := qe.computeStatsFromRows(rows)

	if err := qe.writerFunc(path, qe.mt); err != nil {
		return err
	}

	// Update cache
	qe.mu.Lock()
	qe.statsCache[filename] = fStats
	qe.mu.Unlock()

	qe.mt.Reset()
	log.Printf("Flushed to disk: %s", filename)
	return nil
}

// Ingest adds a log row to the WAL and MemTable, triggering a background flush if needed.
func (qe *QueryEngine) Ingest(ts int64, level, service, host, msg string) {
	// 1. Write to WAL first for durability
	if qe.wal != nil {
		if err := qe.wal.Write(ts, level, service, host, msg); err != nil {
			log.Printf("WAL write error: %v", err)
		}
	}

	// 2. Append to MemTable
	qe.mt.Append(ts, level, service, host, msg)

	// Periodically log size for user visibility (every ~10MB)
	currentSize := qe.mt.GetSize()
	if currentSize > 0 && currentSize%(10*1024*1024) < 2000 { // Approx every 10MB
		log.Printf("Current MemTable size: %.2f MB / %d MB", float64(currentSize)/(1024*1024), qe.MaxTableSize/(1024*1024))
	}

	if currentSize >= qe.MaxTableSize {
		qe.mu.Lock()
		// Double check size under lock
		if qe.mt.GetSize() < qe.MaxTableSize {
			qe.mu.Unlock()
			return
		}

		log.Printf("MemTable reached threshold (%d MB), swapping for async flush...", qe.MaxTableSize/(1024*1024))
		oldTable := qe.mt
		qe.mt = NewMemTable()
		// Inherit stats ticker for the new table
		qe.mt.StartStatsTicker(1 * time.Second)
		qe.mu.Unlock()

		// Background flush
		go qe.flushMemTable(oldTable)
	}
}

// SyncWAL flushes the WAL file to disk.
func (qe *QueryEngine) SyncWAL() {
	if qe.wal != nil {
		if err := qe.wal.Sync(); err != nil {
			log.Printf("WAL sync error: %v", err)
		}
	}
}

func (qe *QueryEngine) flushMemTable(mt *MemTable) {
	if mt.Len() == 0 {
		return
	}

	// Ensure data directory exists
	if err := os.MkdirAll(qe.dataDir, 0755); err != nil {
		log.Printf("Background flush directory error: %v", err)
		return
	}

	minTs := mt.MinTimestamp()
	maxTs := mt.MaxTimestamp()
	filename := fmt.Sprintf("log_%d_%d.nano", minTs, maxTs)
	path := filepath.Join(qe.dataDir, filename)

	// Compute stats for cache
	rows := mt.Search(Filter{}, -1)
	fStats := qe.computeStatsFromRows(rows)

	if err := qe.writerFunc(path, mt); err != nil {
		log.Printf("Background flush write error: %v", err)
		return
	}

	// Store in cache
	qe.mu.Lock()
	qe.statsCache[filename] = fStats
	qe.mu.Unlock()

	// Truncate WAL after successful write
	if qe.wal != nil {
		if err := qe.wal.Reset(); err != nil {
			log.Printf("WAL reset error: %v", err)
		}
	}

	log.Printf("Background flush completed: %s", filename)
}

// ExecuteScan scans memory and then .nano files and returns up to `limit` rows matching the filter.
func (qe *QueryEngine) ExecuteScan(filter Filter, limit int) ([]LogRow, error) {
	// 1. Grab current MemTable under lock to avoid inconsistency if swapped
	qe.mu.RLock()
	mt := qe.mt
	qe.mu.RUnlock()

	// 2. Search MemTable first (memory)
	result := mt.Search(filter, limit)

	if len(result) >= limit {
		return result, nil
	}

	// 2. Search persisted files
	files, err := qe.findNanoFiles()
	if err != nil {
		return result, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i] > files[j]
	})

	for _, file := range files {
		if len(result) >= limit {
			break
		}

		// File Pruning: Parse timestamps from filename (log_minTs_maxTs.nano)
		minTs, maxTs, err := parseTsFromFilename(file)
		if err == nil {
			if filter.MinTime > 0 && maxTs < filter.MinTime {
				continue // File is too old
			}
			if filter.MaxTime > 0 && minTs > filter.MaxTime {
				continue // File is too new
			}
		}

		rows, err := qe.readerFunc(file, filter)
		if err != nil {
			// Log error but continue with other files
			continue
		}

		// Append rows up to limit
		remaining := limit - len(result)
		if len(rows) <= remaining {
			// Files internally are sorted ASC, but we want DESC result.
			// However, for simplicity now we just append.
			// High performance result merging would be better.
			result = append(result, rows...)
		} else {
			result = append(result, rows[:remaining]...)
		}
	}

	return result, nil
}

// findNanoFiles returns all .nano files in the data directory.
func (qe *QueryEngine) findNanoFiles() ([]string, error) {
	var files []string

	entries, err := os.ReadDir(qe.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return files, nil // Empty result if dir doesn't exist
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".nano") {
			files = append(files, filepath.Join(qe.dataDir, entry.Name()))
		}
	}

	return files, nil
}

func (qe *QueryEngine) loadStatsCache() {
	files, err := qe.findNanoFiles()
	if err != nil {
		log.Printf("Failed to load stats cache: %v", err)
		return
	}

	corruptedCount := 0
	for _, file := range files {
		// Optimization: Read all rows to aggregate stats.
		rows, err := qe.readerFunc(file, Filter{})
		if err != nil {
			// If file is corrupted, we log and skip it.
			// In the future, we could move it to a 'corrupted' subfolder.
			log.Printf("Skipping corrupted file %s: %v", filepath.Base(file), err)
			corruptedCount++
			continue
		}

		fStats := qe.computeStatsFromRows(rows)

		qe.mu.Lock()
		qe.statsCache[filepath.Base(file)] = fStats
		qe.mu.Unlock()
	}

	if corruptedCount > 0 {
		log.Printf("Loaded stats cache: %d files loaded, %d corrupted files skipped", len(qe.statsCache), corruptedCount)
	} else {
		log.Printf("Loaded stats cache for %d files", len(qe.statsCache))
	}
}

func (qe *QueryEngine) computeStatsFromRows(rows []LogRow) SystemStats {
	s := SystemStats{
		TotalLogs:   int64(len(rows)),
		LevelDist:   make(map[string]int),
		TopServices: make(map[string]int),
	}
	for _, r := range rows {
		lvlStr := "UNKNOWN"
		switch r.Level {
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
		s.LevelDist[lvlStr]++
		s.TopServices[r.Service]++
	}
	return s
}

func parseTsFromFilename(filename string) (int64, int64, error) {
	base := filepath.Base(filename)
	if !strings.HasPrefix(base, "log_") || !strings.HasSuffix(base, ".nano") {
		return 0, 0, fmt.Errorf("invalid format")
	}
	content := strings.TrimSuffix(strings.TrimPrefix(base, "log_"), ".nano")
	parts := strings.Split(content, "_")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid parts")
	}
	minTs, err1 := strconv.ParseInt(parts[0], 10, 64)
	maxTs, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("invalid timestamps")
	}
	return minTs, maxTs, nil
}
