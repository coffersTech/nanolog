package controller

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coffersTech/nanolog/server/internal/pkg/security"
	"golang.org/x/crypto/bcrypt"
)

// User represents a system user profile.
type User struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"` // bcrypt hashed
	Role         string `json:"role"`          // "super_admin", "admin", "viewer"
	CreatedAt    int64  `json:"created_at"`
}

// APIToken represents a machine-to-machine access key.
type APIToken struct {
	ID        string `json:"id"`    // UUID
	Name      string `json:"name"`  // e.g. "OrderService SDK"
	Token     string `json:"token"` // Sk-xxxxxx
	Type      string `json:"type"`  // "write" (SDK), "read" (Grafana)
	CreatedBy string `json:"created_by"`
}

// Config holds system-wide settings.
type Config struct {
	Retention string `json:"retention"` // e.g. "168h"
}

// MetaData is the top-level container for system metadata.
type MetaData struct {
	Initialized bool       `json:"initialized"`
	Users       []User     `json:"users"`
	Tokens      []APIToken `json:"tokens"`
	Config      Config     `json:"config"`
}

// Store handles the persistence and in-memory management of MetaData.
type Store struct {
	filePath string
	mu       sync.RWMutex
	data     *MetaData
}

// NewStore creates a new metadata store.
func NewStore(filePath string) *Store {
	return &Store{
		filePath: filePath,
		data: &MetaData{
			Users:  make([]User, 0),
			Tokens: make([]APIToken, 0),
			Config: Config{Retention: "168h"},
		},
	}
}

// Load reads metadata from disk.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		s.data.Initialized = false
		return nil
	}

	encryptedData, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	if len(encryptedData) == 0 {
		return nil
	}

	// Hardened decryption: no fallback to plain JSON
	decrypted, err := security.Decrypt(encryptedData)
	if err != nil {
		return errors.New("failed to decrypt metadata (invalid key or corrupted file): " + err.Error())
	}

	return json.Unmarshal(decrypted, s.data)
}

// Save writes metadata to disk.
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveLocked()
}

// saveLocked writes metadata to disk with encryption.
func (s *Store) saveLocked() error {
	jsonData, err := json.Marshal(s.data)
	if err != nil {
		return err
	}

	encrypted, err := security.Encrypt(jsonData)
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, encrypted, 0600)
}

// GetData returns a copy of the current metadata.
func (s *Store) GetData() MetaData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.data
}

// IsInitialized returns the initialization status.
func (s *Store) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Initialized
}

// InitializeSystem creates the first super_admin user.
func (s *Store) InitializeSystem(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data.Initialized {
		return os.ErrExist
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	s.data.Users = append(s.data.Users, User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         "super_admin",
		CreatedAt:    time.Now().Unix(),
	})
	s.data.Initialized = true

	return s.saveLocked()
}

// AddUser adds a new user to the system.
func (s *Store) AddUser(u User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.data.Users {
		if existing.Username == u.Username {
			return os.ErrExist
		}
	}

	s.data.Users = append(s.data.Users, u)
	return s.saveLocked()
}

// DeleteUser removes a user by username.
func (s *Store) DeleteUser(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, u := range s.data.Users {
		if u.Username == username {
			s.data.Users = append(s.data.Users[:i], s.data.Users[i+1:]...)
			return s.saveLocked()
		}
	}
	return os.ErrNotExist
}

// GetUser returns a user by username (case-insensitive).
func (s *Store) GetUser(username string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.data.Users {
		if strings.EqualFold(u.Username, username) {
			return u, true
		}
	}
	return User{}, false
}

// AddToken adds a new API token.
func (s *Store) AddToken(t APIToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Tokens = append(s.data.Tokens, t)
	return s.saveLocked()
}

// DeleteToken removes a token by ID.
func (s *Store) DeleteToken(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.data.Tokens {
		if t.ID == id {
			s.data.Tokens = append(s.data.Tokens[:i], s.data.Tokens[i+1:]...)
			return s.saveLocked()
		}
	}
	return os.ErrNotExist
}

// GetTokenByValue finds a token by its secret value.
func (s *Store) GetTokenByValue(val string) (APIToken, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.data.Tokens {
		if t.Token == val {
			return t, true
		}
	}
	return APIToken{}, false
}

// UpdateConfig updates system configuration.
func (s *Store) UpdateConfig(cfg Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Config = cfg
	return s.saveLocked()
}

// UpdateUserPassword updates the password hash for a user.
func (s *Store) UpdateUserPassword(username, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, u := range s.data.Users {
		if u.Username == username {
			s.data.Users[i].PasswordHash = passwordHash
			return s.saveLocked()
		}
	}
	return os.ErrNotExist
}
