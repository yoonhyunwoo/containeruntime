package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
	cmd "github.com/yoonhyunwoo/containeruntime/cmd"
	"github.com/yoonhyunwoo/containeruntime/internal/container"
)

func main() {
	if err := container.InitStateDir(); err != nil {
		log.Fatal(err)
	}

	rootCmd := &cli.Command{
		Commands: []*cli.Command{
			cmd.CreateCommand,
			cmd.DeleteCommand,
			cmd.InitCommand,
			cmd.KillCommand,
			cmd.StartCommand,
			cmd.StateCommand,
		},
	}

	if err := rootCmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}

}
