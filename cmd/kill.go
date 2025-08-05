package cmd

import (
	"context"
	"fmt"
	"strconv"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli/v3"
	"github.com/yoonhyunwoo/containeruntime/internal/container"
)

var KillCommand = &cli.Command{
	Name:      "kill",
	Usage:     "This command sends a specific signal to the main process of a container.",
	ArgsUsage: "<containerid> <signal>",
	Action: func(ctx context.Context, command *cli.Command) error {

		if command.Args().Len() != 2 {
			return nil
		}

		containerId := command.Args().First()

		signalNumber, err := strconv.Atoi((command.Args().Get(1)))
		if err != nil {
			fmt.Printf("Invalid signal number: %s\n", command.Args().Get(1))
			return nil
		}

		containerState, err := container.State(containerId)
		if err != nil {
			fmt.Printf("Unable to get container state: %s\n", err)
			return nil
		}

		if containerState.Status == specs.StateRunning || containerState.Status == specs.StateCreated {
			return fmt.Errorf("You can send a signal only to containers in the running or created state.")
		}

		signal := syscall.Signal(signalNumber)
		err = container.Kill(containerId, signal)
		if err != nil {
			fmt.Printf("Can not start conateinr %s", containerId)
		}
		return nil
	},
}
