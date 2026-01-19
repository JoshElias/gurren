package sshconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kevinburke/ssh_config"
)

func TestResolve_FromSSHConfig(t *testing.T) {
	// Create a temporary SSH config file for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	configContent := `
Host test-server
    HostName 192.168.1.100
    User testuser
    Port 2222
    IdentityFile ~/.ssh/test_key

Host bastion-*
    User admin
    IdentityFile ~/.ssh/bastion_key

Host *
    ServerAliveInterval 30
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Configure the library to use our test config
	settings := &ssh_config.UserSettings{IgnoreErrors: true}
	settings.ConfigFinder(func() string {
		return configPath
	})

	// Test explicit host entry
	t.Run("explicit host entry", func(t *testing.T) {
		hostname, _ := settings.GetStrict("test-server", "HostName")
		if hostname != "192.168.1.100" {
			t.Errorf("expected hostname 192.168.1.100, got %s", hostname)
		}

		user, _ := settings.GetStrict("test-server", "User")
		if user != "testuser" {
			t.Errorf("expected user testuser, got %s", user)
		}

		port, _ := settings.GetStrict("test-server", "Port")
		if port != "2222" {
			t.Errorf("expected port 2222, got %s", port)
		}
	})

	// Test wildcard pattern matching
	t.Run("wildcard pattern", func(t *testing.T) {
		user, _ := settings.GetStrict("bastion-prod", "User")
		if user != "admin" {
			t.Errorf("expected user admin for bastion-prod, got %s", user)
		}
	})
}

func TestResolve_NoSSHConfig(t *testing.T) {
	// Test with a host that's not in any SSH config
	resolved := Resolve("unknown-host.example.com")

	if resolved.Hostname != "unknown-host.example.com" {
		t.Errorf("expected hostname to be the alias itself, got %s", resolved.Hostname)
	}

	if resolved.Port != "22" {
		t.Errorf("expected default port 22, got %s", resolved.Port)
	}
}

func TestResolvedHost_Address(t *testing.T) {
	tests := []struct {
		name     string
		host     ResolvedHost
		expected string
	}{
		{
			name:     "default port",
			host:     ResolvedHost{Hostname: "example.com", Port: "22"},
			expected: "example.com:22",
		},
		{
			name:     "custom port",
			host:     ResolvedHost{Hostname: "192.168.1.1", Port: "2222"},
			expected: "192.168.1.1:2222",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.host.Address(); got != tt.expected {
				t.Errorf("Address() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestResolvedHost_IsFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		host     ResolvedHost
		alias    string
		expected bool
	}{
		{
			name:     "hostname differs from alias",
			host:     ResolvedHost{Hostname: "192.168.1.100", User: "", Port: "22"},
			alias:    "myserver",
			expected: true,
		},
		{
			name:     "has user",
			host:     ResolvedHost{Hostname: "myserver", User: "admin", Port: "22"},
			alias:    "myserver",
			expected: true,
		},
		{
			name:     "has identity files",
			host:     ResolvedHost{Hostname: "myserver", Port: "22", IdentityFiles: []string{"~/.ssh/key"}},
			alias:    "myserver",
			expected: true,
		},
		{
			name:     "custom port",
			host:     ResolvedHost{Hostname: "myserver", Port: "2222"},
			alias:    "myserver",
			expected: true,
		},
		{
			name:     "not from config",
			host:     ResolvedHost{Hostname: "example.com", Port: "22"},
			alias:    "example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.host.IsFromConfig(tt.alias); got != tt.expected {
				t.Errorf("IsFromConfig(%s) = %v, want %v", tt.alias, got, tt.expected)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("could not get home directory")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde expansion",
			input:    "~/test/path",
			expected: filepath.Join(home, "test/path"),
		},
		{
			name:     "no tilde",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expandPath(tt.input); got != tt.expected {
				t.Errorf("expandPath(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}
