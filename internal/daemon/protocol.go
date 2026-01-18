// Package daemon provides the background daemon for managing SSH tunnels
package daemon

import (
	"encoding/json"

	"github.com/JoshElias/gurren/internal/config"
	"github.com/JoshElias/gurren/internal/tunnel"
)

// Method constants for the JSON-RPC style protocol
const (
	MethodTunnelStart    = "tunnel.start"
	MethodTunnelStop     = "tunnel.stop"
	MethodTunnelStatus   = "tunnel.status"
	MethodTunnelList     = "tunnel.list"
	MethodDaemonPing     = "daemon.ping"
	MethodDaemonShutdown = "daemon.shutdown"
	MethodSubscribe      = "subscribe"

	// Notification methods (server -> client)
	MethodStatusChanged = "tunnel.statusChanged"
)

// Request is a message from client to daemon
type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response is a message from daemon to client
type Response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Notification is a push message from daemon to client (no ID)
type Notification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// Error represents an error in a response
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error codes
const (
	ErrCodeInternal       = -32603
	ErrCodeInvalidParams  = -32602
	ErrCodeMethodNotFound = -32601
	ErrCodeTunnelNotFound = 1001
	ErrCodeTunnelActive   = 1002
	ErrCodeTunnelInactive = 1003
	ErrCodeAuthRequired   = 1004
)

// --- Request Parameters ---

// TunnelStartParams are parameters for tunnel.start
type TunnelStartParams struct {
	Name string `json:"name"`
}

// TunnelStopParams are parameters for tunnel.stop
type TunnelStopParams struct {
	Name string `json:"name"`
}

// TunnelStatusParams are parameters for tunnel.status
type TunnelStatusParams struct {
	Name string `json:"name"`
}

// --- Response Results ---

// TunnelStatusResult is the result of tunnel.status
type TunnelStatusResult struct {
	Name   string       `json:"name"`
	Status tunnel.State `json:"status"`
	Error  string       `json:"error,omitempty"`
}

// TunnelInfo represents a tunnel in the list response
type TunnelInfo struct {
	Name   string              `json:"name"`
	Status tunnel.State        `json:"status"`
	Error  string              `json:"error,omitempty"`
	Config config.TunnelConfig `json:"config"`
}

// TunnelListResult is the result of tunnel.list
type TunnelListResult struct {
	Tunnels []TunnelInfo `json:"tunnels"`
}

// PingResult is the result of daemon.ping
type PingResult struct {
	Version string `json:"version"`
}

// --- Notification Parameters ---

// StatusChangedParams are parameters for tunnel.statusChanged notification
type StatusChangedParams struct {
	Name   string       `json:"name"`
	Status tunnel.State `json:"status"`
	Error  string       `json:"error,omitempty"`
}

// Helper functions for creating responses

// NewResult creates a successful response
func NewResult(id string, result any) Response {
	data, _ := json.Marshal(result)
	return Response{
		ID:     id,
		Result: data,
	}
}

// NewError creates an error response
func NewError(id string, code int, message string) Response {
	return Response{
		ID: id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}

// NewNotification creates a notification
func NewNotification(method string, params any) Notification {
	data, _ := json.Marshal(params)
	return Notification{
		Method: method,
		Params: data,
	}
}
