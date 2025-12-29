package engine

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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

	// Stats Cache
	statsCache map[string]SystemStats
	mu         sync.RWMutex
}

// NewQueryEngine creates a new QueryEngine and initializes the stats cache.
func NewQueryEngine(dataDir string, mt *MemTable, readerFunc SnapshotReaderFunc, writerFunc SnapshotWriterFunc, retention time.Duration) *QueryEngine {
	qe := &QueryEngine{
		dataDir:    dataDir,
		mt:         mt,
		readerFunc: readerFunc,
		writerFunc: writerFunc,
		Retention:  retention,
		statsCache: make(map[string]SystemStats),
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

// ExecuteScan scans memory and then .nano files and returns up to `limit` rows matching the filter.
func (qe *QueryEngine) ExecuteScan(filter Filter, limit int) ([]LogRow, error) {
	// 1. Search MemTable first (memory)
	result := qe.mt.Search(filter, limit)

	if len(result) >= limit {
		return result, nil
	}

	// 2. Search persisted files
	files, err := qe.findNanoFiles()
	if err != nil {
		return result, err
	}

	// Sort files by name DESC (latest first)
	// Filenames are log_minTs_maxTs.nano, sorting DESC puts larger timestamps first.
	sort.Slice(files, func(i, j int) bool {
		return files[i] > files[j]
	})

	for _, file := range files {
		if len(result) >= limit {
			break
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

	for _, file := range files {
		// Optimization: Read all rows to aggregate stats.
		// In a real system, we might store these in a dedicated index or file header.
		rows, err := qe.readerFunc(file, Filter{})
		if err != nil {
			log.Printf("Failed to read file %s for stats: %v", file, err)
			continue
		}

		fStats := qe.computeStatsFromRows(rows)

		qe.mu.Lock()
		qe.statsCache[filepath.Base(file)] = fStats
		qe.mu.Unlock()
	}
	log.Printf("Loaded stats cache for %d files", len(qe.statsCache))
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
