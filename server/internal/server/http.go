package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coffersTech/nanolog/server/internal/engine"
	"github.com/valyala/fastjson"
)

// IngestServer holds the HTTP server execution dependencies.
type IngestServer struct {
	mt          *engine.MemTable
	queryEngine *engine.QueryEngine
	webDir      string // Directory for static web files
	srv         *http.Server
	parser      fastjson.ParserPool // Pool of parsers to reduce allocations
}

func NewIngestServer(mt *engine.MemTable, qe *engine.QueryEngine, webDir string) *IngestServer {
	return &IngestServer{
		mt:          mt,
		queryEngine: qe,
		webDir:      webDir,
	}
}

// Start runs the HTTP server.
func (s *IngestServer) Start(addr string) error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/ingest", s.handleIngest)
	mux.HandleFunc("/api/search", s.handleQuery)

	// Static file serving for web directory
	if s.webDir != "" {
		fs := http.FileServer(http.Dir(s.webDir))
		mux.Handle("/", fs)
	}

	s.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *IngestServer) Shutdown(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

// handleIngest processes POST requests with JSON logs.
func (s *IngestServer) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Log request entry
	log.Printf("Incoming request from %s", r.RemoteAddr)

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Log body content (debug only)
	log.Printf("Request Body: %s", string(body))

	// Parse
	p := s.parser.Get()
	defer s.parser.Put(p)

	v, err := p.ParseBytes(body)
	if err != nil {
		log.Printf("JSON Parse Error: %v. Body: %s", err, string(body))
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Helper function to process a single log object
	processLog := func(val *fastjson.Value) {
		tsVal := val.GetInt64("timestamp")
		if tsVal == 0 {
			tsVal = time.Now().UnixNano()
		}

		levelStr := string(val.GetStringBytes("level"))

		serviceStr := string(val.GetStringBytes("service"))
		if serviceStr == "" {
			serviceStr = "default"
		}

		hostStr := string(val.GetStringBytes("host"))
		if hostStr == "" {
			// Fallback: Use IP from connection (strip port)
			hostStr = r.RemoteAddr
			if idx := strings.LastIndex(hostStr, ":"); idx != -1 {
				hostStr = hostStr[:idx]
			}
		}

		msg := string(val.GetStringBytes("message"))
		if msg == "" {
			msg = string(val.GetStringBytes("msg"))
		}

		s.mt.Append(tsVal, levelStr, serviceStr, hostStr, msg)
	}

	// Handle batch (Array) or single (Object)
	if v.Type() == fastjson.TypeArray {
		arr, _ := v.Array()
		for _, val := range arr {
			processLog(val)
		}
	} else {
		processLog(v)
	}

	// Auto-flush logic
	if s.mt.Len() >= 10 {
		if err := s.queryEngine.Flush(); err != nil {
			log.Printf("Auto-flush failed: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleQuery processes GET /api/search requests.
func (s *IngestServer) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse filter parameters
	filter := engine.Filter{}

	// Support both min_ts/max_ts and start/end aliases
	minTsStr := r.URL.Query().Get("min_ts")
	if minTsStr == "" {
		minTsStr = r.URL.Query().Get("start")
	}
	if minTsStr != "" {
		if val, err := strconv.ParseInt(minTsStr, 10, 64); err == nil {
			filter.MinTime = val
		}
	}

	maxTsStr := r.URL.Query().Get("max_ts")
	if maxTsStr == "" {
		maxTsStr = r.URL.Query().Get("end")
	}
	if maxTsStr != "" {
		if val, err := strconv.ParseInt(maxTsStr, 10, 64); err == nil {
			filter.MaxTime = val
		}
	}

	if levelStr := r.URL.Query().Get("level"); levelStr != "" {
		if val, err := strconv.Atoi(levelStr); err == nil {
			filter.Level = uint8(val)
		}
	}
	filter.Service = r.URL.Query().Get("service")
	filter.Host = r.URL.Query().Get("host")
	filter.Query = r.URL.Query().Get("q")

	// Parse limit parameter (default 100)
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Execute scan
	rows, err := s.queryEngine.ExecuteScan(filter, limit)
	if err != nil {
		log.Printf("Query error: %v", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rows); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}
