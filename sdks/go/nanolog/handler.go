package nanolog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Options struct {
	ServerURL  string
	APIKey     string
	Service    string
	SourceHost string
}

type NanoHandler struct {
	opts       Options
	instanceID string
	queue      chan []byte
	done       chan struct{}
	wg         sync.WaitGroup
	attrs      []slog.Attr
	groups     []string
}

type LogRow struct {
	Timestamp  int64                  `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Logger     string                 `json:"logger,omitempty"`
	Thread     string                 `json:"thread,omitempty"`
	File       string                 `json:"file,omitempty"`
	Line       int                    `json:"line,omitempty"`
	Service    string                 `json:"service"`
	Host       string                 `json:"host"`
	InstanceID string                 `json:"instance_id"`
	Attributes map[string]interface{} `json:"attributes"`
}

func NewHandler(opts Options) *NanoHandler {
	id, _ := ensureInstanceID()
	h := &NanoHandler{
		opts:       opts,
		instanceID: id,
		queue:      make(chan []byte, 10000),
		done:       make(chan struct{}),
	}

	if h.opts.SourceHost == "" {
		h.opts.SourceHost, _ = os.Hostname()
	}

	// Register asynchronously to not block startup
	go func() {
		if err := registerInstance(opts.ServerURL, opts.APIKey, opts.Service, h.instanceID); err != nil {
			fmt.Fprintf(os.Stderr, "NanoLog Handshake Failed: %v\n", err)
		}
	}()

	h.wg.Add(1)
	go h.runLoop()

	return h
}

func (h *NanoHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *NanoHandler) Handle(ctx context.Context, r slog.Record) error {
	row := LogRow{
		Timestamp:  r.Time.UnixNano(),
		Level:      r.Level.String(),
		Message:    r.Message,
		Service:    h.opts.Service,
		Host:       h.opts.SourceHost,
		InstanceID: h.instanceID,
		Attributes: make(map[string]interface{}),
	}

	// Add basic source info
	fs := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := fs.Next()
	row.File = f.File
	row.Line = f.Line
	row.Thread = "goroutine" // Go doesn't expose thread names easily

	// Add attributes
	// 1. Stored attributes (from WithAttrs)
	for _, a := range h.attrs {
		row.Attributes[a.Key] = a.Value.Any()
	}
	// 2. Record attributes
	r.Attrs(func(a slog.Attr) bool {
		row.Attributes[a.Key] = a.Value.Any()
		return true
	})

	data, err := json.Marshal(row)
	if err != nil {
		return err
	}

	select {
	case h.queue <- data:
	default:
		// Drop logging
		fmt.Fprintf(os.Stderr, "NanoLog Queue Full: Dropping log\n")
	}

	return nil
}

func (h *NanoHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	h2.attrs = append(h2.attrs, attrs...)
	return &h2
}

func (h *NanoHandler) WithGroup(name string) slog.Handler {
	h2 := *h
	h2.groups = append(h2.groups, name)
	return &h2
}

func (h *NanoHandler) runLoop() {
	defer h.wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var batch [][]byte

	send := func() {
		if len(batch) == 0 {
			return
		}
		
		// Encode as JSON Array: [ {}, {}, {} ]
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i, b := range batch {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.Write(b)
		}
		buf.WriteByte(']')

		req, err := http.NewRequest("POST", strings.TrimRight(h.opts.ServerURL, "/")+"/api/ingest/batch", &buf)
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+h.opts.APIKey)
			req.Header.Set("X-Instance-ID", h.instanceID)
			
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "NanoLog Network Error: %v\n", err)
			} else {
				resp.Body.Close()
				if resp.StatusCode != 200 {
					fmt.Fprintf(os.Stderr, "NanoLog Send Failed: HTTP %d\n", resp.StatusCode)
				}
			}
		}

		batch = nil // Reset batch
	}

	for {
		select {
		case data := <-h.queue:
			batch = append(batch, data)
			if len(batch) >= 100 {
				send()
			}
		case <-ticker.C:
			send()
		case <-h.done:
			// Flush remaining
			for {
				select {
				case data := <-h.queue:
					batch = append(batch, data)
				default:
					send()
					return
				}
			}
		}
	}
}

func (h *NanoHandler) Shutdown() {
	close(h.done)
	h.wg.Wait()
}
