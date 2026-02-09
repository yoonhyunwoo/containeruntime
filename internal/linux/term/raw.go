package term

import (
	"fmt"

	xterm "golang.org/x/term"
)

// EnterRawMode puts the given terminal file descriptor into raw mode and
// returns a restore function that should be called to restore the previous mode.
func EnterRawMode(fd int) (func() error, error) {
	if !xterm.IsTerminal(fd) {
		return func() error { return nil }, nil
	}

	state, err := xterm.MakeRaw(fd)
	if err != nil {
		return nil, fmt.Errorf("term: failed to enter raw mode: %w", err)
	}

	return func() error {
		if restoreErr := xterm.Restore(fd, state); restoreErr != nil {
			return fmt.Errorf("term: failed to restore terminal mode: %w", restoreErr)
		}
		return nil
	}, nil
}
