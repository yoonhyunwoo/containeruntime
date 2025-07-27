package cmd

import (
	"context"

	"github.com/yoonhyunwoo/containeruntime/internal/container"
	"github.com/yoonhyunwoo/containeruntime/internal/linux/cgroup"

	"github.com/urfave/cli/v3"
)

var CreateCommand = &cli.Command{
	Name: "create",
	Action: func(ctx context.Context, command *cli.Command) error {
		cgroup.SetupCgroups()
		container.Run()
		return nil
	},
}
