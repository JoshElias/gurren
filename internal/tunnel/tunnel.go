// Package tunnel manages the SSH connection between 2 hosts via a remote bastion
package tunnel

import (
	"fmt"
	"io"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
)

// Tunnel represents an SSH tunnel configuration.
type Tunnel struct {
	SSHHost    string // SSH server address (host:port)
	SSHUser    string // SSH username
	RemoteAddr string // Remote endpoint to tunnel to (host:port)
	LocalAddr  string // Local bind address (host:port)
}

// Start establishes the SSH tunnel and listens for local connections.
// This function blocks until the tunnel is closed or an error occurs.
func Start(t *Tunnel, authMethods []ssh.AuthMethod) error {
	config := &ssh.ClientConfig{
		User:            t.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: implement proper host key verification
	}

	// Connect to SSH server
	sshClient, err := ssh.Dial("tcp", t.SSHHost, config)
	if err != nil {
		return fmt.Errorf("unable to connect to SSH server %s: %w", t.SSHHost, err)
	}
	defer func() {
		if err := sshClient.Close(); err != nil {
			log.Printf("Warning: error closing SSH client: %v", err)
		}
	}()

	log.Printf("Connected to %s", t.SSHHost)

	// Start local listener
	listener, err := net.Listen("tcp", t.LocalAddr)
	if err != nil {
		return fmt.Errorf("unable to listen on %s: %w", t.LocalAddr, err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			log.Printf("Warning: error closing listener: %v", err)
		}
	}()

	log.Printf("Tunnel active: %s -> %s (via %s)", t.LocalAddr, t.RemoteAddr, t.SSHHost)

	// Accept connections
	for {
		localConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleConnection(sshClient, localConn, t.RemoteAddr)
	}
}

func handleConnection(sshClient *ssh.Client, localConn net.Conn, remoteAddr string) {
	defer func() {
		if err := localConn.Close(); err != nil {
			log.Printf("Warning: error closing local connection: %v", err)
		}
	}()

	// Dial remote through SSH
	remoteConn, err := sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("Failed to dial remote %s: %v", remoteAddr, err)
		return
	}
	defer func() {
		if err := remoteConn.Close(); err != nil {
			log.Printf("Warning: error closing remote connection: %v", err)
		}
	}()

	// Bidirectional copy
	done := make(chan struct{}, 2)

	go func() {
		_, err := io.Copy(remoteConn, localConn)
		if err != nil {
			log.Printf("Error copying to remote: %v", err)
		}
		done <- struct{}{}
	}()

	go func() {
		_, err := io.Copy(localConn, remoteConn)
		if err != nil {
			log.Printf("Error copying from remote: %v", err)
		}
		done <- struct{}{}
	}()

	<-done // Wait for one side to close
}
