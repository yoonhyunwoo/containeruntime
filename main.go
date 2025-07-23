package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go run <cmd> <args>")
	}

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		log.Fatal("bad command")
	}
}

func run() {
	fmt.Printf("Running: %v\n", os.Args[2:])

	selfExe, err := os.Executable()
	must(err)

	cmd := exec.Command(selfExe, append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	must(cmd.Run())
}

func child() {
	fmt.Printf("Running: %v\n", os.Args[2:])

	setupCgroups()

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(syscall.Sethostname([]byte("container")))

	const rootfs = "/root/ubuntufs"
	must(syscall.Chroot(rootfs))
	must(os.Chdir("/"))

	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	must(syscall.Mount("tmpfs", "mytemp", "tmpfs", 0, ""))

	defer must(syscall.Unmount("proc", 0))
	defer must(syscall.Unmount("mytemp", 0))

	must(cmd.Run())
}
func setupCgroups() {
	cgroupRoot := "/sys/fs/cgroup"
	containerCgroup := filepath.Join(cgroupRoot, "gamap-container")
	processCgroup := filepath.Join(containerCgroup, "processes")

	err := os.Mkdir(containerCgroup, 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatalln(err)
	}

	err = os.WriteFile(filepath.Join(containerCgroup, "cgroup.subtree_control"), []byte("+pids +memory"), 0700)
	if err != nil {
		log.Println(err)
	}

	err = os.Mkdir(processCgroup, 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatalln(err)
	}

	err = os.WriteFile(filepath.Join(processCgroup, "pids.max"), []byte("20"), 0700)
	if err != nil {
		log.Println(err)
	}

	err = os.WriteFile(filepath.Join(processCgroup, "memory.max"), []byte("100M"), 0700)
	if err != nil {
		log.Println(err)
	}

	pid := strconv.Itoa(os.Getpid())
	err = os.WriteFile(filepath.Join(processCgroup, "cgroup.procs"), []byte(pid), 0700)
	if err != nil {
		log.Println(err)
	}
}

func must(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
