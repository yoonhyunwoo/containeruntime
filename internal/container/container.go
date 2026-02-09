package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/yoonhyunwoo/containeruntime/internal/linux/cgroup/v2"
	"github.com/yoonhyunwoo/containeruntime/internal/linux/pty"
)

// Create initializes a new container with the given ID and root filesystem path.
func Create(containerID, bundlePath string) error {
	absBundlePath, absErr := filepath.Abs(bundlePath)
	if absErr != nil {
		return fmt.Errorf("container: failed to get absolute path for bundle: %w", absErr)
	}
	bundlePath = absBundlePath

	configPath := filepath.Join(bundlePath, "config.json")

	state := newContainerState(containerID, bundlePath)

	spec, specErr := loadSpec(configPath)
	if specErr != nil {
		return specErr
	}

	saveErr := saveState(state)
	if saveErr != nil {
		return fmt.Errorf("container: failed to save initial state: %w", saveErr)
	}
	cgroupSubSystems := createCgroupSubSystems(spec)
	cgroupManager := cgroup.NewCgroupManager(containerID, cgroupSubSystems)
	_ = cgroupManager.Setup()
	setupErr := cgroupManager.Setup()
	if setupErr != nil {
		return fmt.Errorf("container: failed to setup cgroups: %w", setupErr)
	}

	selfExe, exeErr := os.Executable()
	if exeErr != nil {
		return fmt.Errorf("container: failed to get executable path: %w", exeErr)
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

	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		return fmt.Errorf("container: failed to create pipe: %w", pipeErr)
	}
	defer w.Close()

	cmd.ExtraFiles = []*os.File{r}

	if spec.Process.Terminal {
		master, slave, ptyErr := pty.PtyPair()
		if ptyErr != nil {
			return fmt.Errorf("container: failed to create pty pair: %w", ptyErr)
		}
		cmd.Stdin = slave
		cmd.Stdout = slave
		cmd.Stderr = slave
		cmd.SysProcAttr.Setctty = true
		cmd.SysProcAttr.Setsid = true
		state.Annotations = map[string]string{
			"containeruntime/pty-master": fmt.Sprintf("%d", master.Fd()),
			"containeruntime/pty-slave":  slave.Name(),
		}

		startErr := cmd.Start()
		if startErr != nil {
			return fmt.Errorf("container: failed to start command: %w", startErr)
		}

		return errors.New("container: terminal mode is not supported yet")
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	startErr := cmd.Start()
	if startErr != nil {
		return fmt.Errorf("container: failed to start command: %w", startErr)
	}

	encodeErr := json.NewEncoder(w).Encode(&spec)
	if encodeErr != nil {
		return fmt.Errorf("container: failed to encode spec: %w", encodeErr)
	}

	state.Pid = cmd.Process.Pid
	state.Status = specs.StateCreated

	saveErr = saveState(state)
	if saveErr != nil {
		return fmt.Errorf("container: failed to update state with PID: %w", saveErr)
	}

	return nil
}

// Start starts the container with the given ID.
func Start(containerID string) error {
	state, loadErr := loadState(containerID)
	if loadErr != nil {
		return fmt.Errorf("container: failed to start container %s: %w", containerID, loadErr)
	}

	state.Status = specs.StateRunning

	killErr := syscall.Kill(state.Pid, syscall.SIGCONT)
	if killErr != nil {
		state.Status = specs.StateStopped
		_ = saveState(state)
		return fmt.Errorf("container: failed to send SIGCONT to PID %d: %w", state.Pid, killErr)
	}

	saveErr := saveState(state)
	if saveErr != nil {
		state.Status = specs.StateStopped
		return fmt.Errorf("container: failed to start container %s: %w", containerID, saveErr)
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
func Kill(containerID string, sig syscall.Signal) error {
	state, loadErr := loadState(containerID)
	if loadErr != nil {
		return loadErr
	}

	killErr := syscall.Kill(state.Pid, sig)
	if killErr != nil {
		return fmt.Errorf("container: failed to send signal %d to container %s with PID %d: %w", sig, containerID, state.Pid, killErr)
	}

	return nil
}

// Init initializes the container environment.
func Init() {
	pipe := os.NewFile(3, "pipe")
	if pipe == nil {
		log.Fatalf("container: failed to create pipe")
	}

	var spec specs.Spec
	decodeErr := json.NewDecoder(pipe).Decode(&spec)
	if decodeErr != nil {
		log.Fatalf("container: failed to decode spec: %v", decodeErr)
	}
	closeErr := pipe.Close()
	if closeErr != nil {
		log.Printf("container: failed to close pipe: %v", closeErr)
	}

	ch := make(chan os.Signal, 1)

	signal.Notify(ch, syscall.SIGCONT)
	<-ch

	if spec.Hostname != "" {
		setHostErr := syscall.Sethostname([]byte(spec.Hostname))
		if setHostErr != nil {
			log.Fatalf("container: failed to set hostname: %v", setHostErr)
		}
	}

	rootfs := spec.Root.Path

	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		log.Fatalf("container: failed to remount / as private: %v", err)
	}

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
	}

	if err := syscall.Exec(spec.Process.Args[0], spec.Process.Args, os.Environ()); err != nil {
		log.Fatalf("container: failed to exec command %s: %v", spec.Process.Args[0], err)
	}
}

// Delete removes the container with the given ID.
func Delete(containerID string) error {
	killErr := Kill(containerID, syscall.SIGKILL)
	if killErr != nil {
		return deleteState(containerID)
	}
	for range 5 {
		time.Sleep(1 * time.Second)
		checkErr := Kill(containerID, 0)
		if checkErr != nil {
			return deleteState(containerID)
		}
	}
	return fmt.Errorf("container: the container %s is stil running: %w", containerID, killErr)
}
