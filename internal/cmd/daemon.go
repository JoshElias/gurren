package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/JoshElias/gurren/internal/config"
	"github.com/JoshElias/gurren/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonForeground bool

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the background daemon",
	Long:  `The daemon runs in the background and manages SSH tunnel connections.`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon",
	Long:  `Starts the daemon in the background to manage SSH tunnels.`,
	Run:   runDaemonStart,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the daemon",
	Long:  `Stops the daemon and all running tunnels.`,
	Run:   runDaemonStop,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	Long:  `Checks if the daemon is running.`,
	Run:   runDaemonStatus,
}

func init() {
	daemonStartCmd.Flags().BoolVar(&daemonForeground, "foreground", false, "Run daemon in foreground (don't detach)")
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd)
}

func runDaemonStart(cmd *cobra.Command, args []string) {
	// Check if already running
	if daemon.IsRunning() {
		fmt.Println("Daemon is already running")
		return
	}

	// If not foreground mode, fork to background
	if !daemonForeground {
		if err := startDaemonInBackground(); err != nil {
			log.Fatalf("Failed to start daemon: %v", err)
		}
		fmt.Println("Daemon started")
		return
	}

	// Foreground mode - run daemon in this process
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	d := daemon.New(cfg)
	if err := d.Start(); err != nil {
		log.Fatalf("Error starting daemon: %v", err)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	fmt.Println("\nShutting down...")
	d.Shutdown()
}

// startDaemonInBackground starts the daemon as a detached background process
func startDaemonInBackground() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := exec.Command(exePath, "daemon", "start", "--foreground")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait for daemon to be ready
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if daemon.IsRunning() {
			return nil
		}
	}

	return fmt.Errorf("daemon did not start in time")
}

func runDaemonStop(cmd *cobra.Command, args []string) {
	client, err := daemon.Connect()
	if err != nil {
		fmt.Println("Daemon is not running")
		return
	}
	defer func() { _ = client.Close() }()

	if err := client.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Daemon stopped")
}

func runDaemonStatus(cmd *cobra.Command, args []string) {
	client, err := daemon.Connect()
	if err != nil {
		fmt.Println("Daemon is not running")
		os.Exit(1)
	}
	defer func() { _ = client.Close() }()

	result, err := client.Ping()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Daemon is running (version %s)\n", result.Version)
}
