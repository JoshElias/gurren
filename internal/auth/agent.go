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
	conn, err := getSocketConn()
	if err != nil {
		return false
	}
	_ = conn.Close()

	return true
}

func (a *AgentAuthenticator) GetAuthMethod() (ssh.AuthMethod, error) {
	conn, err := getSocketConn()
	if err != nil {
		return nil, err
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers), nil
}

func getSocketConn() (net.Conn, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	return net.Dial("unix", socket)
}
