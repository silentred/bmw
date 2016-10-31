package lib

import (
	"syscall"
	"unsafe"
)

type ProtectVar []byte

func newProtectVar(size int) (ProtectVar, error) {
	b, err := syscall.Mmap(0, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}

	return ProtectVar(b), nil
}

func (p ProtectVar) Free() error {
	return syscall.Munmap([]byte(p))
}

func (p ProtectVar) Readonly() error {
	return syscall.Mprotect([]byte(p), syscall.PROT_READ)
}

func (p ProtectVar) ReadWrite() error {
	return syscall.Mprotect([]byte(p), syscall.PROT_READ|syscall.PROT_WRITE)
}

func (p ProtectVar) Pointer() unsafe.Pointer {
	return unsafe.Pointer(&p[0])
}
