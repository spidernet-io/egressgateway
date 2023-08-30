// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package markallocator_test

import (
	"github.com/spidernet-io/egressgateway/pkg/markallocator"
	"testing"
)

func TestAllocatorMarkRange(t *testing.T) {

	Allocator, _ := markallocator.NewAllocatorMarkRange("0x26000000")
	_, _ = Allocator.AllocateNext()
	_ = Allocator.Has("0x26000000")
	_ = Allocator.Allocate("0x26000000")
	_ = Allocator.Release("0x26000000")
	Allocator.ForEach(func(mark string) {
	})
	_ = Allocator.Has("0x23000000")
	_ = Allocator.Allocate("0x23000000")
	_ = Allocator.Release("0x23000000")
}
