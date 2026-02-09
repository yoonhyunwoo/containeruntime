package cgroup

import "fmt"

type RDMASubsystem struct {
	Max     RDMAHCA
	Current RDMAHCA
}

type RDMAHCA struct {
	HCAHandle int64
	HCAObject int64
}

func (n *RDMASubsystem) Name() string {
	return "RDMA"
}

// Setup applies RDMA subsystem limits.
func (n *RDMASubsystem) Setup(path string) error {
	cgroupFiles := []CgroupFile{
		{"rdma.max", fmt.Sprintf("hca_handle=%d hca_object=%d", n.Max.HCAHandle, n.Max.HCAObject)},
		{"rdma.current", fmt.Sprintf("hca_handle=%d hca_object=%d", n.Current.HCAHandle, n.Current.HCAObject)},
	}
	for _, f := range cgroupFiles {
		if err := writeCgroupFile(path, f.Filename, f.Value); err != nil {
			return fmt.Errorf("rdma subsystem: failed to set %s: %w", f.Filename, err)
		}
	}
	return nil
}
