package cmd

import (
	"context"

	"github.com/urfave/cli/v3"
)

var DeleteCommand = &cli.Command{
	Name: "delete",
	Action: func(ctx context.Context, command *cli.Command) error {
		return nil
	},
}
