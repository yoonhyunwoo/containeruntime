package cgroup

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func SetupCgroups() {
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
