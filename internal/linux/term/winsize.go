package term

import (
	"fmt"

	"golang.org/x/sys/unix"
	xterm "golang.org/x/term"
)

// SyncWinsizeFromTerminal copies the current terminal window size from srcFD to dstFD.
func SyncWinsizeFromTerminal(srcFD, dstFD int) error {
	if !xterm.IsTerminal(srcFD) {
		return nil
	}

	size, err := unix.IoctlGetWinsize(srcFD, unix.TIOCGWINSZ)
	if err != nil {
		return fmt.Errorf("term: failed to read terminal size: %w", err)
	}
	if err := unix.IoctlSetWinsize(dstFD, unix.TIOCSWINSZ, size); err != nil {
		return fmt.Errorf("term: failed to set pty window size: %w", err)
	}

	return nil
}
