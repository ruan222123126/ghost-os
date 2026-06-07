package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFailsFastOnInvalidFileHeaders(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "provider headers",
			content: "provider_headers = { \"   \" = \"tenant-1\" }\n",
			want:    "invalid provider_headers: header key cannot be empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(configPath, []byte(tc.content), 0o600); err != nil {
				t.Fatalf("write config file: %v", err)
			}
			t.Setenv("GHOST_CONFIG_PATH", configPath)
			t.Setenv("GHOST_PROVIDER", "custom")

			_, _, err := Load()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestWriteBridgeFileConfigFailsFastOnInvalidHeaders(t *testing.T) {
	cases := []struct {
		name string
		cfg  FileConfig
		want string
	}{
		{
			name: "provider headers",
			cfg: FileConfig{
				ProviderHeaders: map[string]string{"   ": "tenant-1"},
			},
			want: "invalid provider_headers: header key cannot be empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Write(filepath.Join(t.TempDir(), "config.toml"), tc.cfg)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}
