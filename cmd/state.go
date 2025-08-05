package cmd

import (
	"context"
	"encoding/json"
	"fmt"
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
			fmt.Printf("Can not get conateinr %s", containerId)
		}

		containerStateBytes, _ := json.MarshalIndent(containerState, "", " ")
		os.Stdout.Write(containerStateBytes)
		return nil
	},
}
