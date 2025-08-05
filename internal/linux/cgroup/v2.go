package cgroup

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
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
func CleanCgroups() error {
	cgroupRoot := "/sys/fs/cgroup"
	processCgroup := filepath.Join(cgroupRoot, "gamap-container", "processes")
	if _, err := os.Stat(processCgroup); err == nil {
		procsFile := filepath.Join(processCgroup, "cgroup.procs")
		err = os.WriteFile(procsFile, []byte(""), 0700)
		if err != nil {
			return errors.New("Error removing cgroup processes")
		}
		time.Sleep(100 * time.Millisecond)
		err = os.Remove(processCgroup)
		if err != nil {
			return errors.New("Error removing cgroup processes")
		}
	}

	return nil
}
