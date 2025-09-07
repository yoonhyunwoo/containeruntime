package pty

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func NewPty() (master *os.File, slavePath string, err error) {
	master, err = os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, "", fmt.Errorf("pty : failed to open /dev/ptmx: %w", err)
	}

	slavePath, err = ptySlaveName(master)
	if err != nil {
		master.Close()
		return nil, "", fmt.Errorf("pty : failed to get slave pty name: %w", err)
	}

	if err = ptyUnlock(master); err != nil {
		master.Close()
		return nil, "", fmt.Errorf("pty : failed to unlock master pty: %w", err)
	}

	return master, slavePath, nil
}

func ptyUnlock(f *os.File) error {
	var unlock int = 0
	fmt.Printf("[DEBUG] %d\n", unlock)
	fmt.Printf("[DEBUG] %d\n", uintptr(unlock))
	return unix.IoctlSetInt(int(f.Fd()), unix.TIOCSPTLCK, unlock)
}

func ptySlaveName(f *os.File) (string, error) {
	n, err := unix.IoctlGetInt(int(f.Fd()), unix.TIOCGPTN)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/dev/pts/%d", n), nil
}

func HandleResize(ptmx *os.File) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	for range ch {
		width, height, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			continue
		}
		winsize := &unix.Winsize{Row: uint16(height), Col: uint16(width)}
		unix.IoctlSetWinsize(int(ptmx.Fd()), unix.TIOCSWINSZ, winsize)
	}
}
