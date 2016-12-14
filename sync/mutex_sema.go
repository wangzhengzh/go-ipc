// Copyright 2015 Aleksandr Demakin. All rights reserved.

package sync

import (
	"os"

	"bitbucket.org/avd/go-ipc"
	"bitbucket.org/avd/go-ipc/internal/allocator"
	"bitbucket.org/avd/go-ipc/mmf"
	"bitbucket.org/avd/go-ipc/shm"

	"github.com/pkg/errors"
)

// all implementations must satisfy IPCLocker interface.
var (
	_ IPCLocker = (*SemaMutex)(nil)
)

// SemaMutex is a semaphore-based mutex for unix.
type SemaMutex struct {
	s      Semaphore
	region *mmf.MemoryRegion
	name   string
	lwm    *lwMutex
}

// NewSemaMutex creates a new semaphore-based mutex.
func NewSemaMutex(name string, flag int, perm os.FileMode) (*SemaMutex, error) {
	if err := ensureOpenFlags(flag); err != nil {
		return nil, err
	}
	region, created, err := createWritableRegion(mutexSharedStateName(name, "s"), flag, perm, lwmCellSize, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create shared state")
	}
	s, err := NewSemaphore(name, flag, perm, 1)
	if err != nil {
		region.Close()
		if created {
			shm.DestroyMemoryObject(mutexSharedStateName(name, "s"))
		}
		return nil, errors.Wrap(err, "failed to create a semaphore")
	}
	result := &SemaMutex{
		s:      s,
		region: region,
		name:   name,
		lwm:    newLightweightMutex(allocator.ByteSliceData(region.Data()), newSemaWaiter(s)),
	}
	if created {
		result.lwm.init()
	}
	return result, nil
}

// Lock locks the mutex. It panics on an error.
func (m *SemaMutex) Lock() {
	m.lwm.lock()
}

// TryLock makes one attempt to lock the mutex. It return true on succeess and false otherwise.
func (m *SemaMutex) TryLock() bool {
	return m.lwm.tryLock()
}

// Unlock releases the mutex. It panics on an error, or if the mutex is not locked.
func (m *SemaMutex) Unlock() {
	m.lwm.unlock()
}

// Close closes shared state of the mutex.
func (m *SemaMutex) Close() error {
	m.s.Close()
	return m.region.Close()
}

// Destroy closes the mutex and removes it permanently.
func (m *SemaMutex) Destroy() error {
	if err := m.Close(); err != nil {
		return errors.Wrap(err, "failed to close shared state")
	}
	if err := shm.DestroyMemoryObject(mutexSharedStateName(m.name, "s")); err != nil {
		return errors.Wrap(err, "failed to destroy shared state")
	}
	if d, ok := m.s.(ipc.Destroyer); ok {
		if err := d.Destroy(); err != nil {
			return errors.Wrap(err, "failed to destroy semaphore")
		}
	} else {
		m.s.Close()
	}
	return nil
}

// DestroySemaMutex permanently removes mutex with the given name.
func DestroySemaMutex(name string) error {
	if err := shm.DestroyMemoryObject(mutexSharedStateName(name, "s")); err != nil {
		return errors.Wrap(err, "failed to destroy shared state")
	}
	if err := DestroySemaphore(name); err != nil && !os.IsNotExist(errors.Cause(err)) {
		return err
	}
	return nil
}
