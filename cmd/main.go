package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"broadcast-server/internal/client"
	"broadcast-server/internal/server"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "broadcast-server",
		Usage: "A simple websocket broadcast server",
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start the broadcast server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "port",
						Value: "8080",
						Usage: "Port to run the server on",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					port := cmd.String("port")
					fmt.Printf("Starting server on port %s...\n", port)
					return server.Start(port)
				},
			},
			{
				Name:  "connect",
				Usage: "Connect to the broadcast server as a client",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "host",
						Value: "localhost",
						Usage: "Host to connect to",
					},
					&cli.StringFlag{
						Name:  "port",
						Value: "8080",
						Usage: "Port to connect to",
					},
					&cli.StringFlag{
						Name:  "username",
						Value: "anonymous",
						Usage: "Username to identify yourself",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					host := cmd.String("host")
					port := cmd.String("port")
					username := cmd.String("username")
					fmt.Printf("Connecting to %s:%s as %s...\n", host, port, username)
					return client.Connect(host, port, username)
				},
			},
		},
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
