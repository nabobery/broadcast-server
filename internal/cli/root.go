package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command for the broadcast-server CLI
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "broadcast-server",
		Short: "A simple broadcast server application",
		Long:  `broadcast-server allows you to create a server that broadcasts messages to all connected clients.`,
	}

	// Add subcommands
	rootCmd.AddCommand(NewServerCommand())
	rootCmd.AddCommand(NewClientCommand())

	return rootCmd
}
