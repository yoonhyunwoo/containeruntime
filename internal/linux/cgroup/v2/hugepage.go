package cgroup

import "fmt"

type HugepageSubSystem struct {
	Pages map[string]int64 // map of hugepage size to limit value
}

func (h *HugepageSubSystem) Name() string {
	return "hugetlb"
}

// Setup applies hugepage subsystem limits
func (h *HugepageSubSystem) Setup(path string) error {
	for pageSize, limit := range h.Pages {
		filename := "hugetlb." + pageSize + ".max"
		if err := writeCgroupFile(path, filename, fmt.Sprintf("%d", limit)); err != nil {
			return fmt.Errorf("hugetlb subsystem: failed to set %s: %w", filename, err)
		}
	}
	return nil
}
