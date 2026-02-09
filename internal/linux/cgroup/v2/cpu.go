package cgroup

import (
	"fmt"
	"strconv"
)

// CPUSubSystem is a struct that holds settings and statistics for the CPU controller in cgroup v2.
type CPUSubSystem struct {
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

// CPUPressure represents pressure stall information (PSI) from the cpu.pressure file.
type CPUPressure struct {
	Some PressureStall // Some pressure stall information
	Full PressureStall // Full pressure stall information
}

func (c *CPUSubSystem) Name() string {
	return "cpu"
}

func (c *CPUSubSystem) Setup(path string) error {
	files := []CgroupFile{
		{"cpu.weight", strconv.FormatUint(c.Weight, 10)},
		{"cpu.max", fmt.Sprintf("%d %d", c.Quota, c.Period)},
		{"cpu.max.burst", strconv.FormatUint(c.MaxBurst, 10)},
		{"cpu.idle", strconv.FormatInt(c.Idle, 10)},
	}

	for _, f := range files {
		if err := writeCgroupFile(path, f.Filename, f.Value); err != nil {
			return fmt.Errorf("cpu subsystem: failed to set %s: %w", f.Filename, err)
		}
	}

	return nil
}
