package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/yoonhyunwoo/containeruntime/internal/container"
)

var StateCommand = &cli.Command{
	Name:      "state",
	Usage:     "This command returns the current state of a container.",
	ArgsUsage: "<container-id>",
	Action: func(ctx context.Context, command *cli.Command) error {
		if command.Args().Len() != 1 {
			return nil
		}

		containerId := command.Args().First()
		containerState, err := container.State(containerId)
		if err != nil {
			return err
		}

		containerStateBytes, err := json.MarshalIndent(containerState, "", " ")
		if err != nil {
			return err
		}
		os.Stdout.Write(containerStateBytes)
		return nil
	},
}
