package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli/v3"
	"github.com/yoonhyunwoo/containeruntime/internal/container"
	"github.com/yoonhyunwoo/containeruntime/internal/linux/cgroup"
)

var (
	CreateCommand = &cli.Command{
		Name:      "create",
		Usage:     "This command creates a new container. You must provide a unique container ID and the path to the bundle containing the container's configuration.",
		ArgsUsage: "<container-id> <path-to-bundle>",
		Action: func(ctx context.Context, command *cli.Command) error {
			if err := cgroup.SetupCgroups(); err != nil {
				return fmt.Errorf("main: failed to set up cgroups: %w", err)
			}
			if err := container.Create(); err != nil {
				return fmt.Errorf("main: failed to create container: %w", err)
			}
			return nil
		},
	}

	DeleteCommand = &cli.Command{
		Name:      "delete",
		Usage:     "This command deletes a container and its associated resources.",
		ArgsUsage: "<container-id>",
		Action: func(ctx context.Context, command *cli.Command) error {
			if command.Args().Len() != 1 {
				return fmt.Errorf("main: container-id is required")
			}

			containerId := command.Args().First()
			if err := container.Delete(containerId); err != nil {
				return fmt.Errorf("main: failed to delete container %s: %w", containerId, err)
			}

			if err := cgroup.CleanCgroups(); err != nil {
				return fmt.Errorf("main: failed to clean up cgroups: %w", err)
			}
			return nil
		},
	}

	InitCommand = &cli.Command{
		Name: "init",
		Action: func(ctx context.Context, command *cli.Command) error {
			container.Init()
			return nil
		},
	}

	KillCommand = &cli.Command{
		Name:      "kill",
		Usage:     "This command sends a specific signal to the main process of a container.",
		ArgsUsage: "<containerid> <signal>",
		Action: func(ctx context.Context, command *cli.Command) error {
			if command.Args().Len() != 2 {
				return fmt.Errorf("main: container ID and signal number are required")
			}

			containerId := command.Args().First()

			signalNumber, err := strconv.Atoi((command.Args().Get(1)))
			if err != nil {
				return fmt.Errorf("main: invalid signal number: %w", err)
			}

			containerState, err := container.State(containerId)
			if err != nil {
				return fmt.Errorf("main: failed to get container state: %w", err)
			}

			if containerState.Status != specs.StateRunning && containerState.Status != specs.StateCreated {
				return fmt.Errorf("main: you can only send a signal to containers in the 'running' or 'created' state, but container %s is in state '%s'", containerId, containerState.Status)
			}

			signal := syscall.Signal(signalNumber)
			err = container.Kill(containerId, signal)
			if err != nil {
				return fmt.Errorf("main: failed to kill container %s: %w", containerId, err)
			}
			return nil
		},
	}

	StartCommand = &cli.Command{
		Name:      "start",
		Usage:     "This command starts a previously created container. It runs the user-specified program defined in the container's configuration.",
		ArgsUsage: "<container-id>",
		Action: func(ctx context.Context, command *cli.Command) error {
			if command.Args().Len() != 1 {
				return fmt.Errorf("main: container ID is required")
			}

			containerId := command.Args().First()
			err := container.Start(containerId)
			if err != nil {
				return fmt.Errorf("main: failed to start container %s: %w", containerId, err)
			}
			return nil
		},
	}

	StateCommand = &cli.Command{
		Name:      "state",
		Usage:     "This command returns the current state of a container.",
		ArgsUsage: "<container-id>",
		Action: func(ctx context.Context, command *cli.Command) error {
			if command.Args().Len() != 1 {
				return fmt.Errorf("main: container ID is required")
			}

			containerId := command.Args().First()
			containerState, err := container.State(containerId)
			if err != nil {
				return fmt.Errorf("main: failed to get container state: %w", err)
			}

			containerStateBytes, err := json.MarshalIndent(containerState, "", "  ")
			if err != nil {
				return fmt.Errorf("main: failed to marshal container state to JSON: %w", err)
			}
			os.Stdout.Write(containerStateBytes)
			return nil
		},
	}
)

func main() {
	if err := container.InitStateDir(); err != nil {
		log.Fatal(err)
	}

	rootCmd := &cli.Command{
		Commands: []*cli.Command{
			CreateCommand,
			DeleteCommand,
			InitCommand,
			KillCommand,
			StartCommand,
			StateCommand,
		},
	}

	if err := rootCmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
