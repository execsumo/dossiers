package store

import (
	"os"
	"syscall"
)

// FileLock represents an active advisory file lock.
type FileLock struct {
	file *os.File
}

// Lock acquires an exclusive lock on the file at the given path.
func Lock(path string) (*FileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
	if err != nil {
		f.Close()
		return nil, err
	}
	return &FileLock{file: f}, nil
}

// Unlock releases the exclusive lock and closes the lock file descriptor.
func (l *FileLock) Unlock() error {
	defer l.file.Close()
	return syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
}
