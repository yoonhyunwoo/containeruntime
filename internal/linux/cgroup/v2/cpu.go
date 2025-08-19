package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
)

// CpuSubSystem is a struct that holds settings and statistics for the CPU controller in cgroup v2.
type CpuSubSystem struct {
	// cpu.max: Sets the CPU bandwidth limit for the group.
	// Quota corresponds to the $MAX value; -1 means 'max' (unlimited).
	Quota int64
	// Period corresponds to the $PERIOD value.
	Period uint64

	// cpu.weight: CPU time distribution weight (1 ~ 10000)
	Weight uint64

	// cpu.max.burst: Additional CPU burst time available within the period
	MaxBurst uint64

	// cpu.idle: Sets the cgroup to idle state (0 or 1)
	Idle int64
}

// PressureStall represents pressure stall information (PSI) for a specific resource.
type PressureStall struct {
	Avg10  float64 // Average over last 10 seconds
	Avg60  float64 // Average over last 60 seconds
	Avg300 float64 // Average over last 300 seconds (5 minutes)
	Total  uint64  // Accumulated time (microseconds)
}

// CpuPressure represents pressure stall information (PSI) from the cpu.pressure file.
type CpuPressure struct {
	Some PressureStall // Some pressure stall information
	Full PressureStall // Full pressure stall information
}

func (c *CpuSubSystem) Name() string {
	return "cpu"
}

func (c *CpuSubSystem) Setup(path string) error {
	cpuWeight := filepath.Join(path, "cpu.weight")
	cpuMax := filepath.Join(path, "cpu.max")
	cpuMaxBurst := filepath.Join(path, "cpu.max.burst")
	cpuIdle := filepath.Join(path, "cpu.idle")

	os.WriteFile(cpuWeight, []byte(fmt.Sprintf("%d", c.Weight)), 0700)
	os.WriteFile(cpuMax, []byte(fmt.Sprintf("%d %d", c.Quota, c.Period)), 0700)
	os.WriteFile(cpuMaxBurst, []byte(fmt.Sprintf("%d", c.MaxBurst)), 0700)
	os.WriteFile(cpuIdle, []byte(fmt.Sprintf("%d", c.Idle)), 0700)

	return nil
}

func (c *CpuSubSystem) Clean(path string) error {
	pidsMax := filepath.Join(path, "pids.max")
	if err := os.WriteFile(pidsMax, []byte("max"), 0700); err != nil {
		return fmt.Errorf("pids subsystem: failed to reset pids.max: %w", err)
	}
	return nil
}
