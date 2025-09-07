package container

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/term"

	"github.com/yoonhyunwoo/containeruntime/internal/linux/cgroup/v2"
	"github.com/yoonhyunwoo/containeruntime/internal/linux/pty"
)

// Create initializes a new container with the given ID and root filesystem path.
func Create(containerID, bundlePath string) error {

	bundlePath, err := filepath.Abs(bundlePath)
	if err != nil {
		return fmt.Errorf("container: failed to get absolute path for bundle: %w", err)
	}

	configPath := filepath.Join(bundlePath, "config.json")

	state := newContainerState(containerID, bundlePath)

	spec, err := loadSpec(configPath)
	if err != nil {
		return err
	}

	if err := saveState(state); err != nil {
		return fmt.Errorf("container: failed to save initial state: %w", err)
	}
	cgroupSubSystems, err := createCgroupSubSystems(spec)
	if err != nil {
		return fmt.Errorf("container: failed to create cgroup subsystems: %w", err)
	}
	cgroupManager := cgroup.NewCgroupManager(containerID, cgroupSubSystems)
	cgroupManager.Setup()
	if err := cgroupManager.Setup(); err != nil {
		return fmt.Errorf("container: failed to setup cgroups: %w", err)
	}

	selfExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("container: failed to get executable path: %w", err)
	}

	cmd := exec.Command(selfExe, append([]string{"init"}, spec.Process.Args...)...)

	var cloneFlags uintptr
	for _, ns := range spec.Linux.Namespaces {
		switch ns.Type {
		case specs.PIDNamespace:
			cloneFlags |= syscall.CLONE_NEWPID
		case specs.UTSNamespace:
			cloneFlags |= syscall.CLONE_NEWUTS
		case specs.MountNamespace:
			cloneFlags |= syscall.CLONE_NEWNS
		case specs.IPCNamespace:
			cloneFlags |= syscall.CLONE_NEWIPC
		case specs.NetworkNamespace:
			cloneFlags |= syscall.CLONE_NEWNET
		case specs.UserNamespace:
			cloneFlags |= syscall.CLONE_NEWUSER
		case specs.CgroupNamespace:
			cloneFlags |= syscall.CLONE_NEWCGROUP
		case specs.TimeNamespace:
			cloneFlags |= syscall.CLONE_NEWTIME
		}
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: cloneFlags,
		Setsid:     spec.Process.Terminal,
		Setctty:    spec.Process.Terminal,
	}

	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("container: failed to create pipe: %w", err)
	}
	defer w.Close()

	cmd.ExtraFiles = []*os.File{r}

	if spec.Process.Terminal {
		ptmx, slavePath, err := pty.NewPty()
		if err != nil {
			return fmt.Errorf("container: failed to create pty: %w", err)
		}
		defer ptmx.Close()

		slave, err := os.OpenFile(slavePath, os.O_RDWR|syscall.O_NOCTTY, 0)
		if err != nil {
			return fmt.Errorf("container: failed to open slave pty: %w", err)
		}
		defer slave.Close()

		cmd.Stdin = slave
		cmd.Stdout = slave
		cmd.Stderr = slave
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("container: failed to start command: %w", err)
		}

		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("container: failed to set terminal to raw mode: %w", err)
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)
		go pty.HandleResize(ptmx)
		pty.HandleResize(ptmx)

		go io.Copy(ptmx, os.Stdin)
		io.Copy(os.Stdout, ptmx)
		cmd.Wait()
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("container: failed to start command: %w", err)
		}
	}

	if err := json.NewEncoder(w).Encode(&spec); err != nil {
		return fmt.Errorf("container: failed to encode spec: %w", err)
	}

	state.Pid = cmd.Process.Pid
	state.Status = specs.StateCreated

	if err := saveState(state); err != nil {
		return fmt.Errorf("container: failed to update state with PID: %w", err)
	}

	return nil
}

