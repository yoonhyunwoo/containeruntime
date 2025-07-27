package cmd

import (
	"context"

	"github.com/urfave/cli/v3"
)

var StartCommand = &cli.Command{
	Name: "start",
	Action: func(ctx context.Context, command *cli.Command) error {
		return nil
	},
}
