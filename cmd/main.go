package main

import (
	"broadcast-server/internal/cli"
	"broadcast-server/pkg/logger"
)

func main() {
	rootCmd := cli.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("Command execution failed: %v", err)
	}
}
