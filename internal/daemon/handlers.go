package daemon

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JoshElias/gurren/internal/auth"
	"github.com/JoshElias/gurren/internal/config"
)

// handleSubscribe adds the client to the subscribers list
func (d *Daemon) handleSubscribe(sub *subscriber, req *Request) Response {
	d.mu.Lock()
	d.subscribers[sub] = struct{}{}
	d.mu.Unlock()

	return NewResult(req.ID, struct{}{})
}

// handleTunnelStart starts a tunnel
func (d *Daemon) handleTunnelStart(req *Request) Response {
	var params TunnelStartParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewError(req.ID, ErrCodeInvalidParams, "invalid params")
	}

	if params.Name == "" {
		return NewError(req.ID, ErrCodeInvalidParams, "name is required")
	}

	// Get tunnel config - first check manager (includes ephemeral), then config file
	tunnelCfg := d.manager.GetConfig(params.Name)
	if tunnelCfg == nil {
		tunnelCfg = d.config.GetTunnelByName(params.Name)
	}
	if tunnelCfg == nil {
		return NewError(req.ID, ErrCodeTunnelNotFound, fmt.Sprintf("tunnel %q not found", params.Name))
	}

	// Get auth methods - for now, use non-interactive methods only
	// In the future, we could support interactive auth via the TUI
	authMethod := d.config.Auth.Method
	authMethods, err := auth.GetAuthMethodsByName(authMethod)
	if err != nil {
		return NewError(req.ID, ErrCodeAuthRequired, fmt.Sprintf("auth error: %v", err))
	}

	// Parse SSH host
	sshHost, sshUser := parseHost(tunnelCfg.Host)

	// Start the tunnel
	if err := d.manager.Start(params.Name, authMethods, sshHost, sshUser); err != nil {
		if strings.Contains(err.Error(), "already") {
			return NewError(req.ID, ErrCodeTunnelActive, err.Error())
		}
		return NewError(req.ID, ErrCodeInternal, err.Error())
	}

	status, errMsg := d.manager.Status(params.Name)
	return NewResult(req.ID, TunnelStatusResult{
		Name:   params.Name,
		Status: status,
		Error:  errMsg,
	})
}

// handleTunnelStop stops a running tunnel
func (d *Daemon) handleTunnelStop(req *Request) Response {
	var params TunnelStopParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewError(req.ID, ErrCodeInvalidParams, "invalid params")
	}

	if params.Name == "" {
		return NewError(req.ID, ErrCodeInvalidParams, "name is required")
	}

	if err := d.manager.Stop(params.Name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewError(req.ID, ErrCodeTunnelNotFound, err.Error())
		}
		if strings.Contains(err.Error(), "not running") {
			return NewError(req.ID, ErrCodeTunnelInactive, err.Error())
		}
		return NewError(req.ID, ErrCodeInternal, err.Error())
	}

	return NewResult(req.ID, struct{}{})
}

// handleTunnelStatus returns the status of a tunnel
func (d *Daemon) handleTunnelStatus(req *Request) Response {
	var params TunnelStatusParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewError(req.ID, ErrCodeInvalidParams, "invalid params")
	}

	if params.Name == "" {
		return NewError(req.ID, ErrCodeInvalidParams, "name is required")
	}

	status, errMsg := d.manager.Status(params.Name)
	if errMsg == "tunnel not found" {
		return NewError(req.ID, ErrCodeTunnelNotFound, errMsg)
	}

	return NewResult(req.ID, TunnelStatusResult{
		Name:   params.Name,
		Status: status,
		Error:  errMsg,
	})
}

// handleTunnelList returns all tunnels with their status
func (d *Daemon) handleTunnelList(req *Request) Response {
	managed := d.manager.List()

	tunnels := make([]TunnelInfo, len(managed))
	for i, mt := range managed {
		tunnels[i] = TunnelInfo{
			Name:      mt.Config.Name,
			Status:    mt.Status,
			Error:     mt.Error,
			Ephemeral: mt.Ephemeral,
			Config:    mt.Config,
		}
	}

	return NewResult(req.ID, TunnelListResult{Tunnels: tunnels})
}

// handleTunnelRegister registers an ad-hoc tunnel with a generated name
func (d *Daemon) handleTunnelRegister(req *Request) Response {
	var params TunnelRegisterParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewError(req.ID, ErrCodeInvalidParams, "invalid params")
	}

	if params.Host == "" || params.Remote == "" || params.Local == "" {
		return NewError(req.ID, ErrCodeInvalidParams, "host, remote, and local are required")
	}

	cfg := config.TunnelConfig{
		Host:   params.Host,
		Remote: params.Remote,
		Local:  params.Local,
	}

	name, err := d.manager.Register(cfg)
	if err != nil {
		return NewError(req.ID, ErrCodeInternal, err.Error())
	}

	return NewResult(req.ID, TunnelRegisterResult{Name: name})
}

// handlePing returns the daemon version
func (d *Daemon) handlePing(req *Request) Response {
	return NewResult(req.ID, PingResult{Version: Version})
}

// handleShutdown stops the daemon
func (d *Daemon) handleShutdown(req *Request) Response {
	// Send response before shutting down
	go func() {
		d.Shutdown()
	}()

	return NewResult(req.ID, struct{}{})
}

// parseHost parses a host string like "user@host:port" or "host"
// Returns (host:port, user)
func parseHost(host string) (string, string) {
	user := ""
	addr := host

	// Extract user if present
	if u, a, ok := strings.Cut(host, "@"); ok {
		user = u
		addr = a
	}

	// Add default port if not present
	if !strings.Contains(addr, ":") {
		addr = addr + ":22"
	}

	return addr, user
}
