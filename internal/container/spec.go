package container

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/yoonhyunwoo/containeruntime/internal/linux/cgroup/v2"
)

func loadSpec(specPath string) (*specs.Spec, error) {
	var spec specs.Spec
	f, err := os.Open(specPath)
	if err != nil {
		return nil, fmt.Errorf("container: failed to open spec file at %s: %w", specPath, err)
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&spec)
	if err != nil {
		return nil, fmt.Errorf("container: failed to decode spec JSON from %s: %w", specPath, err)
	}

	return &spec, nil
}

func createCgroupSubSystems(spec *specs.Spec) []cgroup.SubSystem {
	var subSystems []cgroup.SubSystem
	if spec.Linux == nil || spec.Linux.Resources == nil {
		return nil
	}
	if spec.Linux.Resources.Memory != nil {
		mem := spec.Linux.Resources.Memory
		oomGroup := int64(0)
		if *mem.DisableOOMKiller {
			oomGroup = int64(1)
		}
		memorySubSys := &cgroup.MemorySubSystem{
			Max:      *mem.Limit,
			High:     *mem.Reservation,
			SwapHigh: *mem.Swap,
			OOMGroup: oomGroup,
		}
		subSystems = append(subSystems, memorySubSys)
	}

	if spec.Linux.Resources.CPU != nil {
		cpu := spec.Linux.Resources.CPU
		cpuSubSys := &cgroup.CPUSubSystem{
			Quota:    *cpu.Quota,
			Period:   *cpu.Period,
			Idle:     *cpu.Idle,
			Weight:   *cpu.Shares,
			MaxBurst: *cpu.Burst,
		}
		subSystems = append(subSystems, cpuSubSys)
	}

	if spec.Linux.Resources.Pids != nil {
		pidsSubSys := &cgroup.PidsSubSystem{
			MaxPids: spec.Linux.Resources.Pids.Limit,
		}
		subSystems = append(subSystems, pidsSubSys)
	}

	if spec.Linux.Resources.Rdma != nil {
		rdma := spec.Linux.Resources.Rdma

		rdmaSubSys := &cgroup.RDMASubsystem{
			Max: cgroup.RDMAHCA{
				HCAHandle: int64(*rdma["max"].HcaHandles),
				HCAObject: int64(*rdma["max"].HcaObjects),
			},
			Current: cgroup.RDMAHCA{
				HCAHandle: int64(*rdma["current"].HcaHandles),
				HCAObject: int64(*rdma["current"].HcaObjects),
			},
		}
		subSystems = append(subSystems, rdmaSubSys)
	}

	if spec.Linux.Resources.HugepageLimits != nil {
		for _, hugepage := range spec.Linux.Resources.HugepageLimits {
			hugepageSubSys := &cgroup.HugepageSubSystem{
				Pages: map[string]int64{
					hugepage.Pagesize: int64(hugepage.Limit),
				},
			}
			subSystems = append(subSystems, hugepageSubSys)
		}

	}

	return subSystems
}
