package cmd

import (
	"fmt"
	"os"

	"github.com/JoshElias/gurren/internal/daemon"
	"github.com/spf13/cobra"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect [tunnel-name]",
	Short: "Disconnect a running tunnel",
	Long:  `Stops a running tunnel managed by the service.`,
	Args:  cobra.ExactArgs(1),
	Run:   runDisconnect,
}

func init() {
	rootCmd.AddCommand(disconnectCmd)
}

func runDisconnect(cmd *cobra.Command, args []string) {
	name := args[0]

	client, err := daemon.Connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: service not running. Start with 'gurren service start'\n")
		os.Exit(1)
	}
	defer client.Close()

	if err := client.TunnelStop(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Tunnel %q disconnected\n", name)
}
