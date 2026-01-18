// Package cmd manages the CLI entrypoint for Cobra
package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/JoshElias/gurren/internal/daemon"
	"github.com/JoshElias/gurren/internal/tui"
	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	authMethod string
)

var rootCmd = &cobra.Command{
	Use:   "gurren",
	Short: "SSH tunnel manager",
	Long:  `Gurren is an SSH tunnel manager CLI and TUI that simplifies connecting to remote services through bastion hosts.`,
	Run:   runRoot,
}

var connectCmd = &cobra.Command{
	Use:   "connect [tunnel-name]",
	Short: "Connect to a configured tunnel",
	Long: `Connect establishes an SSH tunnel to a remote service.

If a tunnel name is provided, it uses the configuration from the config file.
Otherwise, you can specify the connection details via flags.`,
	Args: cobra.MaximumNArgs(1),
	Run:  runConnect,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/gurren/config.toml)")
	rootCmd.PersistentFlags().StringVarP(&authMethod, "auth", "a", "", "auth method: auto, agent, publickey, password (default: auto)")

	// Connect command flags
	connectCmd.Flags().String("host", "", "SSH host (user@host:port or host from ~/.ssh/config)")
	connectCmd.Flags().String("remote", "", "Remote address (host:port)")
	connectCmd.Flags().String("local", "", "Local bind address (host:port)")

	rootCmd.AddCommand(connectCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func runConnect(cmd *cobra.Command, args []string) {
	// Ensure daemon is running
	if !daemon.IsRunning() {
		if err := startDaemonBackground(); err != nil {
			log.Fatalf("Failed to start daemon: %v", err)
		}
	}

	// Connect to daemon
	client, err := daemon.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to daemon: %v", err)
	}
	defer client.Close()

	var tunnelName string

	// If tunnel name provided, use it directly
	if len(args) > 0 {
		tunnelName = args[0]
	} else {
		// Ad-hoc tunnel from flags - register it first
		host, _ := cmd.Flags().GetString("host")
		remote, _ := cmd.Flags().GetString("remote")
		local, _ := cmd.Flags().GetString("local")

		if host == "" || remote == "" || local == "" {
			log.Fatal("When not using a named tunnel, --host, --remote, and --local are required")
		}

		result, err := client.TunnelRegister(host, remote, local)
		if err != nil {
			log.Fatalf("Failed to register tunnel: %v", err)
		}
		tunnelName = result.Name
		fmt.Printf("Registered ad-hoc tunnel: %s\n", tunnelName)
	}

	// Start the tunnel
	_, err = client.TunnelStart(tunnelName)
	if err != nil {
		log.Fatalf("Failed to start tunnel: %v", err)
	}

	// Get tunnel details for display
	tunnelList, err := client.TunnelList()
	if err != nil {
		log.Printf("Warning: couldn't fetch tunnel details: %v", err)
	} else {
		for _, t := range tunnelList.Tunnels {
			if t.Name == tunnelName {
				fmt.Printf("Tunnel %q connected.\n", tunnelName)
				fmt.Printf("  %s -> %s (via %s)\n", t.Config.Local, t.Config.Remote, t.Config.Host)
				break
			}
		}
	}

	fmt.Println("Press Ctrl+C to disconnect.")

	// Subscribe to notifications to detect if tunnel is stopped elsewhere
	if err := client.Subscribe(); err != nil {
		log.Printf("Warning: couldn't subscribe to notifications: %v", err)
	}

	// Wait for either:
	// 1. Interrupt signal (user pressed Ctrl+C)
	// 2. Tunnel disconnected notification (stopped from TUI or another CLI)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	disconnectedByRemote := false

	// Listen for notifications in background
	doneCh := make(chan struct{})
	go func() {
		for notif := range client.Notifications() {
			if notif.Method == daemon.MethodStatusChanged {
				var params daemon.StatusChangedParams
				if err := json.Unmarshal(notif.Params, &params); err == nil {
					if params.Name == tunnelName && !params.Status.IsActive() {
						disconnectedByRemote = true
						close(doneCh)
						return
					}
				}
			}
		}
	}()

	// Wait for signal or remote disconnect
	select {
	case <-sigCh:
		fmt.Println("\nDisconnecting...")
		if err := client.TunnelStop(tunnelName); err != nil {
			log.Printf("Warning: failed to stop tunnel: %v", err)
		}
	case <-doneCh:
		fmt.Println("\nTunnel disconnected.")
	}

	if !disconnectedByRemote {
		fmt.Printf("Tunnel %q disconnected.\n", tunnelName)
	}
}

// Silence usage output
func init() {
	rootCmd.SilenceUsage = true
}

// runRoot launches the TUI when no subcommand is specified
func runRoot(cmd *cobra.Command, args []string) {
	// Ensure daemon is running
	if !daemon.IsRunning() {
		// Start daemon in background
		if err := startDaemonBackground(); err != nil {
			log.Fatalf("Failed to start daemon: %v", err)
		}
	}

	// Connect to daemon
	client, err := daemon.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to daemon: %v", err)
	}
	defer client.Close()

	// Run TUI
	if err := tui.Run(client); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

// startDaemonBackground starts the daemon as a background process
func startDaemonBackground() error {
	// Get the path to our own executable
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Start daemon in background
	cmd := exec.Command(exePath, "daemon", "start")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Detach from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait a bit for daemon to start
	// Poll until daemon is running or timeout
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if daemon.IsRunning() {
			return nil
		}
	}

	return fmt.Errorf("daemon did not start in time")
}
