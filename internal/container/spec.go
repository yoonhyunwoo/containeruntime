package container

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func loadSpec(specPath string) (*specs.Spec, error) {
	var spec specs.Spec
	f, err := os.Open(specPath)
	if err != nil {
		return nil, fmt.Errorf("container: failed to open spec file at %s: %w", specPath, err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&spec); err != nil {
		return nil, fmt.Errorf("container: failed to decode spec JSON from %s: %w", specPath, err)
	}

	return &spec, nil

}
