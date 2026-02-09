package cgroup

import (
	"fmt"
	"strconv"
)

// MemorySubSystem defines configurable memory limits.
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

func NewMemorySubSystem(minVal, low, high, max, peak, oomGroup, swapHigh, swapPeak, swapMax, zswapMax, zswapWriteback int64) *MemorySubSystem {
	return &MemorySubSystem{
		Min:            minVal,
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

// Setup applies memory subsystem limits.
func (m *MemorySubSystem) Setup(path string) error {
	files := []CgroupFile{
		{"memory.min", strconv.FormatInt(m.Min, 10)},
		{"memory.low", strconv.FormatInt(m.Low, 10)},
		{"memory.high", strconv.FormatInt(m.High, 10)},
		{"memory.max", strconv.FormatInt(m.Max, 10)},
		{"memory.peak", strconv.FormatInt(m.Peak, 10)},
		{"memory.oom.group", strconv.FormatInt(m.OOMGroup, 10)},
		{"memory.swap.high", strconv.FormatInt(m.SwapHigh, 10)},
		{"memory.swap.peak", strconv.FormatInt(m.SwapPeak, 10)},
		{"memory.swap.max", strconv.FormatInt(m.SwapMax, 10)},
		{"memory.zswap.max", strconv.FormatInt(m.ZswapMax, 10)},
		{"memory.zswap.writeback", strconv.FormatInt(m.ZswapWriteback, 10)},
	}

	for _, f := range files {
		if err := writeCgroupFile(path, f.Filename, f.Value); err != nil {
			return fmt.Errorf("memory subsystem: failed to set %s: %w", f.Filename, err)
		}
	}
	return nil
}
