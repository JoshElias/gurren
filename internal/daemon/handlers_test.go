package daemon

import (
	"testing"
)

func TestParseHost(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedAddr      string
		expectedUser      string
		expectIdentityLen int // 0 means we don't check identity files (they come from real SSH config)
	}{
		{
			name:         "simple hostname",
			input:        "example.com",
			expectedAddr: "example.com:22",
			expectedUser: "",
		},
		{
			name:         "user@host format",
			input:        "admin@example.com",
			expectedAddr: "example.com:22",
			expectedUser: "admin",
		},
		{
			name:         "user@host:port format",
			input:        "admin@example.com:2222",
			expectedAddr: "example.com:2222",
			expectedUser: "admin",
		},
		{
			name:         "host:port format",
			input:        "example.com:2222",
			expectedAddr: "example.com:2222",
			expectedUser: "",
		},
		{
			name:         "IP address",
			input:        "192.168.1.100",
			expectedAddr: "192.168.1.100:22",
			expectedUser: "",
		},
		{
			name:         "user@IP:port",
			input:        "root@192.168.1.100:22",
			expectedAddr: "192.168.1.100:22",
			expectedUser: "root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, user, _ := parseHost(tt.input)

			if addr != tt.expectedAddr {
				t.Errorf("parseHost(%q) addr = %q, want %q", tt.input, addr, tt.expectedAddr)
			}

			if user != tt.expectedUser {
				t.Errorf("parseHost(%q) user = %q, want %q", tt.input, user, tt.expectedUser)
			}
		})
	}
}

func TestParseHost_SSHAlias(t *testing.T) {
	// Test that a simple alias (no @ or :) goes through SSH config resolution
	// This test uses the real SSH config, so we can't predict exact values,
	// but we can verify the function doesn't panic and returns valid data
	t.Run("ssh alias format", func(t *testing.T) {
		addr, _, _ := parseHost("some-alias")

		// Should have some address with a port
		if addr == "" {
			t.Error("parseHost returned empty address for SSH alias")
		}

		// If the alias isn't in SSH config, it should return alias:22
		// We can't test the exact value without mocking SSH config
	})
}
