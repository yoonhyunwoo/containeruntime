package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
)

// CgroupManager manages the cgroups for a container.
type CgroupManager struct {
	root          string
	containerName string
	subsystems    []SubSystem
}

// SubSystem represents a cgroup v2 controller.
type SubSystem interface {
	Name() string
	Setup(path string) error
	Clean(path string) error
}

// NewCgroupManager creates a new CgroupManager for a given container name.
func NewCgroupManager(containerName string, subsystems []SubSystem) *CgroupManager {
	return &CgroupManager{
		root:          "/sys/fs/cgroup",
		containerName: containerName,
		subsystems:    subsystems,
	}
}

// Setup creates the cgroup hierarchy and configures all subsystems.
func (m *CgroupManager) Setup() error {
	containerCgroup := filepath.Join(m.root, m.containerName)
	if err := os.Mkdir(containerCgroup, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("cgroup: failed to create container cgroup: %w", err)
	}

	var controllers []string
	for _, s := range m.subsystems {
		controllers = append(controllers, "+"+s.Name())
	}
	if len(controllers) > 0 {
		ctrl := []byte(fmt.Sprintf("%s", controllers))
		if err := os.WriteFile(filepath.Join(containerCgroup, "cgroup.subtree_control"), ctrl, 0700); err != nil {
			return fmt.Errorf("cgroup: failed to set controllers: %w", err)
		}
	}

	for _, s := range m.subsystems {
		if err := s.Setup(containerCgroup); err != nil {
			return fmt.Errorf("cgroup: subsystem %s setup failed: %w", s.Name(), err)
		}
	}
	return nil
}

// Clean removes the cgroup hierarchy and cleans up all subsystems.
func (m *CgroupManager) Clean() error {
	containerCgroup := filepath.Join(m.root, m.containerName)
	for _, s := range m.subsystems {
		if err := s.Clean(containerCgroup); err != nil {
			return fmt.Errorf("cgroup: subsystem %s clean failed: %w", s.Name(), err)
		}
	}
	if err := os.Remove(containerCgroup); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cgroup: failed to remove container cgroup: %w", err)
	}
	return nil
}
