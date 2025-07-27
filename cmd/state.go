package cmd

import (
	"context"

	"github.com/urfave/cli/v3"
)

var StateCommand = &cli.Command{
	Name: "state",
	Action: func(ctx context.Context, command *cli.Command) error {
		return nil
	},
}
