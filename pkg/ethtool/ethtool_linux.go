// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0
//
// Based on the version in the Tigera project, Copyright Tigera:
// https://github.com/projectcalico/calico/blob/9673cc9500c1614f2d24df7b7e555a3e517dbc5c/felix/ethtool/ethtool.go
//

package ethtool

import (
	"fmt"
	"unsafe"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"modernc.org/memory"
)

// IFReqData represents linux/if.h 'struct ifreq'
type IFReqData struct {
	Name [unix.IFNAMSIZ]byte
	Data uintptr
}

// EthtoolValue represents linux/ethtool.h 'struct ethtool_value'
type EthtoolValue struct {
	Cmd  uint32
	Data uint32
}

func ioctlEthtool(fd int, argp *IFReqData) error {
	// Important that the cast to uintptr is _syntactically_ within the Syscall() invocation in order to guarantee
	// safety.  (See notes in the unsafe.Pointer docs.)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(unix.SIOCETHTOOL), uintptr(unsafe.Pointer(argp)))
	if errno != 0 {
		return errno
	}
	return nil
}

// EthtoolTXOff disables the TX checksum offload on the specified interface
func EthtoolTXOff(name string) error {
	if len(name)+1 > unix.IFNAMSIZ {
		return fmt.Errorf("name too long")
	}

	// To access the IOCTL, we need a socket file descriptor.
	socket, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer func() {
		err := unix.Close(socket)
		if err != nil {
			// Super unlikely; normally a failure from Close means that some data couldn't be flushed,
			// but we didn't send any.
			logrus.WithError(err).Warn("unix.Close(socket) failed")
		}
	}()

	// Allocate an EthtoolValue using manual memory management.  This is required because we need to pass
	// a struct to the Syscall that in turn points to the EthtoolValue.  If we allocate the EthtooValue on the
	// go stack/heap then it would not be protected from being moved by the GC while the syscall is in progress.
	// (Only the directly-passed struct is protected from being moved during the syscall.)
	alloc := memory.Allocator{}
	defer func() {
		err := alloc.Close()
		if err != nil {
			logrus.WithError(err).Panic("Failed to release memory to the system")
		}
	}()
	valueUPtr, err := alloc.UnsafeCalloc(int(unsafe.Sizeof(EthtoolValue{})))
	if err != nil {
		return fmt.Errorf("failed to allocate memory: %w", err)
	}
	defer func() {
		err := alloc.UnsafeFree(valueUPtr)
		if err != nil {
			logrus.WithError(err).Warn("UnsafeFree() failed")
		}
	}()
	value := (*EthtoolValue)(valueUPtr)

	// Get the current value, so we only set it if it needs to change.
	*value = EthtoolValue{Cmd: unix.ETHTOOL_GTXCSUM}
	request := IFReqData{Data: uintptr(valueUPtr)}
	copy(request.Name[:], name)
	if err := ioctlEthtool(socket, &request); err != nil {
		return err
	}
	if value.Data == 0 { // if already off, don't try to change
		return nil
	}

	// Set the value.
	*value = EthtoolValue{Cmd: unix.ETHTOOL_STXCSUM, Data: 0 /* off */}
	return ioctlEthtool(socket, &request)
}
