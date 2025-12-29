package engine

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RunCleaner periodically scans the data directory and removes expired .nano files.
func (qe *QueryEngine) RunCleaner(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Cleaner started. Retention: %v, Interval: %v", qe.Retention, interval)

	for range ticker.C {
		if qe.Retention <= 0 {
			continue
		}
		qe.purgeExpiredFiles()
	}
}

func (qe *QueryEngine) purgeExpiredFiles() {
	entries, err := os.ReadDir(qe.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("Cleaner error: failed to read data dir: %v", err)
		return
	}

	now := time.Now()
	threshold := now.Add(-qe.Retention).UnixNano()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".nano") {
			continue
		}

		// Filename format: log_{minTs}_{maxTs}.nano
		name := entry.Name()
		maxTs, err := extractMaxTs(name)
		if err != nil {
			continue // Skip files with unexpected names
		}

		if maxTs < threshold {
			path := filepath.Join(qe.dataDir, name)
			if err := os.Remove(path); err != nil {
				log.Printf("Cleaner error: failed to delete %s: %v", name, err)
			} else {
				log.Printf("Expired file deleted: %s", name)
				// Update stats cache
				qe.mu.Lock()
				delete(qe.statsCache, name)
				qe.mu.Unlock()
			}
		}
	}
}

func extractMaxTs(filename string) (int64, error) {
	// log_1735230000_1735233600.nano
	base := strings.TrimSuffix(filename, ".nano")
	parts := strings.Split(base, "_")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid format")
	}

	maxTs, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return 0, err
	}

	return maxTs, nil
}
