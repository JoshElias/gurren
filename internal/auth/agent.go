package auth

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AgentAuthenticator provides SSH authentication via the SSH agent.
type AgentAuthenticator struct{}

func (a *AgentAuthenticator) Name() string {
	return "agent"
}

func (a *AgentAuthenticator) Priority() int {
	return 1 // Highest priority - try first
}

func (a *AgentAuthenticator) IsAvailable() bool {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return false
	}

	// Try to connect to verify agent is running
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return false
	}
	_ = conn.Close()

	return true
}

func (a *AgentAuthenticator) GetAuthMethod() (ssh.AuthMethod, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, err
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers), nil
}
