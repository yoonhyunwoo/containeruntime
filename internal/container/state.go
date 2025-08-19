package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
)

var (
	containeruntimeStateDir string = "/run/containeruntime"
)

// InitStateDir initializes the state directory for container runtime.
func InitStateDir() error {
	if err := os.MkdirAll(containeruntimeStateDir, 0755); err != nil {
		return fmt.Errorf("container: failed to create state directory: %v", err)
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

	if err := json.NewEncoder(f).Encode(state); err != nil {
		return fmt.Errorf("container: failed to encode state to JSON: %w", err)
	}

	f.Close()

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

func listState(containerID string) ([]*specs.State, error) {
	var states []*specs.State
	files, err := os.ReadDir(containeruntimeStateDir)
	if err != nil {
		return nil, fmt.Errorf("container: failed to list states in directory %s: %w", containeruntimeStateDir, err)
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
