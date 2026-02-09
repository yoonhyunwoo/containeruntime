package term

import xterm "golang.org/x/term"

// IsTerminal reports whether fd is an interactive terminal.
func IsTerminal(fd int) bool {
	return xterm.IsTerminal(fd)
}
