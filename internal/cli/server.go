package cli

import (
	"fmt"

	"broadcast-server/internal/server"
	"github.com/spf13/cobra"
)

// NewServerCommand creates the server command
func NewServerCommand() *cobra.Command {
	var port int

	serverCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the broadcast server",
		Long:  `Start a broadcast server that listens for client connections and broadcasts messages.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Starting broadcast server on port %d...\n", port)

			s := server.NewServer(port)
			return s.Start()
		},
	}

	// Add flags
	serverCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")

	return serverCmd
}
