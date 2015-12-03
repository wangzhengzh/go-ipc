// Copyright 2015 Aleksandr Demakin. All rights reserved.

// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package ipc

import (
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

type Fifo struct {
	file *os.File
}

// Creates or opens new FIFO object
// name - object name. if it does not contain '/', then '/tmp/' prefix will be added
// mode - access mode. can be one of the following:
//	O_READ_ONLY
//	O_WRITE_ONLY
//	O_READWRITE
// perm - file permissions
func NewFifo(name string, mode int, perm os.FileMode) (*Fifo, error) {
	path := fifoPath(name)
	osMode, err := accessModeToOsMode(mode)
	if err != nil {
		return nil, err
	}
	if mode&O_NONBLOCK != 0 {
		osMode |= unix.O_NONBLOCK
	}
	if err := unix.Mkfifo(path, uint32(perm)); err != nil && !os.IsExist(err) {
		return nil, err
	}
	file, err := os.OpenFile(path, osMode, perm)
	if err != nil {
		return nil, err
	}
	return &Fifo{file: file}, nil
}

func (f *Fifo) Read(b []byte) (n int, err error) {
	return f.file.Read(b)
}

func (f *Fifo) Write(b []byte) (n int, err error) {
	return f.file.Write(b)
}

func (f *Fifo) Close() error {
	return f.file.Close()
}

func (f *Fifo) Destroy() error {
	if err := f.file.Close(); err == nil {
		return os.Remove(f.file.Name())
	} else {
		return err
	}
}

func DestroyFifo(name string) error {
	err := os.Remove(fifoPath(name))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// returns full path for the fifo
// if its name contains '/' ('/tmp/fifo', './fifo') - use it
// if only filename was passed, create it in /tmp
func fifoPath(name string) string {
	if strings.Contains(name, "/") {
		return name
	}
	return "/tmp/" + name
}