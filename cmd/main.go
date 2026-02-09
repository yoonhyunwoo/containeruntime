package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli/v3"

	"github.com/yoonhyunwoo/containeruntime/internal/container"
	"github.com/yoonhyunwoo/containeruntime/internal/linux/cgroup/v2"
)

func newRootCommand() *cli.Command {
	createCommand := &cli.Command{
		Name:      "create",
		Usage:     "This command creates a new container. You must provide a unique container ID and the path to the bundle containing the container's configuration.",
		ArgsUsage: "<container-id> <path-to-bundle>",
		Action: func(_ context.Context, command *cli.Command) error {
			if command.Args().Len() != 2 {
				return errors.New("main: container-id and path-to-bundle are required")
			}

			containerID := command.Args().Get(0)
			bundlePath := command.Args().Get(1)

			if err := cgroup.SetupCgroups(); err != nil {
				return fmt.Errorf("main: failed to set up cgroups: %w", err)
			}
			if err := container.Create(containerID, bundlePath); err != nil {
				return fmt.Errorf("main: failed to create container: %w", err)
			}

			fmt.Println(containerID)

			return nil
		},
	}

	deleteCommand := &cli.Command{
		Name:      "delete",
		Usage:     "This command deletes a container and its associated resources.",
		ArgsUsage: "<container-id>",
		Action: func(_ context.Context, command *cli.Command) error {
			if command.Args().Len() != 1 {
				return errors.New("main: container-id is required")
			}

			containerID := command.Args().First()
			if err := container.Delete(containerID); err != nil {
				return fmt.Errorf("main: failed to delete container %s: %w", containerID, err)
			}

			if err := cgroup.CleanCgroups(); err != nil {
				return fmt.Errorf("main: failed to clean up cgroups: %w", err)
			}
			return nil
		},
	}

	initCommand := &cli.Command{
		Name: "init",
		Action: func(_ context.Context, _ *cli.Command) error {
			container.Init()
			return nil
		},
	}

	killCommand := &cli.Command{
		Name:      "kill",
		Usage:     "This command sends a specific signal to the main process of a container.",
		ArgsUsage: "<containeriD> <signal>",
		Action: func(_ context.Context, command *cli.Command) error {
			if command.Args().Len() != 2 {
				return errors.New("main: container ID and signal number are required")
			}

			containerID := command.Args().First()

			signalNumber, err := strconv.Atoi((command.Args().Get(1)))
			if err != nil {
				return fmt.Errorf("main: invalid signal number: %w", err)
			}

			containerState, err := container.State(containerID)
			if err != nil {
				return fmt.Errorf("main: failed to get container state: %w", err)
			}

			if containerState.Status != specs.StateRunning && containerState.Status != specs.StateCreated {
				return fmt.Errorf("main: you can only send a signal to containers in the 'running' or 'created' state, but container %s is in state '%s'", containerID, containerState.Status)
			}

			signal := syscall.Signal(signalNumber)
			err = container.Kill(containerID, signal)
			if err != nil {
				return fmt.Errorf("main: failed to kill container %s: %w", containerID, err)
			}
			return nil
		},
	}

	startCommand := &cli.Command{
		Name:      "start",
		Usage:     "This command starts a previously created container. It runs the user-specified program defined in the container's configuration.",
		ArgsUsage: "<container-id>",
		Action: func(_ context.Context, command *cli.Command) error {
			if command.Args().Len() != 1 {
				return errors.New("main: container ID is required")
			}

			containerID := command.Args().First()
			err := container.Start(containerID)
			if err != nil {
				return fmt.Errorf("main: failed to start container %s: %w", containerID, err)
			}
			return nil
		},
	}

	stateCommand := &cli.Command{
		Name:      "state",
		Usage:     "This command returns the current state of a container.",
		ArgsUsage: "<container-id>",
		Action: func(_ context.Context, command *cli.Command) error {
			if command.Args().Len() != 1 {
				return errors.New("main: container ID is required")
			}

			containerID := command.Args().First()
			containerState, err := container.State(containerID)
			if err != nil {
				return fmt.Errorf("main: failed to get container state: %w", err)
			}

			err = container.Kill(containerID, 0)
			if err != nil {
				containerState.Status = specs.StateStopped
				if saveErr := container.SetContainerState(containerID, containerState); saveErr != nil {
					return fmt.Errorf("main: failed to save container state: %w", saveErr)
				}
			}

			containerStateBytes, err := json.MarshalIndent(containerState, "", "  ")
			if err != nil {
				return fmt.Errorf("main: failed to marshal container state to JSON: %w", err)
			}
			_, _ = os.Stdout.Write(containerStateBytes)
			return nil
		},
	}

	return &cli.Command{
		Commands: []*cli.Command{
			createCommand,
			deleteCommand,
			initCommand,
			killCommand,
			startCommand,
			stateCommand,
		},
	}
}

func main() {
	if err := container.InitStateDir(); err != nil {
		log.Fatal(err)
	}

	rootCmd := newRootCommand()

	if err := rootCmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
