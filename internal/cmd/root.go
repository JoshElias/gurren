// Package cmd manages the CLI entrypoint for Cobra
package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/JoshElias/gurren/internal/auth"
	"github.com/JoshElias/gurren/internal/config"
	"github.com/JoshElias/gurren/internal/tunnel"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var tunnelCfg *config.TunnelConfig

	// If tunnel name provided, look it up
	if len(args) > 0 {
		tunnelCfg = cfg.GetTunnel(args[0])
		if tunnelCfg == nil {
			log.Fatalf("Tunnel %q not found. Available tunnels: %v", args[0], cfg.TunnelNames())
		}
	} else {
		// Build tunnel config from flags
		host, _ := cmd.Flags().GetString("host")
		remote, _ := cmd.Flags().GetString("remote")
		local, _ := cmd.Flags().GetString("local")

		if host == "" || remote == "" || local == "" {
			log.Fatal("When not using a named tunnel, --host, --remote, and --local are required")
		}

		tunnelCfg = &config.TunnelConfig{
			Host:   host,
			Remote: remote,
			Local:  local,
		}
	}

	// Determine auth method
	method := authMethod
	if method == "" {
		method = cfg.Auth.Method
		fmt.Printf("Auth method that came from config: %v", method)
	}
	if method == "" {
		method = "auto"
	}

	// Get auth methods
	authMethods, err := auth.GetAuthMethodsByName(method)
	if err != nil {
		log.Fatalf("Failed to get auth methods: %v", err)
	}

	// Parse SSH host
	sshHost, sshUser := parseHost(tunnelCfg.Host)

	// Start tunnel
	t := &tunnel.Tunnel{
		SSHHost:    sshHost,
		SSHUser:    sshUser,
		RemoteAddr: tunnelCfg.Remote,
		LocalAddr:  tunnelCfg.Local,
	}

	if err := tunnel.Start(t, authMethods); err != nil {
		log.Fatalf("Tunnel error: %v", err)
	}
}

// parseHost parses a host string like "user@host:port" or "host"
// Returns (host:port, user)
func parseHost(host string) (string, string) {
	user := ""
	addr := host

	// Extract user if present
	if idx := strings.Index(host, "@"); idx != -1 {
		user = host[:idx]
		addr = host[idx+1:]
	}

	// Add default port if not present
	if !strings.Contains(addr, ":") {
		addr = addr + ":22"
	}

	return addr, user
}

// Silence usage output
func init() {
	rootCmd.SilenceUsage = true
}
