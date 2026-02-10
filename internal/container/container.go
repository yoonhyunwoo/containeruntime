package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/yoonhyunwoo/containeruntime/internal/linux/cgroup/v2"
	"github.com/yoonhyunwoo/containeruntime/internal/linux/pty"
	linuxtty "github.com/yoonhyunwoo/containeruntime/internal/linux/term"
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

	// #nosec G204 -- self executable path is resolved from os.Executable and invoked intentionally.
	// Note: do not append spec.Process.Args here; init reads spec from the pipe and execs it.
	cmd := exec.CommandContext(context.Background(), selfExe, "init")

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
		restoreTerminal, rawErr := linuxtty.EnterRawMode(int(os.Stdin.Fd()))
		if rawErr != nil {
			return rawErr
		}
		defer func() {
			if err := restoreTerminal(); err != nil {
				log.Printf("container: failed to restore terminal mode: %v", err)
			}
		}()

		master, slave, ptyErr := pty.PtyPair()
		if ptyErr != nil {
			return fmt.Errorf("container: failed to create pty pair: %w", ptyErr)
		}
		defer master.Close()
		defer slave.Close()

		// Keep a copy of the pty master in the child process so `start` can
		// attach via /proc/<pid>/fd/<n> when invoked from a real terminal.
		cmd.ExtraFiles = append(cmd.ExtraFiles, master)

		if resizeErr := linuxtty.SyncWinsizeFromTerminal(int(os.Stdin.Fd()), int(slave.Fd())); resizeErr != nil {
			return resizeErr
		}

		resizeSignal := make(chan os.Signal, 1)
		signal.Notify(resizeSignal, syscall.SIGWINCH)
		defer signal.Stop(resizeSignal)
		done := make(chan struct{})
		defer close(done)
		go func() {
			for {
				select {
				case <-done:
					return
				case <-resizeSignal:
					if err := linuxtty.SyncWinsizeFromTerminal(int(os.Stdin.Fd()), int(slave.Fd())); err != nil {
						log.Printf("container: failed to sync terminal size: %v", err)
					}
				}
			}
		}()

		cmd.Stdin = slave
		cmd.Stdout = slave
		cmd.Stderr = slave
		cmd.SysProcAttr.Setctty = true
		cmd.SysProcAttr.Setsid = true
		state.Annotations = map[string]string{
			"containeruntime/pty-master-fd": strconv.Itoa(3 + len(cmd.ExtraFiles) - 1),
			"containeruntime/pty-master":    fmt.Sprintf("%d", master.Fd()),
			"containeruntime/pty-slave":     slave.Name(),
		}

		startErr := cmd.Start()
		if startErr != nil {
			return fmt.Errorf("container: failed to start command: %w", startErr)
		}
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		startErr := cmd.Start()
		if startErr != nil {
			return fmt.Errorf("container: failed to start command: %w", startErr)
		}
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

func shouldAttachTerminal(state *specs.State) bool {
	if state == nil || state.Annotations == nil {
		return false
	}
	_, hasMasterFD := state.Annotations["containeruntime/pty-master-fd"]
	return hasMasterFD
}

// Attach attaches the current terminal to a terminal-mode container (if supported).
// If stdin/stdout are not TTYs, this is a no-op.
func Attach(containerID string) error {
	state, loadErr := loadState(containerID)
	if loadErr != nil {
		return fmt.Errorf("container: failed to attach container %s: %w", containerID, loadErr)
	}
	if !shouldAttachTerminal(state) {
		return nil
	}
	return attachTerminal(state)
}

// Run is a convenience helper: create + start + (tty) attach.
func Run(containerID, bundlePath string) error {
	if err := Create(containerID, bundlePath); err != nil {
		return err
	}
	if err := Start(containerID); err != nil {
		return err
	}
	if err := Attach(containerID); err != nil {
		return err
	}
	return nil
}

func attachTerminal(state *specs.State) error {
	if !linuxtty.IsTerminal(int(os.Stdin.Fd())) || !linuxtty.IsTerminal(int(os.Stdout.Fd())) {
		return nil
	}

	masterFD := state.Annotations["containeruntime/pty-master-fd"]
	masterPath := fmt.Sprintf("/proc/%d/fd/%s", state.Pid, masterFD)
	master, err := os.OpenFile(masterPath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("container: failed to open terminal attach path %s: %w", masterPath, err)
	}
	defer master.Close()

	restoreTerminal, rawErr := linuxtty.EnterRawMode(int(os.Stdin.Fd()))
	if rawErr != nil {
		return rawErr
	}
	defer func() {
		if err := restoreTerminal(); err != nil {
			log.Printf("container: failed to restore terminal mode: %v", err)
		}
	}()

	if resizeErr := linuxtty.SyncWinsizeFromTerminal(int(os.Stdin.Fd()), int(master.Fd())); resizeErr != nil {
		return resizeErr
	}

	resizeSignal := make(chan os.Signal, 1)
	signal.Notify(resizeSignal, syscall.SIGWINCH)
	defer signal.Stop(resizeSignal)
	done := make(chan struct{})
	defer close(done)

	forwardSignal := make(chan os.Signal, 8)
	signal.Notify(forwardSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGTSTP, syscall.SIGCONT)
	defer signal.Stop(forwardSignal)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-resizeSignal:
				if err := linuxtty.SyncWinsizeFromTerminal(int(os.Stdin.Fd()), int(master.Fd())); err != nil {
					log.Printf("container: failed to sync terminal size: %v", err)
				}
			case sig := <-forwardSignal:
				// Forward to the container process group so interactive signals
				// reach the shell and its children.
				if err := syscall.Kill(-state.Pid, sig.(syscall.Signal)); err != nil && !errors.Is(err, syscall.ESRCH) {
					log.Printf("container: failed to forward signal %v to container pid %d: %v", sig, state.Pid, err)
				}
			}
		}
	}()

	inputErrCh := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(master, os.Stdin)
		inputErrCh <- copyErr
	}()

	_, outputErr := io.Copy(os.Stdout, master)
	inputErr := <-inputErrCh

	if !isIgnorableAttachErr(outputErr) {
		return fmt.Errorf("container: terminal output bridge failed: %w", outputErr)
	}
	if !isIgnorableAttachErr(inputErr) {
		return fmt.Errorf("container: terminal input bridge failed: %w", inputErr)
	}

	return nil
}

func isIgnorableAttachErr(err error) bool {
	if err == nil {
		return true
	}
	return errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) || errors.Is(err, syscall.EIO)
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

	if err := os.MkdirAll(pivotDir, 0o750); err != nil {
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
		if err := os.MkdirAll(m.Destination, 0o750); err != nil {
			log.Fatalf("container: failed to create mount destination %s: %v", m.Destination, err)
		}

		if err := syscall.Mount(m.Source, m.Destination, m.Type, 0, ""); err != nil {
			log.Fatalf("container: failed to mount %s: %v", m.Destination, err)
		}
	}

	// #nosec G204 -- process args are explicitly provided through OCI config and are expected runtime input.
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
