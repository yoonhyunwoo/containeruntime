package cmd

import (
	"context"

	"github.com/urfave/cli/v3"
)

var KillCommand = &cli.Command{
	Name: "kill",
	Action: func(ctx context.Context, command *cli.Command) error {
		return nil
	},
}
