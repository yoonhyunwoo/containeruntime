package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
)

// memory subsystem with configurable limit
type MemorySubSystem struct {
	Limit int64
}

func NewMemorySubSystem(limit int64) *MemorySubSystem {
	return &MemorySubSystem{Limit: limit}
}

func (m *MemorySubSystem) Name() string {
	return "memory"
}

func (m *MemorySubSystem) Setup(path string) error {
	memoryMax := filepath.Join(path, "memory.max")
	if err := os.WriteFile(memoryMax, []byte(fmt.Sprintf("%d", m.Limit)), 0700); err != nil {
		return fmt.Errorf("memory subsystem: failed to set memory.max: %w", err)
	}
	return nil
}

func (m *MemorySubSystem) Clean(path string) error {
	memoryMax := filepath.Join(path, "memory.max")
	if err := os.WriteFile(memoryMax, []byte("max"), 0700); err != nil {
		return fmt.Errorf("memory subsystem: failed to reset memory.max: %w", err)
	}
	return nil
}
