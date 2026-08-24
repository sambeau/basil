//go:build unix

package deploy

import (
	"os"
	"syscall"
)

// tryLockFile takes a non-blocking exclusive flock on f, returning
// errLockHeld when another process holds it. flock is advisory and released
// by the kernel on process exit.
func tryLockFile(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
		return errLockHeld
	}
	return err
}

// unlockFile drops the flock.
func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
