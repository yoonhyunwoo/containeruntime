package container

import (
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

const contaeruntimeStateDir string = "/run/containeruntime"

func initStateDir() error {
	return os.MkdirAll(contaeruntimeStateDir, 0755)
}

func getStatePath(containerId string) string {
	return filepath.Join(contaeruntimeStateDir, containerId+".json")
}

func saveState(state *specs.State) error {
	return nil
}

func loadState(containerId string) (*specs.State, error) {
	return nil, nil
}

func deleteState(containerId string) error {
	return nil
}

func listState(containerId string) ([]*specs.State, error) {
	return nil, nil
}

func newContainerState(id, bundlePath string) (*specs.State, error) {
	return nil, nil
}

func updateContainerState(containerId string, state *specs.State) error {
	return nil
}

func setContainerPID(containerId string, pid int) error {
	return nil
}
