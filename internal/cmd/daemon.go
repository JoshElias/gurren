package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JoshElias/gurren/internal/config"
	"github.com/JoshElias/gurren/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the background daemon",
	Long:  `The daemon runs in the background and manages SSH tunnel connections.`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon",
	Long:  `Starts the daemon in the foreground. Use & to run in background.`,
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
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd)
}

func runDaemonStart(cmd *cobra.Command, args []string) {
	// Check if already running
	if daemon.IsRunning() {
		fmt.Println("Gurren daemon is already running")
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Create and start daemon
	d := daemon.New(cfg)
	if err := d.Start(); err != nil {
		log.Fatalf("Error starting Gurren daemon: %v", err)
	}

	fmt.Println("Gurren daemon started")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	fmt.Println("\nShutting down...")
	d.Shutdown()
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
