package pty

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func ioctl(fd uintptr, req, arg uintptr) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, arg); errno != 0 {
		return errno
	}
	return nil
}

func PtyPair() (masterPty, slavePty *os.File, err error) {
	masterPty, err = os.Open("/dev/ptmx")
	if err != nil {
		return nil, nil, fmt.Errorf("pty: failed to open /dev/ptmx: %w", err)
	}

	var unlock int32
	if err = ioctl(masterPty.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); err != nil {
		_ = masterPty.Close()
		return nil, nil, fmt.Errorf("pty: failed to unlock ptmx: %w", err)
	}

	var ptn uint32
	// #nosec G103 -- ioctl requires pointer passing to kernel for PTY index retrieval.
	if err = ioctl(masterPty.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&ptn))); err != nil {
		_ = masterPty.Close()
		return nil, nil, fmt.Errorf("pty: failed to get slave pty number: %w", err)
	}

	slaveName := fmt.Sprintf("/dev/pts/%d", ptn)
	slavePty, err = os.OpenFile(slaveName, os.O_RDWR, 0)
	if err != nil {
		_ = masterPty.Close()
		return nil, nil, fmt.Errorf("pty: failed to open slave pty %s: %w", slaveName, err)
	}
	return masterPty, slavePty, nil
}
