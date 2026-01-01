package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStore_Cleanup(t *testing.T) {
	s := NewStore()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add an instance
	inst := Instance{
		InstanceID: "test-1",
	}
	s.RegisterOrUpdate(inst)
	
	// Manually set LastSeenAt to be stale (bypassing RegisterOrUpdate's overwrite)
	s.mu.Lock()
	if i, ok := s.instances["test-1"]; ok {
		i.LastSeenAt = time.Now().Add(-20 * time.Minute).Unix()
	}
	s.mu.Unlock()

	// Add a fresh instance
	inst2 := Instance{
		InstanceID: "test-2",
	}
	s.RegisterOrUpdate(inst2)

	// Start cleanup loop with quick interval
	s.StartCleanupLoop(ctx, 10*time.Millisecond, 10*time.Minute)

	time.Sleep(50 * time.Millisecond)

	// Check results
	if _, ok := s.GetInstance("test-1"); ok {
		t.Error("test-1 should have been pruned")
	}
	if _, ok := s.GetInstance("test-2"); !ok {
		t.Error("test-2 should still exist")
	}
}

func TestServer_HandleHandshake(t *testing.T) {
	store := NewStore()
	server := NewServer(store)

	body := `{"instance_id":"sdk-123", "service_name":"my-service", "sdk_version":"1.0"}`
	req := httptest.NewRequest("POST", "/api/registry/handshake", strings.NewReader(body))
	w := httptest.NewRecorder()

	server.HandleHandshake(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if _, ok := store.GetInstance("sdk-123"); !ok {
		t.Error("Instance should be registered")
	}
}
