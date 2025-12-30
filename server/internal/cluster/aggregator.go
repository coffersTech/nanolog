package cluster

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/coffersTech/nanolog/server/internal/engine"
)

// Aggregator centralizes distributed query logic for the Console node.
type Aggregator struct {
	DataNodes []string
	Client    *http.Client
}

// NewAggregator creates a new Aggregator instance.
type QueryParams struct {
	RawQuery string
	Limit    int
	Auth     string
}

func NewAggregator(nodes []string) *Aggregator {
	return &Aggregator{
		DataNodes: nodes,
		Client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Search performs a scatter-gather query across all data nodes.
func (a *Aggregator) Search(params QueryParams) ([]engine.LogRow, error) {
	var allRows []engine.LogRow
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, node := range a.DataNodes {
		wg.Add(1)
		go func(nodeURL string) {
			defer wg.Done()
			url := fmt.Sprintf("%s/api/search?%s", nodeURL, params.RawQuery)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return
			}
			if params.Auth != "" {
				req.Header.Set("Authorization", params.Auth)
			}

			resp, err := a.Client.Do(req)
			if err != nil {
				log.Printf("[Aggregator] Error from node %s: %v", nodeURL, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var rows []engine.LogRow
				if err := json.NewDecoder(resp.Body).Decode(&rows); err == nil {
					mu.Lock()
					allRows = append(allRows, rows...)
					mu.Unlock()
				}
			} else {
				log.Printf("[Aggregator] Node %s returned status %d", nodeURL, resp.StatusCode)
			}
		}(node)
	}

	wg.Wait()

	// Merge-Sort by timestamp descending
	sort.Slice(allRows, func(i, j int) bool {
		return allRows[i].Timestamp > allRows[j].Timestamp
	})

	// Consolidate and Limit
	if len(allRows) > params.Limit {
		allRows = allRows[:params.Limit]
	}

	return allRows, nil
}

// Histogram performs scatter-gather histogram aggregation.
func (a *Aggregator) Histogram(params QueryParams) ([]engine.HistogramPoint, error) {
	combined := make(map[int64]int64)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, node := range a.DataNodes {
		wg.Add(1)
		go func(nodeURL string) {
			defer wg.Done()
			url := fmt.Sprintf("%s/api/histogram?%s", nodeURL, params.RawQuery)
			req, _ := http.NewRequest("GET", url, nil)
			if params.Auth != "" {
				req.Header.Set("Authorization", params.Auth)
			}
			resp, err := a.Client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var data []engine.HistogramPoint
				if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
					mu.Lock()
					for _, p := range data {
						combined[p.Time] += int64(p.Count)
					}
					mu.Unlock()
				}
			}
		}(node)
	}
	wg.Wait()

	var result []engine.HistogramPoint
	for t, c := range combined {
		result = append(result, engine.HistogramPoint{Time: t, Count: int(c)})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Time < result[j].Time
	})

	return result, nil
}

// Stats performs scatter-gather stats aggregation.
func (a *Aggregator) Stats(auth string) (engine.SystemStats, error) {
	var total engine.SystemStats
	total.LevelDist = make(map[string]int)
	total.TopServices = make(map[string]int)

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, node := range a.DataNodes {
		wg.Add(1)
		go func(nodeURL string) {
			defer wg.Done()
			url := fmt.Sprintf("%s/api/stats", nodeURL)
			req, _ := http.NewRequest("GET", url, nil)
			if auth != "" {
				req.Header.Set("Authorization", auth)
			}
			resp, err := a.Client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var nodeStats engine.SystemStats
				if err := json.NewDecoder(resp.Body).Decode(&nodeStats); err == nil {
					mu.Lock()
					total.IngestionRate += nodeStats.IngestionRate
					total.TotalLogs += nodeStats.TotalLogs
					total.DiskUsage += nodeStats.DiskUsage
					for k, v := range nodeStats.LevelDist {
						total.LevelDist[k] += v
					}
					for k, v := range nodeStats.TopServices {
						total.TopServices[k] += v
					}
					mu.Unlock()
				}
			}
		}(node)
	}
	wg.Wait()

	return total, nil
}
