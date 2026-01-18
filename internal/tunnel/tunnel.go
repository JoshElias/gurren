// Package tunnel manages the SSH connection between 2 hosts via a remote bastion
package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

// ErrTunnelClosed is returned when the tunnel is closed via context cancellation
var ErrTunnelClosed = errors.New("tunnel closed")

// Tunnel represents an SSH tunnel configuration.
type Tunnel struct {
	SSHHost    string // SSH server address (host:port)
	SSHUser    string // SSH username
	RemoteAddr string // Remote endpoint to tunnel to (host:port)
	LocalAddr  string // Local bind address (host:port)
}

// Start establishes the SSH tunnel and listens for local connections.
// This function blocks until the context is cancelled or an error occurs.
func Start(ctx context.Context, t *Tunnel, authMethods []ssh.AuthMethod) error {
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
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", t.LocalAddr)
	if err != nil {
		return fmt.Errorf("unable to listen on %s: %w", t.LocalAddr, err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			log.Printf("Warning: error closing listener: %v", err)
		}
	}()

	log.Printf("Tunnel active: %s -> %s (via %s)", t.LocalAddr, t.RemoteAddr, t.SSHHost)

	// Track active connections for graceful shutdown
	var wg sync.WaitGroup
	connCtx, connCancel := context.WithCancel(ctx)
	defer connCancel()

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	// Accept connections
	for {
		localConn, err := listener.Accept()
		if err != nil {
			// Check if we're shutting down
			if ctx.Err() != nil {
				// Wait for active connections to finish
				wg.Wait()
				return ErrTunnelClosed
			}
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			handleConnection(connCtx, sshClient, localConn, t.RemoteAddr)
		}()
	}
}

func handleConnection(ctx context.Context, sshClient *ssh.Client, localConn net.Conn, remoteAddr string) {
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
		if err != nil && ctx.Err() == nil {
			log.Printf("Error copying to remote: %v", err)
		}
		done <- struct{}{}
	}()

	go func() {
		_, err := io.Copy(localConn, remoteConn)
		if err != nil && ctx.Err() == nil {
			log.Printf("Error copying from remote: %v", err)
		}
		done <- struct{}{}
	}()

	// Wait for one side to close or context cancellation
	select {
	case <-done:
	case <-ctx.Done():
		// Force close connections to unblock io.Copy
		_ = localConn.Close()
		_ = remoteConn.Close()
		<-done
	}
}
