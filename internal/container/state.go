package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

const containeruntimeStateDir = "/run/containeruntime"

// InitStateDir initializes the state directory for container runtime.
func InitStateDir() error {
	if err := os.MkdirAll(containeruntimeStateDir, 0755); err != nil {
		return fmt.Errorf("container: failed to create state directory: %w", err)
	}
	return nil
}

func getStatePath(containerID string) string {
	return filepath.Join(containeruntimeStateDir, containerID+".json")
}

func saveState(state *specs.State) error {
	statePath := getStatePath(state.ID)
	tempPath := statePath + ".tmp"

	f, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("container: failed to create temporary state file: %w", err)
	}
	defer f.Close()
	defer os.Remove(tempPath)

	encodeErr := json.NewEncoder(f).Encode(state)
	if encodeErr != nil {
		return fmt.Errorf("container: failed to encode state to JSON: %w", encodeErr)
	}

	closeErr := f.Close()
	if closeErr != nil {
		return fmt.Errorf("container: failed to close temporary state file: %w", closeErr)
	}

	if err = os.Rename(tempPath, statePath); err != nil {
		return fmt.Errorf("container: failed to rename temporary file: %w", err)
	}
	return nil
}

func loadState(containerID string) (*specs.State, error) {
	statePath := getStatePath(containerID)
	state := &specs.State{}

	f, err := os.Open(statePath)
	if err != nil {
		return nil, fmt.Errorf("container: failed to open state file for container %s: %w", containerID, err)
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(state); err != nil {
		return nil, fmt.Errorf("container: failed to decode state file for container %s: %w", containerID, err)
	}

	return state, nil
}

func deleteState(containerID string) error {
	statePath := getStatePath(containerID)
	if err := os.Remove(statePath); err != nil {
		return fmt.Errorf("container: failed to delete state file for container %s: %w", containerID, err)
	}
	return nil
}

func newContainerState(id, bundlePath string) *specs.State {
	state := &specs.State{
		Version:     specs.Version,
		ID:          id,
		Status:      specs.StateCreating,
		Pid:         0,
		Bundle:      bundlePath,
		Annotations: nil,
	}
	return state
}

func SetContainerState(containerID string, state *specs.State) error {
	if err := saveState(state); err != nil {
		return fmt.Errorf("container: failed to update state for container %s: %w", containerID, err)
	}
	return nil
}
