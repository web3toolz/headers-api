package main

import (
	"github.com/urfave/cli/v2"
	"headers-api/cmd/cli/server"
	"log"
	"os"
)

func main() {
	app := &cli.App{
		Name:                 "headers-api",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			server.Cmd(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
