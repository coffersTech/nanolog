package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coffersTech/nanolog/server/internal/engine"
	"github.com/coffersTech/nanolog/server/internal/server"
	"github.com/coffersTech/nanolog/server/internal/storage"
)

func main() {
	// Command-line flags
	port := flag.Int("port", 8088, "HTTP port to listen on")
	retentionStr := flag.String("retention", "168h", "Data retention duration (e.g. 72h, 7d)")
	dataDir := flag.String("data", "../data", "Directory to store .nano files")
	webDir := flag.String("web", "../web", "Directory for static web files")
	flag.Parse()

	// Parse retention duration
	retention, err := time.ParseDuration(*retentionStr)
	if err != nil {
		log.Fatalf("Invalid retention duration: %v", err)
	}

	log.Println("NanoLog Kernel v0.1 Started...")

	// 1. Initialize global MemTable
	mt := engine.NewMemTable()
	mt.StartStatsTicker(1 * time.Second)
	log.Printf("MemTable initialized. Capacity: %d rows", 4096)

	// 2. Initialize QueryEngine with retention
	reader, err := storage.NewColumnReader()
	if err != nil {
		log.Fatalf("Failed to create reader: %v", err)
	}
	writer, err := storage.NewColumnWriter()
	if err != nil {
		log.Fatalf("Failed to create writer: %v", err)
	}
	qe := engine.NewQueryEngine(*dataDir, mt, reader.ReadSnapshot, writer.WriteSnapshot, retention)
	log.Printf("QueryEngine initialized. Data: %s, Retention: %v", *dataDir, retention)

	// Start Background Cleaner
	go qe.RunCleaner(1 * time.Hour)

	// 3. Initialize IngestServer with web directory
	srv := server.NewIngestServer(mt, qe, *webDir, *dataDir)
	addr := fmt.Sprintf(":%d", *port)

	// 4. Start HTTP Server in a goroutine
	go func() {
		log.Printf("Listening on %s", addr)
		log.Printf("Dashboard available at http://localhost%s", addr)
		if err := srv.Start(addr); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	// 4. Graceful Shutdown Hook
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal
	sig := <-quit
	log.Printf("Received signal: %v. Shutting down...", sig)

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Flushing memory to disk...")
	if err := qe.Flush(); err != nil {
		log.Printf("Final flush failed: %v", err)
	}

	log.Println("NanoLog exited gracefully.")
}
