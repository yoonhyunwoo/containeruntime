package pty

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func PtyPair() (masterPty *os.File, slavePty *os.File, err error) {
	masterPty, err = os.Open("/dev/ptmx")
	if err != nil {
		return nil, nil, fmt.Errorf("pty: failed to open /dev/ptmx: %w", err)
	}

	var ptn uint32
	syscall.Syscall(syscall.TIOCGPTN, masterPty.Fd(), uintptr(unsafe.Pointer(&ptn)), 0)
	slaveName := fmt.Sprintf("/dev/pts/%d", ptn)
	slavePty, err = os.OpenFile(slaveName, os.O_RDWR, 0)
	if err != nil {
		masterPty.Close()
		return nil, nil, fmt.Errorf("pty: failed to open slave pty %s: %w", slaveName, err)
	}
	return masterPty, slavePty, nil
}
