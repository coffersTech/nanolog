package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coffersTech/nanolog/server/internal/cluster"
	"github.com/coffersTech/nanolog/server/internal/controller"
	"github.com/coffersTech/nanolog/server/internal/engine"
	"github.com/coffersTech/nanolog/server/internal/pkg/security"
	"github.com/coffersTech/nanolog/server/internal/registry"
	"github.com/coffersTech/nanolog/server/internal/server"
	"github.com/coffersTech/nanolog/server/internal/storage"
)

func main() {
	// Command-line flags
	port := flag.Int("port", 8088, "HTTP port to listen on")
	retentionStr := flag.String("retention", "168h", "Data retention duration (e.g. 72h, 7d)")
	dataDir := flag.String("data", "../data", "Directory to store .nano files")
	webDir := flag.String("web", "../web", "Directory for static web files")
	keyPath := flag.String("key", "", "Path to the master key file (defaults to <data>/.nanolog.key)")
	role := flag.String("role", "standalone", "Server role: standalone, console, ingester")
	dataNodes := flag.String("data-nodes", "", "Comma-separated list of data node URLs (for console role)")
	adminAddr := flag.String("admin-addr", "localhost:8080", "Upstream admin address (for ingester nodes)")
	flag.Parse()

	// Parse retention duration
	retention, err := time.ParseDuration(*retentionStr)
	if err != nil {
		log.Fatalf("Invalid retention duration: %v", err)
	}

	var dataNodeList []string
	if *dataNodes != "" {
		dataNodeList = strings.Split(*dataNodes, ",")
		for i := range dataNodeList {
			dataNodeList[i] = strings.TrimSpace(dataNodeList[i])
		}
	}

	var metaStore *controller.Store
	var qe *engine.QueryEngine

	// 0. Initialize Security & Metadata (Standalone or Console)
	if *role == "standalone" || *role == "console" {
		realKeyPath := *keyPath
		if realKeyPath == "" {
			realKeyPath = fmt.Sprintf("%s/.nanolog.key", *dataDir)
		}

		generated, err := security.InitMasterKey(realKeyPath)
		if err != nil {
			log.Fatalf("Failed to initialize security layer: %v", err)
		}

		if generated {
			fmt.Println("┌────────────────────────────────────────────────────────────────────────┐")
			fmt.Println("│                                WARNING                                 │")
			fmt.Println("├────────────────────────────────────────────────────────────────────────┤")
			fmt.Printf("│ A new Master Key has been generated at %-31s │\n", realKeyPath)
			fmt.Println("│ Please back up this file! Without it, you cannot recover your data.    │")
			fmt.Println("└────────────────────────────────────────────────────────────────────────┘")
		}

		// Initialize Metadata Store
		metaStore = controller.NewStore(fmt.Sprintf("%s/.nanolog.sys", *dataDir))
		if err := metaStore.Load(); err != nil {
			log.Fatalf("Failed to load systems metadata: %v", err)
		}

		if *role == "console" {
			log.Println("NanoLog Console Node Started (Management & Query Aggregation)")
		} else {
			log.Println("NanoLog Standalone Node Started")
		}
	}

	// 1. Initialize Engine (Standalone or Ingester)
	if *role == "standalone" || *role == "ingester" {
		// Initialize global MemTable
		mt := engine.NewMemTable()
		mt.StartStatsTicker(1 * time.Second)
		log.Printf("MemTable initialized. Capacity: %d rows", 4096)

		// Initialize QueryEngine with retention
		reader, err := storage.NewColumnReader()
		if err != nil {
			log.Fatalf("Failed to create reader: %v", err)
		}
		writer, err := storage.NewColumnWriter()
		if err != nil {
			log.Fatalf("Failed to create writer: %v", err)
		}
		qe = engine.NewQueryEngine(*dataDir, mt, reader.ReadSnapshot, writer.WriteSnapshot, retention)
		log.Printf("QueryEngine initialized. Data: %s, Retention: %v", *dataDir, retention)

		// Start Background Cleaner
		go qe.RunCleaner(1 * time.Hour)

		if *role == "ingester" {
			log.Printf("NanoLog Ingester Node Started (Storage & Local Ingest)")
		}
	}

	log.Println("NanoLog Kernel v0.1 Starting HTTP Server...")

	// 4. Initialize Aggregator for Console role
	aggregator := cluster.NewAggregator(dataNodeList)
	
	// 5. Initialize Registry (Standalone or Console)
	var regStore *registry.Store
	if *role == "standalone" || *role == "console" {
		regStore = registry.NewStore()
		// Cleanup stale instances every minute (stale = 10 mins inactive)
		regStore.StartCleanupLoop(context.Background(), 1*time.Minute, 10*time.Minute)
	}

	// 6. Initialize IngestServer
	srv := server.NewIngestServer(qe, metaStore, *webDir, *dataDir, *role, aggregator, regStore)
	
	// Register Handshake Route manually for now (since IngestServer doesn't encapsulate it yet)
	// Ideally IngestServer should accept additional handlers or we register it to srv's mux?
	// But srv.Start creates a NEW mux. We need to register it INSIDE IngestServer or pass it?
	// Wait, IngestServer.Start creates the mux locally. I can't register external handlers easily unless I modify IngestServer to expose registration or do it inside IngestServer.
	// For now, let's keep it simple: Use IngestServer to handle registry? No, separation of concerns.
	// We need to modify IngestServer to Register Registry routes.
	
	addr := fmt.Sprintf(":%d", *port)
	_ = adminAddr 

	// 4. Start HTTP Server in a goroutine
	go func() {
		log.Printf("Listening on %s", addr)
		if *role == "standalone" || *role == "console" {
			log.Printf("Web Console available at http://localhost%s", addr)
		}
		if err := srv.Start(addr, *role); err != nil {
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

	if qe != nil {
		log.Println("Flushing memory to disk...")
		if err := qe.Flush(); err != nil {
			log.Printf("Final flush failed: %v", err)
		}
	}

	log.Println("NanoLog exited gracefully.")
}
