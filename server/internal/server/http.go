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

	"sync/atomic"

	"crypto/rand"
	"encoding/hex"
	"sync"

	"github.com/coffersTech/nanolog/server/internal/controller"
	"github.com/coffersTech/nanolog/server/internal/engine"
	"github.com/valyala/fastjson"
	"golang.org/x/crypto/bcrypt"
)

// UserSession represents a logged-in Web user session.
type UserSession struct {
	Token      string
	Username   string
	ExpireTime time.Time
}

type IngestServer struct {
	queryEngine   *engine.QueryEngine
	metaStore     *controller.Store
	webDir        string // Directory for static web files
	dataDir       string // Directory for log data
	sessions      map[string]UserSession
	sessionsMu    sync.RWMutex
	srv           *http.Server
	parser        fastjson.ParserPool
	ingestCounter int64 // Monotonic counter for total requests
	ingestRate    int64 // Requests per second (updated periodically)
}

func NewIngestServer(qe *engine.QueryEngine, ms *controller.Store, webDir string, dataDir string) *IngestServer {
	return &IngestServer{
		queryEngine: qe,
		metaStore:   ms,
		webDir:      webDir,
		dataDir:     dataDir,
		sessions:    make(map[string]UserSession),
	}
}

// Start runs the HTTP server.
func (s *IngestServer) Start(addr string) error {
	mux := http.NewServeMux()

	// API routes (protected)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/system/status", s.handleSystemStatus)
	mux.HandleFunc("/api/system/init", s.handleSystemInit)
	mux.Handle("/api/system/config", s.AuthMiddleware(http.HandlerFunc(s.handleSystemConfig)))

	// User management (SuperAdmin)
	mux.Handle("/api/users", s.AuthMiddleware(http.HandlerFunc(s.handleUsers)))
	mux.Handle("/api/users/", s.AuthMiddleware(http.HandlerFunc(s.handleUserItem)))

	// Token management (Authenticated)
	mux.Handle("/api/tokens", s.AuthMiddleware(http.HandlerFunc(s.handleTokens)))
	mux.Handle("/api/tokens/", s.AuthMiddleware(http.HandlerFunc(s.handleTokenItem)))

	mux.Handle("/api/ingest", s.AuthMiddleware(http.HandlerFunc(s.handleIngest)))
	mux.Handle("/api/search", s.AuthMiddleware(http.HandlerFunc(s.handleQuery)))
	mux.Handle("/api/histogram", s.AuthMiddleware(http.HandlerFunc(s.handleHistogram)))
	mux.Handle("/api/stats", s.AuthMiddleware(http.HandlerFunc(s.handleStats)))

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

// AuthMiddleware checks for a valid token in the Authorization header.
func (s *IngestServer) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		var token string
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			token = r.URL.Query().Get("token")
		}

		if token == "" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="NanoLog"`)
			http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
			return
		}

		// Logic Branch A: SDK / API Key (from meta)
		if apiToken, exists := s.metaStore.GetTokenByValue(token); exists {
			// Attach user info to context if needed?
			_ = apiToken
			next.ServeHTTP(w, r)
			return
		}

		// Logic Branch B: Web Session
		s.sessionsMu.RLock()
		session, exists := s.sessions[token]
		s.sessionsMu.RUnlock()

		if exists {
			if time.Now().Before(session.ExpireTime) {
				// Optional: Check if user also exists in meta (role check)
				user, exists := s.metaStore.GetUser(session.Username)
				if !exists {
					http.Error(w, "User no longer exists", http.StatusUnauthorized)
					return
				}

				// Role check for specific routes
				if strings.HasPrefix(r.URL.Path, "/api/users") {
					if user.Role != "super_admin" {
						http.Error(w, "Forbidden: SuperAdmin required", http.StatusForbidden)
						return
					}
				}

				next.ServeHTTP(w, r)
				return
			}
			s.sessionsMu.Lock()
			delete(s.sessions, token)
			s.sessionsMu.Unlock()
		}

		w.Header().Set("WWW-Authenticate", `Bearer realm="NanoLog"`)
		http.Error(w, "Unauthorized: Invalid or expired token", http.StatusUnauthorized)
	})
}

// handleSystemStatus returns the system initialization status.
func (s *IngestServer) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]bool{
		"initialized": s.metaStore.IsInitialized(),
	})
}

// handleSystemInit initializes the system with the first SuperAdmin.
func (s *IngestServer) handleSystemInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.metaStore.IsInitialized() {
		http.Error(w, "System already initialized", http.StatusBadRequest)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	if err := s.metaStore.InitializeSystem(req.Username, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.createSession(w, req.Username, "super_admin")
}

func (s *IngestServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	user, exists := s.metaStore.GetUser(req.Username)
	if !exists {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	s.createSession(w, req.Username, user.Role)
}

func (s *IngestServer) createSession(w http.ResponseWriter, username, role string) {
	b := make([]byte, 16)
	rand.Read(b)
	sessionToken := hex.EncodeToString(b)

	s.sessionsMu.Lock()
	s.sessions[sessionToken] = UserSession{
		Token:      sessionToken,
		Username:   username,
		ExpireTime: time.Now().Add(24 * time.Hour),
	}
	s.sessionsMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":    sessionToken,
		"username": username,
		"role":     role,
	})
}

func (s *IngestServer) handleSystemConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := s.metaStore.GetData()
		json.NewEncoder(w).Encode(data.Config)
		return
	}

	if r.Method == http.MethodPost {
		var cfg controller.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate retention duration
		if _, err := time.ParseDuration(cfg.Retention); err != nil {
			http.Error(w, "Invalid retention duration format", http.StatusBadRequest)
			return
		}

		if err := s.metaStore.UpdateConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Note: Ideally we would trigger an update in QueryEngine too.
		// For now, it will take effect on next restart or we can pass it via reference.
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (s *IngestServer) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := s.metaStore.GetData()
		// Strip hashes for security
		users := make([]controller.User, len(data.Users))
		for i, u := range data.Users {
			users[i] = u
			users[i].PasswordHash = ""
		}
		json.NewEncoder(w).Encode(users)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		err := s.metaStore.AddUser(controller.User{
			Username:     req.Username,
			PasswordHash: string(hash),
			Role:         req.Role,
			CreatedAt:    time.Now().Unix(),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	}
}

func (s *IngestServer) handleUserItem(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	username := parts[len(parts)-1]

	if r.Method == http.MethodDelete {
		if err := s.metaStore.DeleteUser(username); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
}

func (s *IngestServer) handleTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := s.metaStore.GetData()
		json.NewEncoder(w).Encode(data.Tokens)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		b := make([]byte, 16)
		rand.Read(b)
		tokenVal := "sk-" + hex.EncodeToString(b)

		idBytes := make([]byte, 8)
		rand.Read(idBytes)
		id := hex.EncodeToString(idBytes)

		// Get creator from session
		authHeader := r.Header.Get("Authorization")
		sessionToken := strings.TrimPrefix(authHeader, "Bearer ")
		s.sessionsMu.RLock()
		session := s.sessions[sessionToken]
		s.sessionsMu.RUnlock()

		err := s.metaStore.AddToken(controller.APIToken{
			ID:        id,
			Name:      req.Name,
			Token:     tokenVal,
			Type:      req.Type,
			CreatedBy: session.Username,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"token": tokenVal, "id": id})
		return
	}
}

func (s *IngestServer) handleTokenItem(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	id := parts[len(parts)-1]

	if r.Method == http.MethodDelete {
		if err := s.metaStore.DeleteToken(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
}

// handleIngest processes POST requests with JSON logs.
func (s *IngestServer) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Log request entry
	// log.Printf("Incoming request from %s", r.RemoteAddr) // Reduce noise
	atomic.AddInt64(&s.ingestCounter, 1)

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

		s.queryEngine.Ingest(tsVal, levelStr, serviceStr, hostStr, msg)
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

	// 4. Batch Sync WAL to disk once per request for high performance
	s.queryEngine.SyncWAL()

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

func (s *IngestServer) handleHistogram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	startStr := q.Get("start")
	endStr := q.Get("end")
	intervalStr := q.Get("interval") // User provides seconds? Let's check logic.

	// Defaults
	end := time.Now().UnixNano()
	start := end - (1 * time.Hour).Nanoseconds() // Default last 1h
	var interval int64 = 60 * 1_000_000_000      // Default 1 min (in nanos)

	if startStr != "" {
		if val, err := strconv.ParseInt(startStr, 10, 64); err == nil {
			start = val * 1_000_000 // Convert ms to nanos
		}
	}
	if endStr != "" {
		if val, err := strconv.ParseInt(endStr, 10, 64); err == nil {
			end = val * 1_000_000 // Convert ms to nanos
		}
	}
	if intervalStr != "" {
		if val, err := strconv.ParseInt(intervalStr, 10, 64); err == nil {
			interval = val * 1_000_000_000 // Convert seconds to nanos
		}
	}

	filter := engine.Filter{
		MinTime: start,
		MaxTime: end,
		Service: q.Get("service"),
		Host:    q.Get("host"),
		Query:   q.Get("q"),
	}

	if lvlStr := q.Get("level"); lvlStr != "" {
		if lvl, err := strconv.Atoi(lvlStr); err == nil {
			filter.Level = uint8(lvl)
		}
	}

	// Computes
	points, err := s.queryEngine.ComputeHistogram(start, end, interval, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(points); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}

// handleStats calculates system statistics.
func (s *IngestServer) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.queryEngine.GetStats()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}
