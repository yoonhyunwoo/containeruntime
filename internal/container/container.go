package container

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/yoonhyunwoo/containeruntime/internal/linux/cgroup"
)

func Run() {
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

	Must(cmd.Run())

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

func Must(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
