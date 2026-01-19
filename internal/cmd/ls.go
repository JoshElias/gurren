package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/JoshElias/gurren/internal/daemon"
	"github.com/spf13/cobra"
)

var jsonOutput bool

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all configured tunnels",
	Long:  `Lists all tunnels from the configuration file along with their current status.`,
	Run:   runLs,
}

func init() {
	lsCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.AddCommand(lsCmd)
}

func runLs(cmd *cobra.Command, args []string) {
	client, err := daemon.Connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: service not running. Start with 'gurren service start'\n")
		os.Exit(1)
	}
	defer client.Close()

	result, err := client.TunnelList()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result.Tunnels); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(result.Tunnels) == 0 {
		fmt.Println("No tunnels configured")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tLOCAL\tREMOTE")

	for _, t := range result.Tunnels {
		status := string(t.Status)
		if t.Status == "error" && t.Error != "" {
			status = fmt.Sprintf("error: %s", t.Error)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Name, status, t.Config.Local, t.Config.Remote)
	}

	w.Flush()
}
