package cmd

import (
	"context"
	"errors"

	"github.com/urfave/cli/v3"
	"github.com/yoonhyunwoo/containeruntime/internal/container"
	"github.com/yoonhyunwoo/containeruntime/internal/linux/cgroup"
)

var DeleteCommand = &cli.Command{
	Name:      "delete",
	Usage:     "This command deletes a container and its associated resources.",
	ArgsUsage: "<container-id>",
	Action: func(ctx context.Context, command *cli.Command) error {
		if command.Args().Len() != 1 {
			return errors.New("container-id is required")
		}

		containerId := command.Args().First()
		if err := container.Delete(containerId); err != nil {
			return err
		}

		return cgroup.CleanCgroups()
	},
}
