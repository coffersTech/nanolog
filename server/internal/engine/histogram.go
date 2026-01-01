package engine

import (
	"sort"
)

type HistogramPoint struct {
	Time  int64 `json:"time"`
	Count int   `json:"count"`
}

// ComputeHistogram aggregates log counts over time buckets.
func (qe *QueryEngine) ComputeHistogram(start, end int64, interval int64, filter Filter) ([]HistogramPoint, error) {
	// Map to store bucket counts: timestamp -> count
	buckets := make(map[int64]int)

	// Parse NanoQL query if present
	var nqlNode interface{}
	if filter.Query != "" {
		node, err := ParseNanoQL(filter.Query)
		if err != nil {
			return nil, err
		}
		nqlNode = node
	}

	// 1. Scan MemTable
	qe.mt.mu.RLock()
	rowCount := len(qe.mt.TsCol)
	for i := 0; i < rowCount; i++ {
		ts := qe.mt.TsCol[i]
		if ts < start || ts > end {
			continue
		}

		// Build row for NanoQL matching
		row := LogRow{
			Timestamp: ts,
			Level:     qe.mt.LvlCol[i],
			Service:   qe.mt.SvcCol[i],
			Host:      qe.mt.HostCol[i],
			Message:   qe.mt.MsgCol[i],
		}

		// Apply NanoQL filter if present
		if nqlNode != nil {
			if !MatchNanoQL(nqlNode, &row) {
				continue
			}
		} else {
			// Legacy filter logic (when no NanoQL)
			if filter.Level > 0 && row.Level != filter.Level {
				continue
			}
			if filter.Service != "" && row.Service != filter.Service {
				continue
			}
			if filter.Host != "" && row.Host != filter.Host {
				continue
			}
		}

		// Bucketize
		bucket := (ts / interval) * interval
		buckets[bucket]++
	}
	qe.mt.mu.RUnlock()

	// 2. Scan Disk Files
	files, err := qe.findNanoFiles()
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// File Pruning: Parse timestamps from filename (log_minTs_maxTs.nano)
		minTs, maxTs, err := parseTsFromFilename(file)
		if err == nil {
			if start > 0 && maxTs < start {
				continue // File is too old
			}
			if end > 0 && minTs > end {
				continue // File is too new
			}
		}

		// Read file with basic filter (time pruning)
		rows, err := qe.readerFunc(file, filter)
		if err != nil {
			continue
		}

		for _, row := range rows {
			if row.Timestamp < start || row.Timestamp > end {
				continue
			}

			// Apply NanoQL filter if present
			if nqlNode != nil {
				if !MatchNanoQL(nqlNode, &row) {
					continue
				}
			}

			bucket := (row.Timestamp / interval) * interval
			buckets[bucket]++
		}
	}

	// 3. Convert Map to Sorted Slice
	var points []HistogramPoint
	for t, c := range buckets {
		points = append(points, HistogramPoint{Time: t, Count: c})
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Time < points[j].Time
	})

	return points, nil
}
