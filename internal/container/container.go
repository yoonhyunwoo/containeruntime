package container

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/yoonhyunwoo/containeruntime/internal/linux/cgroup"
)

var (
	containeruntimeStateDir string = "/run/containeruntime"

	ErrNotFound         = errors.New("container : not found")
	ErrAlreadyExists    = errors.New("container : already exists")
	ErrStateCorrupted   = errors.New("container : state corrupted")
	ErrInitContainer    = errors.New("container : init container failed")
	ErrStillRunning     = errors.New("container : still running")
	ErrInvalidArguments = errors.New("container : invalid arguments")
	ErrStateOperation   = errors.New("container : state operation failed")
	ErrInitState        = errors.New("container : init directory failed")
)

func Create() error {

	state, _ := newContainerState("id", "/rootfs/ubuntu")
	err := saveState(state)
	if err != nil {
		return ErrStateOperation
	}

	fmt.Printf("Running: %v\n", os.Args[2:])

	selfExe, err := os.Executable()
	if err != nil {
		return err
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
		return ErrInitContainer
	}

	state.Pid = cmd.Process.Pid
	state.Status = specs.StateCreated
	return saveState(state)
}

func Start(containerId string) error {
	state, err := loadState(containerId)
	if err != nil {
		return err
	}
	state.Status = specs.StateRunning
	syscall.Kill(state.Pid, syscall.SIGCONT)
	return saveState(state)
}

func State(containerId string) (*specs.State, error) {
	state, err := loadState(containerId)
	if err != nil {
		return nil, ErrNotFound
	}
	return state, nil
}

func Kill(containerId string, signal syscall.Signal) error {
	state, err := loadState(containerId)
	if err != nil {
		return err
	}

	return syscall.Kill(state.Pid, signal)
}

func Init() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGCONT)
	<-ch

	fmt.Printf("Running: %v\n", os.Args[2:])

	if err := cgroup.SetupCgroups(); err != nil {
		log.Fatal(err)
	}

	if err := syscall.Sethostname([]byte("container")); err != nil {
		log.Fatal(err)
	}

	const rootfs = "/root/ubuntufs"
	if err := syscall.Chroot(rootfs); err != nil {
		log.Fatal(err)
	}
	if err := os.Chdir("/"); err != nil {
		log.Fatal(err)
	}

	if err := syscall.Mount("proc", "proc", "proc", 0, ""); err != nil {
		log.Fatal(err)
	}
	if err := syscall.Mount("tmpfs", "mytemp", "tmpfs", 0, ""); err != nil {
		log.Fatal(err)
	}

	defer syscall.Unmount("proc", 0)
	defer syscall.Unmount("mytemp", 0)

	if len(os.Args) < 3 {
		log.Fatal("Usage: containeruntime")
	}
	if err := syscall.Exec(os.Args[2], os.Args[3:], os.Environ()); err != nil {
		log.Fatal(err)
	}
}

func Delete(containerId string) error {
	_ = Kill(containerId, syscall.SIGKILL)
	for range 5 {
		time.Sleep(1 * time.Second)
		if err := Kill(containerId, 0); err != nil {
			return deleteState(containerId)
		}
	}
	return ErrStillRunning
}
