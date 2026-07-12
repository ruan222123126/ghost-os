package mobile

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/internal/stringutil"
)

const secretBytes = 32

type DeviceCredential struct {
	DeviceID        string     `json:"device_id"`
	Label           string     `json:"label"`
	Secret          string     `json:"secret"`
	PairedAt        time.Time  `json:"paired_at"`
	Revoked         bool       `json:"revoked"`
	LastConnectedAt *time.Time `json:"last_connected_at,omitempty"`
}

type PairingOptions struct {
	DeviceID       string
	Label          string
	PCID           string
	SignalingURL   string
	SignalingToken string
	ICEServersJSON string
	Now            time.Time
}

type PairingResult struct {
	Device DeviceCredential
	URI    string
}

type CredentialStore struct {
	path string
	mu   sync.Mutex
}

func NewCredentialStore(path string) *CredentialStore {
	return &CredentialStore{path: strings.TrimSpace(path)}
}

func (s *CredentialStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *CredentialStore) List() ([]DeviceCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *CredentialStore) Pair(options PairingOptions) (PairingResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	devices, err := s.loadLocked()
	if err != nil {
		return PairingResult{}, err
	}
	device, err := newDeviceCredential(options)
	if err != nil {
		return PairingResult{}, err
	}
	devices = upsertDevice(devices, device)
	if err := s.saveLocked(devices); err != nil {
		return PairingResult{}, err
	}
	uri, err := buildPairingURI(device, options)
	if err != nil {
		return PairingResult{}, err
	}
	return PairingResult{Device: device, URI: uri}, nil
}

func (s *CredentialStore) Revoke(deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(deviceID)
	if id == "" {
		return errors.New("device_id is required")
	}
	devices, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range devices {
		if devices[i].DeviceID == id {
			devices[i].Revoked = true
			return s.saveLocked(devices)
		}
	}
	return fmt.Errorf("paired device not found: %s", id)
}

func (s *CredentialStore) SecretFor(deviceID string) ([]byte, DeviceCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(deviceID)
	if id == "" {
		return nil, DeviceCredential{}, errors.New("device_id is required")
	}
	devices, err := s.loadLocked()
	if err != nil {
		return nil, DeviceCredential{}, err
	}
	for _, device := range devices {
		if device.DeviceID != id {
			continue
		}
		if device.Revoked {
			return nil, DeviceCredential{}, fmt.Errorf("paired device is revoked: %s", id)
		}
		secret, err := DecodeSecret(device.Secret)
		if err != nil {
			return nil, DeviceCredential{}, fmt.Errorf("decode device secret: %w", err)
		}
		return secret, device, nil
	}
	return nil, DeviceCredential{}, fmt.Errorf("paired device not found: %s", id)
}

func (s *CredentialStore) MarkConnected(deviceID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	devices, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range devices {
		if devices[i].DeviceID == strings.TrimSpace(deviceID) {
			value := at.UTC()
			devices[i].LastConnectedAt = &value
			return s.saveLocked(devices)
		}
	}
	return nil
}

func (s *CredentialStore) loadLocked() ([]DeviceCredential, error) {
	if s == nil || s.path == "" {
		return nil, errors.New("credential store path is required")
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read credential store: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var devices []DeviceCredential
	if err := json.Unmarshal(data, &devices); err != nil {
		return nil, fmt.Errorf("decode credential store: %w", err)
	}
	return devices, nil
}

func (s *CredentialStore) saveLocked(devices []DeviceCredential) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create credential store directory: %w", err)
	}
	data, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credential store: %w", err)
	}
	if err := os.WriteFile(s.path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write credential store: %w", err)
	}
	return nil
}

func newDeviceCredential(options PairingOptions) (DeviceCredential, error) {
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	id := strings.TrimSpace(options.DeviceID)
	if id == "" {
		var err error
		id, err = randomToken(16)
		if err != nil {
			return DeviceCredential{}, err
		}
	}
	secret, err := randomToken(secretBytes)
	if err != nil {
		return DeviceCredential{}, err
	}
	return DeviceCredential{
		DeviceID: id,
		Label:    stringutil.FirstNonEmpty(options.Label, "mobile"),
		Secret:   secret,
		PairedAt: now.UTC(),
	}, nil
}

func upsertDevice(devices []DeviceCredential, next DeviceCredential) []DeviceCredential {
	for i := range devices {
		if devices[i].DeviceID == next.DeviceID {
			devices[i] = next
			return devices
		}
	}
	return append(devices, next)
}

func buildPairingURI(device DeviceCredential, options PairingOptions) (string, error) {
	if strings.TrimSpace(options.PCID) == "" {
		return "", errors.New("pc_id is required")
	}
	if strings.TrimSpace(options.SignalingURL) == "" {
		return "", errors.New("signaling_url is required")
	}
	values := url.Values{}
	values.Set("device_id", device.DeviceID)
	values.Set("secret", device.Secret)
	values.Set("pc_id", strings.TrimSpace(options.PCID))
	values.Set("signaling_url", strings.TrimSpace(options.SignalingURL))
	if token := strings.TrimSpace(options.SignalingToken); token != "" {
		values.Set("signaling_token", token)
	}
	if ice := strings.TrimSpace(options.ICEServersJSON); ice != "" {
		values.Set("ice_servers_json", ice)
	}
	return (&url.URL{Scheme: "ghost-os", Host: "mobile-pair", RawQuery: values.Encode()}).String(), nil
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func DecodeSecret(value string) ([]byte, error) {
	secret, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	if len(secret) < secretBytes {
		return nil, fmt.Errorf("secret must be at least %d bytes", secretBytes)
	}
	return secret, nil
}
