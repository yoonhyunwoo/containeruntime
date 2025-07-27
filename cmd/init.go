package cmd

import (
	"context"

	"github.com/urfave/cli/v3"
	"github.com/yoonhyunwoo/containeruntime/internal/container"
)

var InitCommand = &cli.Command{
	Name: "init",
	Action: func(ctx context.Context, command *cli.Command) error {
		container.Init()
		return nil
	},
}
