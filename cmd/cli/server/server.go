package server

import (
	"github.com/urfave/cli/v2"
	cfg "headers-api/config"
	"headers-api/internal/app"
	"log"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "Config file path",
		Value:   "config.yaml",
	},
}

func Cmd() *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Run http server",
		Flags: flags,
		Action: func(cCtx *cli.Context) error {
			cfgPath := cCtx.String("config")
			config, err := cfg.LoadConfigFromYaml(cfgPath)

			if err != nil {
				log.Fatal("failed to load config", config)
			}

			application := app.NewApp(config)

			return application.Run()
		},
	}
}
