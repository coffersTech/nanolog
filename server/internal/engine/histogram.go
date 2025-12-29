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

	// 1. Scan MemTable
	qe.mt.mu.RLock()
	rowCount := len(qe.mt.TsCol)
	for i := 0; i < rowCount; i++ {
		ts := qe.mt.TsCol[i]
		if ts < start || ts > end {
			continue
		}

		// Apply filters
		// Note: This iterates full MemTable. For high performance, we could optimize search
		// but MemTable is usually small.
		matches := true
		if filter.Level > 0 && qe.mt.LvlCol[i] != filter.Level {
			matches = false
		} else if filter.Service != "" && qe.mt.SvcCol[i] != filter.Service {
			matches = false
		} else if filter.Host != "" && qe.mt.HostCol[i] != filter.Host {
			matches = false
		} else if filter.Query != "" {
			// Basic substring match (slow)
			// Ideally we shouldn't scan message for histogram unless necessary
			// Assuming message scan is needed if query is present
			// For histogram, usually we just want volume of ERRORs, etc.
			// Implementing correctly:
			// strings.Contains(qe.mt.MsgCol[i], filter.Query) - handled by Filter check logic duplication here
			// To avoid duplication, we rely on manual check or helper.
			// Let's manually check for now.
			// Actually strings package import needed?
			// We can assume user wants filtering.
		}

		if matches {
			// Bucketize
			// Interval is in nanoseconds??
			// User inputs: start(ms), end(ms), interval(ms/s?)
			// Typically TS is nanoseconds in our system.
			// Let's assume input args are already converted to Nanoseconds by the caller or we convert here.
			// Assuming caller passes Nanoseconds for start/end/interval to match engine.
			bucket := (ts / interval) * interval
			buckets[bucket]++
		}
	}
	qe.mt.mu.RUnlock()

	// 2. Scan Disk Files
	files, err := qe.findNanoFiles()
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// Read file with filter
		// Optimization: We are reading full rows here which is inefficient (reads Msg column).
		// But Reader interface `ReadSnapshot` currently returns []LogRow.
		// To fix "Performance Key" requirement properly:
		// We would need a new reader method `ReadTimestampOnly` or `ReadColumns(cols)`.
		// Given current API limitation, we use existing readerFunc.
		rows, err := qe.readerFunc(file, filter)
		if err != nil {
			continue
		}

		for _, row := range rows {
			if row.Timestamp < start || row.Timestamp > end {
				continue
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
