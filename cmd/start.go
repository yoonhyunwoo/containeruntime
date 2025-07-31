package cmd

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"github.com/yoonhyunwoo/containeruntime/internal/container"
)

var StartCommand = &cli.Command{
	Name: "start",
	Action: func(ctx context.Context, command *cli.Command) error {
		if command.Args().Len() != 1 {
			return nil
		}

		containerId := command.Args().First()
		err := container.Start(containerId)
		if err != nil {
			fmt.Printf("Can not start conateinr %s", containerId)
		}
		return nil
	},
}
