package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func SetupCgroups() error {
	cgroupRoot := "/sys/fs/cgroup"
	containerCgroup := filepath.Join(cgroupRoot, "gamap-container")
	processCgroup := filepath.Join(containerCgroup, "processes")

	if err := os.Mkdir(containerCgroup, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("cgroup: failed to create container cgroup directory %s: %w", containerCgroup, err)
	}

	if err := os.WriteFile(filepath.Join(containerCgroup, "cgroup.subtree_control"), []byte("+pids +memory"), 0700); err != nil {
		return fmt.Errorf("cgroup: failed to set cgroup controllers: %w", err)
	}

	if err := os.Mkdir(processCgroup, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("cgroup: failed to create process cgroup directory %s: %w", processCgroup, err)
	}

	if err := os.WriteFile(filepath.Join(processCgroup, "pids.max"), []byte("20"), 0700); err != nil {
		return fmt.Errorf("cgroup: failed to set pids.max: %w", err)
	}

	if err := os.WriteFile(filepath.Join(processCgroup, "memory.max"), []byte("100M"), 0700); err != nil {
		return fmt.Errorf("cgroup: failed to set memory.max: %w", err)
	}

	pid := strconv.Itoa(os.Getpid())
	if err := os.WriteFile(filepath.Join(processCgroup, "cgroup.procs"), []byte(pid), 0700); err != nil {
		return fmt.Errorf("cgroup: failed to add process %s to cgroup: %w", pid, err)
	}

	return nil
}

func CleanCgroups() error {
	cgroupRoot := "/sys/fs/cgroup"
	processCgroup := filepath.Join(cgroupRoot, "gamap-container", "processes")

	if _, err := os.Stat(processCgroup); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("cgroup: failed to check cgroup directory: %w", err)
	}

	procsFile := filepath.Join(processCgroup, "cgroup.procs")
	if err := os.WriteFile(procsFile, []byte(""), 0700); err != nil {
		return fmt.Errorf("cgroup: failed to remove processes from cgroup: %w", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := os.Remove(processCgroup); err != nil {
		return fmt.Errorf("cgroup: failed to remove process cgroup directory: %w", err)
	}

	return nil
}
