package cmd

import (
	"context"

	"github.com/yoonhyunwoo/containeruntime/internal/container"
	"github.com/yoonhyunwoo/containeruntime/internal/linux/cgroup"

	"github.com/urfave/cli/v3"
)

var CreateCommand = &cli.Command{
	Name:      "create",
	Usage:     "This command creates a new container. You must provide a unique container ID and the path to the bundle containing the container's configuration.",
	ArgsUsage: "<container-id> <path-to-bundle>",
	Action: func(ctx context.Context, command *cli.Command) error {
		cgroup.SetupCgroups()
		container.Create()
		return nil
	},
}
