package cgroup

import (
	"fmt"
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
		{"pids.max", fmt.Sprintf("%d", p.MaxPids)},
		{"pids.current", fmt.Sprintf("%d", p.Current)},
		{"pids.peak", fmt.Sprintf("%d", p.Peak)},
		{"pids.events", fmt.Sprintf("%d", p.Events)},
		{"pids.events.local", fmt.Sprintf("%d", p.EventsLocal)},
	}

	for _, f := range files {
		if err := writeCgroupFile(path, f.Filename, f.Value); err != nil {
			return fmt.Errorf("pids subsystem: failed to set %s: %w", f.Filename, err)
		}
	}
	return nil
}
