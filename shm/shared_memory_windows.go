// Copyright 2015 Aleksandr Demakin. All rights reserved.

package shm

import (
	"os"
	"path/filepath"
	"runtime"

	"bitbucket.org/avd/go-ipc/internal/common"
)

// Shared memory on Windows is emulated via regular files
// like it is done in boost c++ library.
type memoryObject struct {
	file *os.File
}

func newMemoryObject(name string, mode int, perm os.FileMode) (impl *memoryObject, err error) {
	path, err := shmName(name)
	if err != nil {
		return nil, err
	}
	osMode, err := common.OpenModeToOsMode(mode)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, osMode, perm)
	if err != nil {
		return nil, err
	}
	return &memoryObject{file}, nil
}

func (obj *memoryObject) Destroy() error {
	if int(obj.Fd()) >= 0 {
		if err := obj.Close(); err != nil {
			return err
		}
	}
	return DestroyMemoryObject(obj.Name())
}

func (obj *memoryObject) Name() string {
	return filepath.Base(obj.file.Name())
}

func (obj *memoryObject) Close() error {
	runtime.SetFinalizer(obj, nil)
	return obj.file.Close()
}

func (obj *memoryObject) Truncate(size int64) error {
	return obj.file.Truncate(size)
}

func (obj *memoryObject) Size() int64 {
	fileInfo, err := obj.file.Stat()
	if err != nil {
		return 0
	}
	return fileInfo.Size()
}

func (obj *memoryObject) Fd() uintptr {
	return obj.file.Fd()
}

func destroyMemoryObject(name string) error {
	path, err := shmName(name)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func shmName(name string) (string, error) {
	path, err := sharedDirName()
	if err != nil {
		return "", err
	}
	return path + "/" + name, nil
}

func sharedDirName() (string, error) {
	rootPath := os.TempDir() + "/go-ipc"
	if err := os.Mkdir(rootPath, 0644); err != nil && !os.IsExist(err) {
		return "", err
	}
	return rootPath, nil
}
