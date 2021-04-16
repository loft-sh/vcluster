package locks

import (
	"sync"
)

// LockFactory is the interface to retrieve named locks from
type LockFactory interface {
	GetLock(string) sync.Locker
}

type defaultLockFactory struct {
	lock  sync.RWMutex
	locks map[string]sync.Locker
}

// NewDefaultLockFactory creates a new lock factory
func NewDefaultLockFactory() LockFactory {
	return &defaultLockFactory{locks: map[string]sync.Locker{}}
}

func (f *defaultLockFactory) GetLock(key string) sync.Locker {
	lock, exists := f.getExistingLock(key)
	if exists {
		return lock
	}

	f.lock.Lock()
	defer f.lock.Unlock()

	lock, exists = f.locks[key]
	if exists {
		return lock
	}

	lock = &sync.Mutex{}
	f.locks[key] = lock
	return lock
}

func (f *defaultLockFactory) getExistingLock(key string) (sync.Locker, bool) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	lock, exists := f.locks[key]
	return lock, exists
}
