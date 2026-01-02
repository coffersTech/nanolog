package nanolog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
)

type HandshakeRequest struct {
	InstanceID  string `json:"instance_id"`
	ServiceName string `json:"service_name"`
	HostName    string `json:"host_name"`
	Platform    string `json:"platform"`
	Version     string `json:"version"`
}

type HandshakeResponse struct {
	Status string `json:"status"`
}

func ensureInstanceID() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return uuid.New().String(), nil // Fallback to ephemeral ID
	}

	nanoDir := filepath.Join(homeDir, ".nanolog")
	if err := os.MkdirAll(nanoDir, 0755); err != nil {
		return uuid.New().String(), nil
	}

	idFile := filepath.Join(nanoDir, "id")
	if data, err := os.ReadFile(idFile); err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	newID := uuid.New().String()
	_ = os.WriteFile(idFile, []byte(newID), 0644)
	return newID, nil
}

func registerInstance(url, apiKey, service, instanceID string) error {
	hostname, _ := os.Hostname()
	reqBody := HandshakeRequest{
		InstanceID:  instanceID,
		ServiceName: service,
		HostName:    hostname,
		Platform:    fmt.Sprintf("go-%s", runtime.Version()),
		Version:     "0.1.0",
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", strings.TrimRight(url, "/")+"/api/registry/handshake", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("handshake failed: %d %s", resp.StatusCode, string(body))
	}

	return nil
}
