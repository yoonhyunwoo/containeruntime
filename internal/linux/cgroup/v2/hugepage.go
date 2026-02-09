package cgroup

import (
	"fmt"
	"strconv"
)

type HugepageSubSystem struct {
	Pages map[string]uint64 // map of hugepage size to limit value
}

func (h *HugepageSubSystem) Name() string {
	return "hugetlb"
}

// Setup applies hugepage subsystem limits.
func (h *HugepageSubSystem) Setup(path string) error {
	for pageSize, limit := range h.Pages {
		filename := "hugetlb." + pageSize + ".max"
		if err := writeCgroupFile(path, filename, strconv.FormatUint(limit, 10)); err != nil {
			return fmt.Errorf("hugetlb subsystem: failed to set %s: %w", filename, err)
		}
	}
	return nil
}
