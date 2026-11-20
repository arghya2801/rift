//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package store

import (
	"golang.org/x/sys/unix"
	"os"
)

func lockFile(f *os.File) error {
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX)
		if err != unix.EINTR {
			return err
		}
	}
}
