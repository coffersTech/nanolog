package models

// Instance represents a registered SDK instance.
type Instance struct {
	InstanceID   string `json:"instance_id"`
	ServiceName  string `json:"service_name"`
	Hostname     string `json:"hostname"`
	IP           string `json:"ip"`
	SdkVersion   string `json:"sdk_version"`
	Language     string `json:"language"`
	RegisteredAt int64  `json:"registered_at"`
	LastSeenAt   int64  `json:"last_seen_at"`
}

// ConfigResponse represents the dynamic configuration sent back to the SDK.
type ConfigResponse struct {
	Level      string `json:"level"`       // "INFO", "DEBUG"
	SampleRate int    `json:"sample_rate"` // 0-100
}
