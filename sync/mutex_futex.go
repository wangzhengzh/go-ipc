// Copyright 2016 Aleksandr Demakin. All rights reserved.

// +build linux freebsd

package sync

import (
	"os"
	"time"
	"unsafe"

	"bitbucket.org/avd/go-ipc/internal/allocator"
	"bitbucket.org/avd/go-ipc/mmf"
	"bitbucket.org/avd/go-ipc/shm"

	"github.com/pkg/errors"
)

// all implementations must satisfy at least IPCLocker interface.
var (
	_ TimedIPCLocker = (*FutexMutex)(nil)
)

// FutexMutex is a mutex based on linux futex object.
type FutexMutex struct {
	lwm    *lwMutex
	region *mmf.MemoryRegion
	name   string
}

// NewFutexMutex creates a new futex-based mutex.
// This implementation is based on a paper 'Futexes Are Tricky' by Ulrich Drepper,
// this document can be found in 'docs' folder.
//	name - object name.
//	flag - flag is a combination of open flags from 'os' package.
//	perm - object's permission bits.
func NewFutexMutex(name string, flag int, perm os.FileMode) (*FutexMutex, error) {
	if err := ensureOpenFlags(flag); err != nil {
		return nil, err
	}
	obj, created, resultErr := shm.NewMemoryObjectSize(mutexSharedStateName(name, "f"), flag, perm, int64(lwmCellSize))
	if resultErr != nil {
		return nil, errors.Wrap(resultErr, "failed to create shm object")
	}
	var region *mmf.MemoryRegion
	defer func() {
		obj.Close()
		if resultErr == nil {
			return
		}
		if region != nil {
			region.Close()
		}
		if created {
			obj.Destroy()
		}
	}()
	if region, resultErr = mmf.NewMemoryRegion(obj, mmf.MEM_READWRITE, 0, lwmCellSize); resultErr != nil {
		return nil, errors.Wrap(resultErr, "failed to create shm region")
	}
	fw := new(futex)
	lwm := newLightweightMutex(allocator.ByteSliceData(region.Data()), fw)
	fw.ptr = unsafe.Pointer(lwm.ptr)
	if created {
		lwm.init()
	}
	return &FutexMutex{lwm: lwm, name: name, region: region}, nil
}

// Lock locks the mutex. It panics on an error.
func (f *FutexMutex) Lock() {
	f.lwm.lock()
}

// TryLock makes one attempt to lock the mutex. It return true on succeess and false otherwise.
func (f *FutexMutex) TryLock() bool {
	return f.lwm.tryLock()
}

// LockTimeout tries to lock the locker, waiting for not more, than timeout.
func (f *FutexMutex) LockTimeout(timeout time.Duration) bool {
	return f.lwm.lockTimeout(timeout)
}

// Unlock releases the mutex. It panics on an error, or if the mutex is not locked.
func (f *FutexMutex) Unlock() {
	f.lwm.unlock()
}

// Close indicates, that the object is no longer in use,
// and that the underlying resources can be freed.
func (f *FutexMutex) Close() error {
	return f.region.Close()
}

// Destroy removes the mutex object.
func (f *FutexMutex) Destroy() error {
	if err := f.Close(); err != nil {
		return errors.Wrap(err, "failed to close shm region")
	}
	f.region = nil
	f.lwm = nil
	return DestroyFutexMutex(f.name)
}

// DestroyFutexMutex permanently removes mutex with the given name.
func DestroyFutexMutex(name string) error {
	err := shm.DestroyMemoryObject(mutexSharedStateName(name, "f"))
	if err != nil {
		return errors.Wrap(err, "failed to destroy memory object")
	}
	return nil
}
