package tunnel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/JoshElias/gurren/internal/config"
	"golang.org/x/crypto/ssh"
)

// StatusChange represents a tunnel status change event
type StatusChange struct {
	Name   string
	Status State
	Error  string
}

// Manager manages multiple tunnels and tracks their state
type Manager struct {
	mu       sync.RWMutex
	tunnels  map[string]*ManagedTunnel
	config   *config.Config
	onChange func(StatusChange) // callback for status changes
}

// ManagedTunnel represents a tunnel being managed by the Manager
type ManagedTunnel struct {
	Config    config.TunnelConfig
	Status    State
	Error     string
	cancel    context.CancelFunc
	startedAt time.Time
}

// NewManager creates a new tunnel manager
func NewManager(cfg *config.Config) *Manager {
	m := &Manager{
		tunnels: make(map[string]*ManagedTunnel),
		config:  cfg,
	}

	// Initialize all configured tunnels as disconnected
	for _, tc := range cfg.Tunnels {
		m.tunnels[tc.Name] = &ManagedTunnel{
			Config: tc,
			Status: StateDisconnected,
		}
	}

	return m
}

// SetOnChange sets the callback for status changes
func (m *Manager) SetOnChange(fn func(StatusChange)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = fn
}

// notifyChange notifies subscribers of a status change
func (m *Manager) notifyChange(name string, status State, errMsg string) {
	if m.onChange != nil {
		m.onChange(StatusChange{
			Name:   name,
			Status: status,
			Error:  errMsg,
		})
	}
}

// Start starts a tunnel by name
func (m *Manager) Start(name string, authMethods []ssh.AuthMethod, sshHost, sshUser string) error {
	m.mu.Lock()

	mt, exists := m.tunnels[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("tunnel %q not found", name)
	}

	if mt.Status.IsActive() {
		m.mu.Unlock()
		return fmt.Errorf("tunnel %q is already %s", name, mt.Status)
	}

	// Update status to connecting
	mt.Status = StateConnecting
	mt.Error = ""
	mt.startedAt = time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	mt.cancel = cancel

	onChange := m.onChange
	m.mu.Unlock()

	// Notify connecting
	if onChange != nil {
		onChange(StatusChange{Name: name, Status: StateConnecting})
	}

	// Start tunnel in goroutine
	go func() {
		t := &Tunnel{
			SSHHost:    sshHost,
			SSHUser:    sshUser,
			RemoteAddr: mt.Config.Remote,
			LocalAddr:  mt.Config.Local,
		}

		err := Start(ctx, t, authMethods)

		m.mu.Lock()
		if err != nil && err != ErrTunnelClosed {
			mt.Status = StateError
			mt.Error = err.Error()
		} else {
			mt.Status = StateDisconnected
			mt.Error = ""
		}
		mt.cancel = nil
		status := mt.Status
		errMsg := mt.Error
		onChange := m.onChange
		m.mu.Unlock()

		if onChange != nil {
			onChange(StatusChange{Name: name, Status: status, Error: errMsg})
		}
	}()

	// Give tunnel a moment to connect or fail
	time.Sleep(100 * time.Millisecond)

	m.mu.Lock()
	// If still connecting after brief wait, consider it connected
	if mt.Status == StateConnecting {
		mt.Status = StateConnected
		onChange = m.onChange
		m.mu.Unlock()

		if onChange != nil {
			onChange(StatusChange{Name: name, Status: StateConnected})
		}
	} else {
		m.mu.Unlock()
	}

	return nil
}

// Stop stops a running tunnel by name
func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mt, exists := m.tunnels[name]
	if !exists {
		return fmt.Errorf("tunnel %q not found", name)
	}

	if !mt.Status.IsActive() {
		return fmt.Errorf("tunnel %q is not running", name)
	}

	if mt.cancel != nil {
		mt.cancel()
	}

	return nil
}

// Status returns the status of a tunnel
func (m *Manager) Status(name string) (State, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mt, exists := m.tunnels[name]
	if !exists {
		return StateDisconnected, "tunnel not found"
	}

	return mt.Status, mt.Error
}

// List returns all managed tunnels
func (m *Manager) List() []ManagedTunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ManagedTunnel, 0, len(m.tunnels))
	for _, mt := range m.tunnels {
		result = append(result, ManagedTunnel{
			Config:    mt.Config,
			Status:    mt.Status,
			Error:     mt.Error,
			startedAt: mt.startedAt,
		})
	}

	return result
}

// StopAll stops all running tunnels
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mt := range m.tunnels {
		if mt.cancel != nil {
			mt.cancel()
		}
	}
}

// GetConfig returns the config for a tunnel
func (m *Manager) GetConfig(name string) *config.TunnelConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mt, exists := m.tunnels[name]
	if !exists {
		return nil
	}

	return &mt.Config
}
