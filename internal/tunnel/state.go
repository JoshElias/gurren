// Package tunnel manages the SSH connection between 2 hosts via a remote bastion
package tunnel

// State represents the current state of a tunnel
type State string

const (
	StateDisconnected State = "disconnected"
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
	StateError        State = "error"
)

// String returns the string representation of the state
func (s State) String() string {
	return string(s)
}

// IsActive returns true if the tunnel is connecting or connected
func (s State) IsActive() bool {
	return s == StateConnecting || s == StateConnected
}
