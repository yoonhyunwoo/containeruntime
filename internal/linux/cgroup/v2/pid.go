package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
)

type PidsSubSystem struct {
	MaxPids int64
}

func NewPidsSubSystem(maxPids int64) *PidsSubSystem {
	return &PidsSubSystem{MaxPids: maxPids}
}

func (p *PidsSubSystem) Name() string {
	return "pids"
}

func (p *PidsSubSystem) Setup(path string) error {
	pidsMax := filepath.Join(path, "pids.max")
	if err := os.WriteFile(pidsMax, []byte(fmt.Sprintf("%d", p.MaxPids)), 0700); err != nil {
		return fmt.Errorf("pids subsystem: failed to set pids.max: %w", err)
	}
	return nil
}

func (p *PidsSubSystem) Clean(path string) error {
	pidsMax := filepath.Join(path, "pids.max")
	if err := os.WriteFile(pidsMax, []byte("max"), 0700); err != nil {
		return fmt.Errorf("pids subsystem: failed to reset pids.max: %w", err)
	}
	return nil
}