// Start starts the container with the given ID.
func Start(containerID string) error {
	state, err := loadState(containerID)
	if err != nil {
		return fmt.Errorf("container: failed to start container %s: %w", containerID, err)
	}

	state.Status = specs.StateRunning

	if err := syscall.Kill(state.Pid, syscall.SIGCONT); err != nil {
		state.Status = specs.StateStopped
		saveState(state)
		return fmt.Errorf("container: failed to send SIGCONT to PID %d: %w", state.Pid, err)
	}

	if err := saveState(state); err != nil {
		state.Status = specs.StateStopped
		return fmt.Errorf("container: failed to start container %s: %w", containerID, err)
	}

	return nil
}

// State returns the current state of the container with the given ID.
func State(containerID string) (*specs.State, error) {
	state, err := loadState(containerID)
	if err != nil {
		return nil, err
	}
	return state, nil
}

// Kill stops and removes the container with the given ID.
func Kill(containerID string, signal syscall.Signal) error {
	state, err := loadState(containerID)
	if err != nil {
		return err
	}

	if err := syscall.Kill(state.Pid, signal); err != nil {
		return fmt.Errorf("container: failed to send signal %d to container %s with PID %d: %w", signal, containerID, state.Pid, err)
	}

	return nil
}

// Init initializes the container environment.
func Init() {
	pipe := os.NewFile(3, "pipe")
	if pipe == nil {
		log.Fatalf("container: failed to create pipe")
	}
	defer pipe.Close()

	var spec specs.Spec
	if err := json.NewDecoder(pipe).Decode(&spec); err != nil {
		log.Fatalf("container: failed to decode spec: %v", err)
	}

	ch := make(chan os.Signal, 1)

	signal.Notify(ch, syscall.SIGCONT)
	<-ch

	if spec.Hostname != "" {
		if err := syscall.Sethostname([]byte(spec.Hostname)); err != nil {
			log.Fatalf("container: failed to set hostname: %v", err)
		}
	}

	rootfs := spec.Root.Path

	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		log.Fatalf("container: failed to bind mount rootfs: %v", err)
	}

	pivotDir := filepath.Join(rootfs, ".old_root")

	if err := os.MkdirAll(pivotDir, 0755); err != nil {
		log.Fatalf("container: failed to create pivot directory %s: %v", pivotDir, err)
	}

	if err := syscall.PivotRoot(rootfs, pivotDir); err != nil {
		log.Fatalf("container: failed to pivot root to %s: %v", rootfs, err)
	}

	if err := os.Chdir("/"); err != nil {
		log.Fatalf("container: failed to change directory to /: %v", err)
	}

	if err := syscall.Unmount("/.old_root", syscall.MNT_DETACH); err != nil {
		log.Fatalf("container: failed to unmount old root: %v", err)
	}

	if err := os.RemoveAll("/.old_root"); err != nil {
		log.Fatalf("container: failed to remove old root directory: %v", err)
	}

	for _, m := range spec.Mounts {
		if err := os.MkdirAll(m.Destination, 0755); err != nil {
			log.Fatalf("container: failed to create mount destination %s: %v", m.Destination, err)
		}

		if err := syscall.Mount(m.Source, m.Destination, m.Type, 0, ""); err != nil {
			log.Fatalf("container: failed to mount %s: %v", m.Destination, err)
		}
		defer func(dest string) {
			if err := syscall.Unmount(dest, 0); err != nil {
				log.Printf("container: failed to unmount %s: %v", dest, err)
			}
		}(m.Destination)
	}

	if err := syscall.Exec(spec.Process.Args[0], spec.Process.Args, os.Environ()); err != nil {
		log.Fatalf("container: failed to exec command %s: %v", spec.Process.Args[0], err)
	}
}

// Delete removes the container with the given ID.
func Delete(containerID string) error {
	err := Kill(containerID, syscall.SIGKILL)
	if err != nil {
		return deleteState(containerID)
	}
	for range 5 {
		time.Sleep(1 * time.Second)
		if err := Kill(containerID, 0); err != nil {
			return deleteState(containerID)
		}
	}
	return fmt.Errorf("container: the container %s is stil running: %w", containerID, err)
}
