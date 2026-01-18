package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

// Client is a client for communicating with the daemon
type Client struct {
	conn    net.Conn
	encoder *json.Encoder
	decoder *json.Decoder
	mu      sync.Mutex

	// Request ID counter
	nextID atomic.Uint64

	// Notifications channel for push updates
	notifications chan Notification

	// For coordinating reads
	responses   map[string]chan Response
	responsesMu sync.Mutex

	// Close handling
	closed   atomic.Bool
	closedCh chan struct{}
}

// Connect connects to the daemon
func Connect() (*Client, error) {
	socketPath, err := SocketPath()
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to daemon: %w", err)
	}

	c := &Client{
		conn:          conn,
		encoder:       json.NewEncoder(conn),
		decoder:       json.NewDecoder(bufio.NewReader(conn)),
		notifications: make(chan Notification, 100),
		responses:     make(map[string]chan Response),
		closedCh:      make(chan struct{}),
	}

	// Start reading responses and notifications
	go c.readLoop()

	return c, nil
}

// readLoop reads responses and notifications from the daemon
func (c *Client) readLoop() {
	defer close(c.closedCh)
	defer close(c.notifications)

	for {
		// We need to read into a raw message first to determine type
		var raw json.RawMessage
		if err := c.decoder.Decode(&raw); err != nil {
			if !c.closed.Load() {
				fmt.Printf("failed to decode json message from daemon")
			}
			return
		}

		// Try to parse as response first (has ID field)
		var resp Response
		if err := json.Unmarshal(raw, &resp); err == nil && resp.ID != "" {
			c.responsesMu.Lock()
			if ch, ok := c.responses[resp.ID]; ok {
				ch <- resp
				delete(c.responses, resp.ID)
			}
			c.responsesMu.Unlock()
			continue
		}

		// Otherwise it's a notification
		var notif Notification
		if err := json.Unmarshal(raw, &notif); err == nil && notif.Method != "" {
			select {
			case c.notifications <- notif:
			default:
				// Channel full, drop notification
			}
		}
	}
}

// Close closes the connection to the daemon
func (c *Client) Close() error {
	c.closed.Store(true)
	return c.conn.Close()
}

// Notifications returns the channel for receiving push notifications
func (c *Client) Notifications() <-chan Notification {
	return c.notifications
}

// call sends a request and waits for a response
func (c *Client) call(method string, params any) (Response, error) {
	id := fmt.Sprintf("%d", c.nextID.Add(1))

	var paramsRaw json.RawMessage
	if params != nil {
		var err error
		paramsRaw, err = json.Marshal(params)
		if err != nil {
			return Response{}, fmt.Errorf("failed to marshal params: %w", err)
		}
	}

	req := Request{
		ID:     id,
		Method: method,
		Params: paramsRaw,
	}

	// Create response channel
	respCh := make(chan Response, 1)
	c.responsesMu.Lock()
	c.responses[id] = respCh
	c.responsesMu.Unlock()

	// Send request
	c.mu.Lock()
	err := c.encoder.Encode(req)
	c.mu.Unlock()
	if err != nil {
		c.responsesMu.Lock()
		delete(c.responses, id)
		c.responsesMu.Unlock()
		return Response{}, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response
	select {
	case resp := <-respCh:
		return resp, nil
	case <-c.closedCh:
		return Response{}, fmt.Errorf("connection closed")
	}
}

// Subscribe subscribes to status change notifications
func (c *Client) Subscribe() error {
	resp, err := c.call(MethodSubscribe, nil)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("subscribe failed: %s", resp.Error.Message)
	}
	return nil
}

// Ping checks if the daemon is running
func (c *Client) Ping() (*PingResult, error) {
	resp, err := c.call(MethodDaemonPing, nil)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("ping failed: %s", resp.Error.Message)
	}

	var result PingResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}
	return &result, nil
}

// TunnelStart starts a tunnel
func (c *Client) TunnelStart(name string) (*TunnelStatusResult, error) {
	resp, err := c.call(MethodTunnelStart, TunnelStartParams{Name: name})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result TunnelStatusResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}
	return &result, nil
}

// TunnelStop stops a tunnel
func (c *Client) TunnelStop(name string) error {
	resp, err := c.call(MethodTunnelStop, TunnelStopParams{Name: name})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("%s", resp.Error.Message)
	}
	return nil
}

// TunnelStatus gets the status of a tunnel
func (c *Client) TunnelStatus(name string) (*TunnelStatusResult, error) {
	resp, err := c.call(MethodTunnelStatus, TunnelStatusParams{Name: name})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result TunnelStatusResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}
	return &result, nil
}

// TunnelList lists all tunnels
func (c *Client) TunnelList() (*TunnelListResult, error) {
	resp, err := c.call(MethodTunnelList, nil)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result TunnelListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}
	return &result, nil
}

// Shutdown tells the daemon to shut down
func (c *Client) Shutdown() error {
	resp, err := c.call(MethodDaemonShutdown, nil)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("%s", resp.Error.Message)
	}
	return nil
}

// IsRunning checks if the daemon is running
func IsRunning() bool {
	client, err := Connect()
	if err != nil {
		return false
	}
	defer func() { _ = client.Close() }()

	_, err = client.Ping()
	return err == nil
}
