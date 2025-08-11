package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func InitStateDir() error {
	if err := os.MkdirAll(containeruntimeStateDir, 0755); err != nil {
		return ErrInitState
	}
	return nil
}

func getStatePath(containerId string) string {
	return filepath.Join(containeruntimeStateDir, containerId+".json")
}

func saveState(state *specs.State) error {
	statePath := getStatePath(state.ID)
	tempPath := statePath + ".tmp"

	f, err := os.Create(tempPath)
	if err != nil {
		return ErrStateOperation
	}
	defer f.Close()
	defer os.Remove(tempPath)

	if err := json.NewEncoder(f).Encode(state); err != nil {
		return ErrStateCorrupted
	}

	f.Close()
	return os.Rename(tempPath, statePath)
}

func loadState(containerId string) (*specs.State, error) {
	statePath := getStatePath(containerId)
	state := &specs.State{}

	f, err := os.Open(statePath)
	if err != nil {
		return nil, ErrNotFound
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(state); err != nil {
		return nil, ErrStateCorrupted
	}

	return state, nil
}

func deleteState(containerId string) error {
	statePath := getStatePath(containerId)
	if err := os.Remove(statePath); err != nil {
		return ErrStateOperation
	}
	return nil
}

func listState() ([]*specs.State, error) {
	var states []*specs.State
	files, err := os.ReadDir(containeruntimeStateDir)
	if err != nil {
		return nil, ErrStateOperation
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			containerID := strings.TrimSuffix(file.Name(), ".json")
			state, err := loadState(containerID)
			if err != nil {
				continue
			}
			states = append(states, state)
		}
	}

	return states, nil
}

func newContainerState(id, bundlePath string) (*specs.State, error) {
	state := &specs.State{
		Version:     specs.Version,
		ID:          id,
		Status:      specs.StateCreating,
		Pid:         0,
		Bundle:      bundlePath,
		Annotations: nil,
	}
	return state, nil
}

func setContainerPID(containerId string, pid int) error {
	state, err := loadState(containerId)
	if err != nil {
		return ErrStateCorrupted
	}

	state.Pid = pid
	if err := saveState(state); err != nil {
		return ErrStateOperation
	}

	return nil
}
