package cgroup

import (
	"fmt"
	"strconv"
)

type PidsSubSystem struct {
	MaxPids     int64
	Current     int64
	Peak        int64
	Events      int64
	EventsLocal int64
}

func NewPidsSubSystem(maxPids int64) *PidsSubSystem {
	return &PidsSubSystem{MaxPids: maxPids}
}

func (p *PidsSubSystem) Name() string {
	return "pids"
}

func (p *PidsSubSystem) Setup(path string) error {
	files := []CgroupFile{
		{"pids.max", strconv.FormatInt(p.MaxPids, 10)},
		{"pids.current", strconv.FormatInt(p.Current, 10)},
		{"pids.peak", strconv.FormatInt(p.Peak, 10)},
		{"pids.events", strconv.FormatInt(p.Events, 10)},
		{"pids.events.local", strconv.FormatInt(p.EventsLocal, 10)},
	}

	for _, f := range files {
		if err := writeCgroupFile(path, f.Filename, f.Value); err != nil {
			return fmt.Errorf("pids subsystem: failed to set %s: %w", f.Filename, err)
		}
	}
	return nil
}
