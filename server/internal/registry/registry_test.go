package registry

import (
	"context"
	"github.com/coffersTech/nanolog/server/internal/models"
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
	inst := models.Instance{
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
	inst2 := models.Instance{
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
	server := NewServer(store, nil)

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

type mockMetaStore struct {
	devices []models.Instance
}

func (m *mockMetaStore) AddOrUpdateDevice(device models.Instance) error {
	for i, d := range m.devices {
		if d.InstanceID == device.InstanceID {
			m.devices[i] = device
			return nil
		}
	}
	for i, d := range m.devices {
		if (time.Now().Unix()-d.LastSeenAt) > 60 && d.ServiceName == device.ServiceName {
			if (device.IP != "" && d.IP == device.IP) || (device.Hostname != "" && d.Hostname == device.Hostname) {
				m.devices[i] = device
				return nil
			}
		}
	}
	m.devices = append(m.devices, device)
	return nil
}

func (m *mockMetaStore) DeleteDevice(id string) error {
	for i, d := range m.devices {
		if d.InstanceID == id {
			m.devices = append(m.devices[:i], m.devices[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockMetaStore) GetDevices() []models.Instance {
	return m.devices
}

func (m *mockMetaStore) ClearOfflineDevices(timeoutSec int64) (int, error) {
	now := time.Now().Unix()
	active := make([]models.Instance, 0)
	removed := 0
	for _, d := range m.devices {
		if now-d.LastSeenAt > timeoutSec {
			removed++
		} else {
			active = append(active, d)
		}
	}
	m.devices = active
	return removed, nil
}

func TestServer_ClearOfflineDevices(t *testing.T) {
	mock := &mockMetaStore{
		devices: []models.Instance{
			{InstanceID: "online-1", ServiceName: "svc-1", LastSeenAt: time.Now().Unix()},
			{InstanceID: "offline-1", ServiceName: "svc-2", LastSeenAt: time.Now().Unix() - 100},
		},
	}
	store := NewStore()
	server := NewServer(store, mock)

	req := httptest.NewRequest("DELETE", "/api/registry/devices/offline", nil)
	w := httptest.NewRecorder()

	server.HandleClearOfflineDevices(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}

	if len(mock.devices) != 1 {
		t.Errorf("Expected 1 device remaining, got %d", len(mock.devices))
	}
	if mock.devices[0].InstanceID != "online-1" {
		t.Errorf("Expected online-1 remaining, got %s", mock.devices[0].InstanceID)
	}
}

