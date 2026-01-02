package main

import (
	"log/slog"
	"time"

	"github.com/coffersTech/nanolog/sdks/go/nanolog"
)

func main() {
	opts := nanolog.Options{
		ServerURL: "http://localhost:8088",
		APIKey:    "sk-dev-test-key",
		Service:   "go-example-service",
	}
	handler := nanolog.NewHandler(opts)
	defer handler.Shutdown()
	logger := slog.New(handler)

	logger.Info("Hello from Go SDK", "user_id", 42, "status", "active")
	logger.Warn("This is a warning", "retry_count", 3)
	logger.Error("Something went wrong", "error", "connection refused")

	// Simulate work to allow async sender to pick up logs
	time.Sleep(2 * time.Second)
	
	logger.Info("Last message before exit")
}
