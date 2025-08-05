package container

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"

	"errors"
)

var (
	containeruntimeStateDir string = "/run/containeruntime"

	ErrInitState = errors.New("container: can not init state directory")
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
		log.Println("this?")
		return fmt.Errorf("Can not save state: %v", err)
	}
	defer f.Close()
	defer os.Remove(tempPath)

	if err := json.NewEncoder(f).Encode(state); err != nil {
		return fmt.Errorf("Can not parsing state : %v", err)
	}

	f.Close()
	return os.Rename(tempPath, statePath)
}

func loadState(containerId string) (*specs.State, error) {
	statePath := getStatePath(containerId)
	state := &specs.State{}

	f, err := os.Open(statePath)
	if err != nil {
		return nil, fmt.Errorf("Can not open state: %v", err)
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(state); err != nil {
		return nil, fmt.Errorf("Can not parsing state : %v", err)
	}

	return state, nil
}

func deleteState(containerId string) error {
	statePath := getStatePath(containerId)
	if err := os.Remove(statePath); err != nil {
		return fmt.Errorf("Can not delete state: %v", err)
	}
	return nil
}

func listState(containerId string) ([]*specs.State, error) {
	var states []*specs.State
	files, err := os.ReadDir(containeruntimeStateDir)
	if err != nil {
		return nil, fmt.Errorf("Can not list state: %v", err)
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
		return fmt.Errorf("Can set container pid: %v", err)
	}

	state.Pid = pid
	if err := saveState(state); err != nil {
		return fmt.Errorf("Can set container pid: %v", err)
	}

	return nil
}
