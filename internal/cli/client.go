package cli

import (
	"broadcast-server/internal/client"
	"broadcast-server/pkg/logger"

	"github.com/spf13/cobra"
)

// NewClientCommand creates the client command
func NewClientCommand() *cobra.Command {
	var host string
	var port int
	var username string

	clientCmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to a broadcast server",
		Long:  `Connect to a broadcast server and send/receive messages.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Connecting to %s:%d as %s...", host, port, username)

			c := client.NewClient(host, port, username)
			return c.Connect()
		},
	}

	// Add flags
	clientCmd.Flags().StringVarP(&host, "host", "H", "localhost", "Server host to connect to")
	clientCmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port to connect to")
	clientCmd.Flags().StringVarP(&username, "username", "u", "anonymous", "Username to use")

	return clientCmd
}
