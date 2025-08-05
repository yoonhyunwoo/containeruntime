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

func Create() {

	state, _ := newContainerState("id", "/rootfs/ubuntu")
	err := saveState(state)
	if err != nil {
		log.Println(err)
	}

	fmt.Printf("Running: %v\n", os.Args[2:])

	selfExe, err := os.Executable()
	Must(err)

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

	Must(cmd.Start())

	state.Pid = cmd.Process.Pid
	state.Status = specs.StateCreated
	saveState(state)
}

func Start(containerId string) error {
	state, err := loadState(containerId)
	if err != nil {
		fmt.Printf("container : %v\n", err)
		return fmt.Errorf("container : %v", err)
	}
	state.Status = specs.StateRunning
	syscall.Kill(state.Pid, syscall.SIGCONT)
	saveState(state)
	return nil
}

func State(containerId string) (*specs.State, error) {
	state, err := loadState(containerId)
	if err != nil {
		fmt.Printf("container : %v\n", err)
	}
	return state, err
}

}

func Init() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGCONT)
	<-ch

	fmt.Printf("Running: %v\n", os.Args[2:])

	cgroup.SetupCgroups()

	Must(syscall.Sethostname([]byte("container")))

	const rootfs = "/root/ubuntufs"
	Must(syscall.Chroot(rootfs))
	Must(os.Chdir("/"))

	Must(syscall.Mount("proc", "proc", "proc", 0, ""))
	Must(syscall.Mount("tmpfs", "mytemp", "tmpfs", 0, ""))

	defer Must(syscall.Unmount("proc", 0))
	defer Must(syscall.Unmount("mytemp", 0))

	if len(os.Args) < 3 {
		log.Fatal("Usage: containeruntime")
	}
	syscall.Exec(os.Args[2], os.Args[3:], os.Environ())
}

func Delete(containerId string) error {
	_ = Kill(containerId, syscall.SIGKILL)
	for range 5 {
		time.Sleep(1 * time.Second)
		if err := Kill(containerId, 0); err != nil {
			return deleteState(containerId)
		}
	}
	return errors.New("The container is still running")
}

func Must(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
