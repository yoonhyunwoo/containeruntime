package cgroup

import (
	"fmt"
)

// memory subsystem with configurable limit
type MemorySubSystem struct {
	Min            int64
	Low            int64
	High           int64
	Max            int64
	Peak           int64
	OOMGroup       int64
	SwapHigh       int64
	SwapPeak       int64
	SwapMax        int64
	ZswapMax       int64
	ZswapWriteback int64
}

func NewMemorySubSystem(min, low, high, max, peak, oomGroup, swapHigh, swapPeak, swapMax, zswapMax, zswapWriteback int64) *MemorySubSystem {
	return &MemorySubSystem{
		Min:            min,
		Low:            low,
		High:           high,
		Max:            max,
		Peak:           peak,
		OOMGroup:       oomGroup,
		SwapHigh:       swapHigh,
		SwapPeak:       swapPeak,
		SwapMax:        swapMax,
		ZswapMax:       zswapMax,
		ZswapWriteback: zswapWriteback,
	}
}

func (m *MemorySubSystem) Name() string {
	return "memory"
}

// Setup applies memory subsystem limits
func (m *MemorySubSystem) Setup(path string) error {
	type memFile struct {
		name  string
		value string
	}
	files := []memFile{
		{"memory.min", fmt.Sprintf("%d", m.Min)},
		{"memory.low", fmt.Sprintf("%d", m.Low)},
		{"memory.high", fmt.Sprintf("%d", m.High)},
		{"memory.max", fmt.Sprintf("%d", m.Max)},
		{"memory.peak", fmt.Sprintf("%d", m.Peak)},
		{"memory.oom.group", fmt.Sprintf("%d", m.OOMGroup)},
		{"memory.swap.high", fmt.Sprintf("%d", m.SwapHigh)},
		{"memory.swap.peak", fmt.Sprintf("%d", m.SwapPeak)},
		{"memory.swap.max", fmt.Sprintf("%d", m.SwapMax)},
		{"memory.zswap.max", fmt.Sprintf("%d", m.ZswapMax)},
		{"memory.zswap.writeback", fmt.Sprintf("%d", m.ZswapWriteback)},
	}

	for _, f := range files {
		if err := writeCgroupFile(path, f.name, f.value); err != nil {
			return fmt.Errorf("memory subsystem: failed to set %s: %w", f.name, err)
		}
	}
	return nil
}
