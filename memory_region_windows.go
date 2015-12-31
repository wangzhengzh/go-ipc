// Copyright 2015 Aleksandr Demakin. All rights reserved.

package ipc

import (
	"fmt"

	"golang.org/x/sys/windows"
)

type memoryRegionImpl struct {
	data       []byte
	size       int
	pageOffset int64
}

func newMemoryRegionImpl(obj MappableHandle, mode int, offset int64, size int) (*memoryRegionImpl, error) {
	prot, flags, err := memProtAndFlagsFromMode(mode)
	if err != nil {
		return nil, err
	}
	//file:///home/avd/dev/boost_1_59_0/boost/interprocess/mapped_region.hpp:441
	// TODO(avd) - check if it is not for shm
	handle := windows.InvalidHandle
	if true {
		// TODO(avd) - security attrs
		var err error
		if handle, err = windows.CreateFileMapping(windows.Handle(obj.Fd()), nil, prot, 0, 0, nil); err != nil {
			return nil, err
		}
	} else {
		// TODO(avd) - finish with it
	}
	if size == 0 { // TODO(avd) get current file size

	}
	defer windows.CloseHandle(handle)
	pageOffset := calcValidOffset(offset)
	lowOffset := uint32(pageOffset)
	highOffset := uint32(pageOffset >> 32)
	addr, err := windows.MapViewOfFile(handle, flags, lowOffset, highOffset, uintptr(int64(size)+pageOffset))
	if err != nil {
		return nil, err
	}
	sz := size + int(pageOffset)
	return &memoryRegionImpl{byteSliceFromUintptr(addr, sz, sz), size, pageOffset}, nil
}

func (impl *memoryRegionImpl) Close() error {
	return windows.UnmapViewOfFile(byteSliceAddress(impl.data))
}

func (impl *memoryRegionImpl) Data() []byte {
	return impl.data[impl.pageOffset:]
}

func (impl *memoryRegionImpl) Size() int {
	return impl.size
}

func (impl *memoryRegionImpl) Flush(async bool) error {
	return windows.FlushViewOfFile(byteSliceAddress(impl.data), uintptr(len(impl.data)))
}

func memProtAndFlagsFromMode(mode int) (prot uint32, flags uint32, err error) {
	switch mode {
	case MEM_READ_ONLY:
		fallthrough
	case MEM_READ_PRIVATE:
		prot = windows.PAGE_READONLY
		flags = windows.FILE_MAP_READ
	case MEM_READWRITE:
		prot = windows.PAGE_READWRITE
		flags = windows.FILE_MAP_WRITE
	case MEM_COPY_ON_WRITE:
		prot = windows.PAGE_WRITECOPY
		flags = windows.FILE_MAP_COPY
	default:
		err = fmt.Errorf("invalid mem region flags")
	}
	return
}