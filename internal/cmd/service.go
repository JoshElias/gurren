package cmd

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/JoshElias/gurren/internal/config"
	"github.com/JoshElias/gurren/internal/daemon"
	"github.com/spf13/cobra"
)

//go:embed gurren.service
var serviceFileTemplate string

var serviceForeground bool

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the background service",
	Long:  `The service runs in the background and manages SSH tunnel connections.`,
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the service",
	Long:  `Starts the service in the background to manage SSH tunnels.`,
	Run:   runServiceStart,
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the service",
	Long:  `Stops the service and all running tunnels.`,
	Run:   runServiceStop,
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check service status",
	Long:  `Checks if the service is running.`,
	Run:   runServiceStatus,
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install systemd user service",
	Long:  `Installs gurren as a systemd user service for automatic startup.`,
	Run:   runServiceInstall,
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall systemd user service",
	Long:  `Removes the gurren systemd user service.`,
	Run:   runServiceUninstall,
}

var serviceEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable systemd user service",
	Long:  `Enables gurren to start automatically on login.`,
	Run:   runServiceEnable,
}

var serviceDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable systemd user service",
	Long:  `Disables gurren from starting automatically on login.`,
	Run:   runServiceDisable,
}

func init() {
	serviceStartCmd.Flags().BoolVar(&serviceForeground, "foreground", false, "Run service in foreground (don't detach)")
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceEnableCmd)
	serviceCmd.AddCommand(serviceDisableCmd)
	rootCmd.AddCommand(serviceCmd)
}

func runServiceStart(cmd *cobra.Command, args []string) {
	// Check if already running
	if daemon.IsRunning() {
		fmt.Println("Service is already running")
		return
	}

	// If not foreground mode, fork to background
	if !serviceForeground {
		if err := startServiceInBackground(); err != nil {
			log.Fatalf("Failed to start service: %v", err)
		}
		fmt.Println("Service started")
		return
	}

	// Foreground mode - run service in this process
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	d := daemon.New(cfg)
	if err := d.Start(); err != nil {
		log.Fatalf("Error starting service: %v", err)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	fmt.Println("\nShutting down...")
	d.Shutdown()
}

// startServiceInBackground starts the service as a detached background process
func startServiceInBackground() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := exec.Command(exePath, "service", "start", "--foreground")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Wait for service to be ready
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if daemon.IsRunning() {
			return nil
		}
	}

	return fmt.Errorf("service did not start in time")
}

func runServiceStop(cmd *cobra.Command, args []string) {
	client, err := daemon.Connect()
	if err != nil {
		fmt.Println("Service is not running")
		return
	}
	defer func() { _ = client.Close() }()

	if err := client.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Service stopped")
}

func runServiceStatus(cmd *cobra.Command, args []string) {
	client, err := daemon.Connect()
	if err != nil {
		fmt.Println("Service is not running")
		os.Exit(1)
	}
	defer func() { _ = client.Close() }()

	result, err := client.Ping()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Service is running (version %s)\n", result.Version)
}

// systemd helpers

func systemdAvailable() bool {
	cmd := exec.Command("systemctl", "--user", "--version")
	return cmd.Run() == nil
}

func systemdServicePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "systemd", "user", "gurren.service"), nil
}

func systemdReload() error {
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	return cmd.Run()
}

func runServiceInstall(cmd *cobra.Command, args []string) {
	if !systemdAvailable() {
		fmt.Fprintln(os.Stderr, "Error: systemd is not available on this system")
		fmt.Fprintln(os.Stderr, "Use 'gurren service start' to run the service manually")
		os.Exit(1)
	}

	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	// Resolve symlinks to get the actual path
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		log.Fatalf("Failed to resolve executable path: %v", err)
	}

	servicePath, err := systemdServicePath()
	if err != nil {
		log.Fatalf("Failed to get service path: %v", err)
	}

	// Create directory if needed
	serviceDir := filepath.Dir(servicePath)
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		log.Fatalf("Failed to create directory %s: %v", serviceDir, err)
	}

	// Generate service file content
	serviceContent := strings.ReplaceAll(serviceFileTemplate, "{{EXEC_PATH}}", exePath)

	// Write service file
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0o644); err != nil {
		log.Fatalf("Failed to write service file: %v", err)
	}

	// Reload systemd
	if err := systemdReload(); err != nil {
		log.Fatalf("Failed to reload systemd: %v", err)
	}

	fmt.Printf("Installed systemd user service to %s\n", servicePath)
	fmt.Println()
	fmt.Println("To enable automatic startup on login:")
	fmt.Println("  gurren service enable")
	fmt.Println()
	fmt.Println("To start the service now:")
	fmt.Println("  systemctl --user start gurren")
}

func runServiceUninstall(cmd *cobra.Command, args []string) {
	if !systemdAvailable() {
		fmt.Fprintln(os.Stderr, "Error: systemd is not available on this system")
		os.Exit(1)
	}

	servicePath, err := systemdServicePath()
	if err != nil {
		log.Fatalf("Failed to get service path: %v", err)
	}

	// Check if service file exists
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		fmt.Println("Service is not installed")
		return
	}

	// Stop the service if running
	_ = exec.Command("systemctl", "--user", "stop", "gurren").Run()

	// Disable the service
	_ = exec.Command("systemctl", "--user", "disable", "gurren").Run()

	// Remove service file
	if err := os.Remove(servicePath); err != nil {
		log.Fatalf("Failed to remove service file: %v", err)
	}

	// Reload systemd
	if err := systemdReload(); err != nil {
		log.Fatalf("Failed to reload systemd: %v", err)
	}

	fmt.Println("Uninstalled systemd user service")
}

func runServiceEnable(cmd *cobra.Command, args []string) {
	if !systemdAvailable() {
		fmt.Fprintln(os.Stderr, "Error: systemd is not available on this system")
		os.Exit(1)
	}

	servicePath, err := systemdServicePath()
	if err != nil {
		log.Fatalf("Failed to get service path: %v", err)
	}

	// Check if service file exists
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Error: Service is not installed. Run 'gurren service install' first.")
		os.Exit(1)
	}

	enableCmd := exec.Command("systemctl", "--user", "enable", "gurren")
	if output, err := enableCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable service: %v\n%s", err, output)
		os.Exit(1)
	}

	fmt.Println("Service enabled - gurren will start automatically on login")
}

func runServiceDisable(cmd *cobra.Command, args []string) {
	if !systemdAvailable() {
		fmt.Fprintln(os.Stderr, "Error: systemd is not available on this system")
		os.Exit(1)
	}

	disableCmd := exec.Command("systemctl", "--user", "disable", "gurren")
	if output, err := disableCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to disable service: %v\n%s", err, output)
		os.Exit(1)
	}

	fmt.Println("Service disabled - gurren will no longer start automatically on login")
}
