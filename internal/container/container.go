package container

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func Create() error {
	state, err := newContainerState("id", "/rootfs/ubuntu")
	if err != nil {
		return fmt.Errorf("container: failed to create new state: %w", err)
	}

	if err := saveState(state); err != nil {
		return fmt.Errorf("container: failed to save initial state: %w", err)
	}

	fmt.Printf("Running: %v\n", os.Args[2:])

	selfExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("container: failed to get executable path: %w", err)
	}

	cmd := exec.Command(selfExe, append([]string{"init"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("container: failed to start command: %w", err)
	}

	state.Pid = cmd.Process.Pid
	state.Status = specs.StateCreated

	if err := saveState(state); err != nil {
		return fmt.Errorf("container: failed to update state with PID: %w", err)
	}

	return nil
}

func Start(containerId string) error {
	state, err := loadState(containerId)
	if err != nil {
		return fmt.Errorf("container: failed to start container %s: %w", containerId, err)
	}

	state.Status = specs.StateRunning

	if err := syscall.Kill(state.Pid, syscall.SIGCONT); err != nil {
		state.Status = specs.StateStopped
		saveState(state)
		return fmt.Errorf("container: failed to send SIGCONT to PID %d: %w", state.Pid, err)
	}

	if err := saveState(state); err != nil {
		state.Status = specs.StateStopped
		return fmt.Errorf("container: failed to start container %s: %w", containerId, err)
	}

	return nil
}

func State(containerId string) (*specs.State, error) {
	state, err := loadState(containerId)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func Kill(containerId string, signal syscall.Signal) error {
	state, err := loadState(containerId)
	if err != nil {
		return err
	}

	if err := syscall.Kill(state.Pid, signal); err != nil {
		return fmt.Errorf("container: failed to send signal %d to container %s with PID %d: %w", signal, containerId, state.Pid, err)
	}

	return nil
}

func Init() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGCONT)
	<-ch

	fmt.Printf("Running: %v\n", os.Args[2:])

	if err := syscall.Sethostname([]byte("container")); err != nil {
		log.Fatalf("container: failed to set hostname: %v", err)
	}

	const rootfs = "/root/ubuntufs"
	if err := syscall.Chroot(rootfs); err != nil {
		log.Fatalf("container: failed to chroot to %s: %v", rootfs, err)
	}
	if err := os.Chdir("/"); err != nil {
		log.Fatalf("container: failed to change directory to /: %v", err)
	}

	defer func() {
		if err := syscall.Unmount("proc", 0); err != nil {
			log.Printf("container: failed to unmount proc: %v", err)
		}
	}()
	if err := syscall.Mount("proc", "proc", "proc", 0, ""); err != nil {
		log.Fatalf("container: failed to mount proc: %v", err)
	}

	defer func() {
		if err := syscall.Unmount("mytemp", 0); err != nil {
			log.Printf("container: failed to unmount mytemp: %v", err)
		}
	}()
	if err := syscall.Mount("tmpfs", "mytemp", "tmpfs", 0, ""); err != nil {
		log.Fatalf("container: failed to mount tmpfs: %v", err)
	}

	if len(os.Args) < 3 {
		log.Fatal("Usage: containeruntime [command] [args...]")
	}

	if err := syscall.Exec(os.Args[2], os.Args[3:], os.Environ()); err != nil {
		log.Fatalf("container: failed to exec command %s: %v", os.Args[2], err)
	}
}

func Delete(containerId string) error {
	err := Kill(containerId, syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("container: failed to delete container %s: %w", containerId, err)
	}
	for range 5 {
		time.Sleep(1 * time.Second)
		if err := Kill(containerId, 0); err != nil {
			return deleteState(containerId)
		}
	}
	return fmt.Errorf("container: the container %s is stil running: %w", containerId, err)
}
