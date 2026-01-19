// Package daemon is the background service that manages SSH tunnels
package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/JoshElias/gurren/internal/config"
	"github.com/JoshElias/gurren/internal/tunnel"
)

const Version = "0.1.1"

type Daemon struct {
	config   *config.Config
	manager  *tunnel.Manager
	listener net.Listener

	// Subscriber management
	mu          sync.RWMutex
	subscribers map[*subscriber]struct{}

	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// subscriber represents a connected client that wants status updates
type subscriber struct {
	conn    net.Conn
	encoder *json.Encoder
	mu      sync.Mutex
}

// New creates a new daemon instance
func New(cfg *config.Config) *Daemon {
	ctx, cancel := context.WithCancel(context.Background())

	d := &Daemon{
		config:      cfg,
		manager:     tunnel.NewManager(cfg),
		subscribers: make(map[*subscriber]struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Set up status change notifications
	d.manager.SetOnChange(d.broadcastStatusChange)

	return d
}

// SocketPath returns the path to the daemon socket
func SocketPath() (string, error) {
	// Use XDG_RUNTIME_DIR if available, otherwise use ~/.local/state
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	stateDirName := "gurren"
	if runtimeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to get home directory: %w", err)
		}
		runtimeDir = filepath.Join(home, ".local", "state")
		stateDirName = ".gurren"
	}

	stateDir := filepath.Join(runtimeDir, stateDirName)
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return "", fmt.Errorf("unable to create state directory: %w", err)
	}

	return filepath.Join(stateDir, "daemon.sock"), nil
}

// Start starts the daemon, listening on the Unix socket
func (d *Daemon) Start() error {
	if IsRunning() {
		return fmt.Errorf("daemon is already running")
	}

	socketPath, err := SocketPath()
	if err != nil {
		return err
	}

	// Remove existing stale socket if present
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to remove existing socket: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("unable to listen on socket: %w", err)
	}
	d.listener = listener

	// Set socket permissions
	if err := os.Chmod(socketPath, 0o600); err != nil {
		log.Printf("Warning: unable to set socket permissions: %v", err)
	}

	log.Printf("Daemon listening on %s", socketPath)

	// Accept connections
	go d.acceptLoop()

	return nil
}

// acceptLoop accepts incoming connections
func (d *Daemon) acceptLoop() {
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			if d.ctx.Err() != nil {
				return // Shutting down
			}
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go d.handleConnection(conn)
	}
}

// handleConnection handles a single client connection
func (d *Daemon) handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	sub := &subscriber{
		conn:    conn,
		encoder: json.NewEncoder(conn),
	}

	reader := bufio.NewReader(conn)
	decoder := json.NewDecoder(reader)

	for {
		var req Request
		if err := decoder.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				break
			}
			log.Printf("Error decoding request: %v", err)
			break
		}

		resp := d.handleRequest(sub, &req)

		sub.mu.Lock()
		if err := sub.encoder.Encode(resp); err != nil {
			sub.mu.Unlock()
			log.Printf("Error encoding response: %v", err)
			break
		}
		sub.mu.Unlock()
	}

	// Remove from subscribers if subscribed
	d.mu.Lock()
	delete(d.subscribers, sub)
	d.mu.Unlock()
}

// handleRequest dispatches a request to the appropriate handler
func (d *Daemon) handleRequest(sub *subscriber, req *Request) Response {
	switch req.Method {
	case MethodSubscribe:
		return d.handleSubscribe(sub, req)
	case MethodTunnelStart:
		return d.handleTunnelStart(req)
	case MethodTunnelStop:
		return d.handleTunnelStop(req)
	case MethodTunnelStatus:
		return d.handleTunnelStatus(req)
	case MethodTunnelList:
		return d.handleTunnelList(req)
	case MethodTunnelRegister:
		return d.handleTunnelRegister(req)
	case MethodDaemonPing:
		return d.handlePing(req)
	case MethodDaemonShutdown:
		return d.handleShutdown(req)
	default:
		return NewError(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("unknown method: %s", req.Method))
	}
}

// broadcastStatusChange sends a status change notification to all subscribers
func (d *Daemon) broadcastStatusChange(change tunnel.StatusChange) {
	notification := NewNotification(MethodStatusChanged, StatusChangedParams{
		Name:   change.Name,
		Status: change.Status,
		Error:  change.Error,
	})

	d.mu.RLock()
	defer d.mu.RUnlock()

	for sub := range d.subscribers {
		sub.mu.Lock()
		if err := sub.encoder.Encode(notification); err != nil {
			log.Printf("Error sending notification: %v", err)
		}
		sub.mu.Unlock()
	}
}

// Shutdown gracefully stops the daemon
func (d *Daemon) Shutdown() {
	d.cancel()
	d.manager.StopAll()
	if d.listener != nil {
		_ = d.listener.Close()
	}
}

// Wait blocks until the daemon context is cancelled
func (d *Daemon) Wait() {
	<-d.ctx.Done()
}

// Manager returns the tunnel manager (for handlers)
func (d *Daemon) Manager() *tunnel.Manager {
	return d.manager
}

// Config returns the configuration (for handlers)
func (d *Daemon) Config() *config.Config {
	return d.config
}
