package store

import (
	"github.com/gofrs/flock"
)

// FileLock represents an active advisory file lock.
type FileLock struct {
	fl *flock.Flock
}

// Lock acquires an exclusive lock on the file at the given path, blocking until
// it is available.
func Lock(path string) (*FileLock, error) {
	fl := flock.New(path)
	if err := fl.Lock(); err != nil {
		return nil, err
	}
	return &FileLock{fl: fl}, nil
}

// Unlock releases the exclusive lock and closes the lock file descriptor.
func (l *FileLock) Unlock() error {
	return l.fl.Unlock()
}
