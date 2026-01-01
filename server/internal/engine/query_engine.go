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

	// mu protects mt pointer swaps
	mu sync.RWMutex

	// Persistent Stats
	globalStats PersistentStats
	statsLock   sync.RWMutex // Protects globalStats

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
		globalStats:  loadPersistentStats(dataDir),
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

	// Initial cache population (not needed with persistent stats)
	// qe.loadStatsCache()

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

	// === Step 1: Write file to disk ===
	if err := qe.writerFunc(path, qe.mt); err != nil {
		return err
	}

	// === Step 2: Atomic stats transfer ===
	qe.mt.mu.RLock()
	rowCount := len(qe.mt.TsCol)
	levelCounts := make(map[int]int64)
	serviceCounts := make(map[string]int64)
	var totalBytes int64

	for i := 0; i < rowCount; i++ {
		lvl := int(qe.mt.LvlCol[i])
		levelCounts[lvl]++
		svc := qe.mt.SvcCol[i]
		serviceCounts[svc]++
		totalBytes += int64(len(qe.mt.MsgCol[i]) + len(svc) + len(qe.mt.HostCol[i]) + 9)
	}
	qe.mt.mu.RUnlock()

	// Atomically update global stats
	qe.statsLock.Lock()
	qe.globalStats.TotalLogs += int64(rowCount)
	qe.globalStats.TotalBytes += totalBytes
	for k, v := range levelCounts {
		qe.globalStats.LevelCounts[k] += v
	}
	for k, v := range serviceCounts {
		qe.globalStats.ServiceCounts[k] += v
	}
	qe.statsLock.Unlock()

	// === Step 3: Persist stats to disk ===
	if err := savePersistentStats(qe.dataDir, qe.globalStats); err != nil {
		log.Printf("Stats persist error: %v", err)
	}

	// === Step 4: Reset MemTable and WAL ===
	qe.mt.Reset()

	if qe.wal != nil {
		if err := qe.wal.Reset(); err != nil {
			log.Printf("WAL reset error: %v", err)
		}
	}

	log.Printf("Flushed to disk: %s (%d rows)", filename, rowCount)
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

	// === Step 1: Write file to disk ===
	if err := qe.writerFunc(path, mt); err != nil {
		log.Printf("Background flush write error: %v", err)
		return
	}

	// === Step 2: Atomic stats transfer ===
	// Get snapshot of MemTable stats before any cleanup
	mt.mu.RLock()
	rowCount := len(mt.TsCol)
	levelCounts := make(map[int]int64)
	serviceCounts := make(map[string]int64)
	var totalBytes int64

	for i := 0; i < rowCount; i++ {
		lvl := int(mt.LvlCol[i])
		levelCounts[lvl]++

		svc := mt.SvcCol[i]
		serviceCounts[svc]++

		// Estimate bytes
		totalBytes += int64(len(mt.MsgCol[i]) + len(svc) + len(mt.HostCol[i]) + 9)
	}
	mt.mu.RUnlock()

	// Atomically update global stats
	qe.statsLock.Lock()
	qe.globalStats.TotalLogs += int64(rowCount)
	qe.globalStats.TotalBytes += totalBytes
	for k, v := range levelCounts {
		qe.globalStats.LevelCounts[k] += v
	}
	for k, v := range serviceCounts {
		qe.globalStats.ServiceCounts[k] += v
	}
	qe.statsLock.Unlock()

	// === Step 3: Persist stats to disk ===
	if err := savePersistentStats(qe.dataDir, qe.globalStats); err != nil {
		log.Printf("Stats persist error: %v", err)
	}

	// === Step 4: Cleanup - WAL reset (after stats are safely persisted) ===
	if qe.wal != nil {
		if err := qe.wal.Reset(); err != nil {
			log.Printf("WAL reset error: %v", err)
		}
	}

	log.Printf("Background flush completed: %s (%d rows)", filename, rowCount)
}

// ExecuteScan scans memory and then .nano files and returns up to `limit` rows matching the filter.
func (qe *QueryEngine) ExecuteScan(filter Filter, limit int) ([]LogRow, error) {
	// Parse NanoQL query if present
	var nqlNode interface{}
	if filter.Query != "" {
		node, err := ParseNanoQL(filter.Query)
		if err != nil {
			return nil, fmt.Errorf("invalid query syntax: %w", err)
		}
		nqlNode = node
	}

	// 1. Grab current MemTable under lock to avoid inconsistency if swapped
	qe.mu.RLock()
	mt := qe.mt
	qe.mu.RUnlock()

	// 2. Search MemTable first (memory) with NanoQL
	result := mt.SearchWithNanoQL(filter, nqlNode, limit)

	if len(result) >= limit {
		return result, nil
	}

	// 3. Search persisted files
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

		// Apply NanoQL filter to disk results
		if nqlNode != nil {
			filteredRows := make([]LogRow, 0, len(rows))
			for i := range rows {
				if MatchNanoQL(nqlNode, &rows[i]) {
					filteredRows = append(filteredRows, rows[i])
				}
			}
			rows = filteredRows
		}

		// Append rows up to limit
		remaining := limit - len(result)
		if len(rows) <= remaining {
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

// computeStatsFromRows and loadStatsCache are removed in favor of PersistentStats

// parseTsFromFilename extracts min and max timestamps from a log filename.
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

// ContextResult represents the result of a context query.
type ContextResult struct {
	Pre    []LogRow `json:"pre"`    // Logs before the anchor
	Anchor *LogRow  `json:"anchor"` // The target log
	Post   []LogRow `json:"post"`   // Logs after the anchor
}

// GetContext retrieves surrounding logs around a specific timestamp for a service.
func (qe *QueryEngine) GetContext(ts int64, service string, limit int) (*ContextResult, error) {
	if limit <= 0 {
		limit = 10
	}

	result := &ContextResult{
		Pre:  make([]LogRow, 0, limit),
		Post: make([]LogRow, 0, limit),
	}

	// Collect all matching logs from memory and disk
	filter := Filter{Service: service}

	// Get current MemTable
	qe.mu.RLock()
	mt := qe.mt
	qe.mu.RUnlock()

	// Search MemTable (returns newest first)
	memRows := mt.Search(filter, -1)

	// Search disk files
	files, err := qe.findNanoFiles()
	if err != nil {
		return nil, err
	}

	// Sort files by timestamp (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i] > files[j]
	})

	var diskRows []LogRow
	for _, file := range files {
		rows, err := qe.readerFunc(file, filter)
		if err != nil {
			continue
		}
		diskRows = append(diskRows, rows...)
	}

	// Combine and sort all rows by timestamp (ascending for easier processing)
	allRows := append(memRows, diskRows...)
	sort.Slice(allRows, func(i, j int) bool {
		return allRows[i].Timestamp < allRows[j].Timestamp
	})

	// Find anchor position
	anchorIdx := -1
	for i, row := range allRows {
		if row.Timestamp == ts {
			anchorIdx = i
			result.Anchor = &allRows[i]
			break
		}
	}

	if anchorIdx == -1 {
		// Anchor not found, try to find closest
		for i, row := range allRows {
			if row.Timestamp >= ts {
				if i > 0 && (ts-allRows[i-1].Timestamp) < (row.Timestamp-ts) {
					anchorIdx = i - 1
				} else {
					anchorIdx = i
				}
				result.Anchor = &allRows[anchorIdx]
				break
			}
		}
	}

	if result.Anchor == nil && len(allRows) > 0 {
		// Timestamp is beyond all logs, use last one
		anchorIdx = len(allRows) - 1
		result.Anchor = &allRows[anchorIdx]
	}

	if result.Anchor == nil {
		return result, nil // No logs found
	}

	// Collect pre (before anchor)
	preStart := anchorIdx - limit
	if preStart < 0 {
		preStart = 0
	}
	for i := preStart; i < anchorIdx; i++ {
		result.Pre = append(result.Pre, allRows[i])
	}

	// Collect post (after anchor)
	postEnd := anchorIdx + limit + 1
	if postEnd > len(allRows) {
		postEnd = len(allRows)
	}
	for i := anchorIdx + 1; i < postEnd; i++ {
		result.Post = append(result.Post, allRows[i])
	}

	return result, nil
}
