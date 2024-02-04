// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package markallocator

import (
	"errors"
	"math/big"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/cilium/ipam/service/allocator"
	"github.com/stretchr/testify/assert"
)

func TestAllocatorMarkRange(t *testing.T) {

	Allocator, _ := NewAllocatorMarkRange("0x26000000")
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

var ErrForMock = errors.New("mock err")

func Test_NewAllocatorMarkRange(t *testing.T) {
	cases := map[string]struct {
		patchFunc func() []gomonkey.Patches
		expErr    bool
	}{
		"failed RangeSize": {
			patchFunc: mock_NewAllocatorMarkRange_RangeSize,
			expErr:    true,
		},

		"failed bigForMark": {
			patchFunc: mock_NewAllocatorMarkRange_bigForMark,
			expErr:    true,
		},
	}

	mark := "0x13413"

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc()
			}

			_, err := NewAllocatorMarkRange(mark)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_bigForMark(t *testing.T) {
	patch := gomonkey.NewPatches()
	patch.ApplyFuncReturn(Parse, uint64(0), ErrForMock)
	defer patch.Reset()

	mark := "0x13413"
	_, err := bigForMark(mark)
	assert.Error(t, err)
}

func Test_RangeSize(t *testing.T) {
	cases := map[string]struct {
		patchFunc func() []gomonkey.Patches
		expErr    bool
	}{
		"failed Parse start": {
			patchFunc: mock_RangeSize_Parse_start,
			expErr:    true,
		},
		"failed Parse end": {
			patchFunc: mock_RangeSize_Parse_end,
			expErr:    true,
		},
	}

	mark := "0x13413"

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc()
			}

			_, _, err := RangeSize(mark)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_Range_Allocate(t *testing.T) {
	cases := map[string]struct {
		patchFunc func(r *Range) []gomonkey.Patches
		expErr    bool
	}{
		"failed Parse": {
			patchFunc: mock_Range_Allocate_Parse,
			expErr:    true,
		},
		"failed Allocate err": {
			patchFunc: mock_Range_Allocate_Allocate_err,
			expErr:    true,
		},
		"failed Allocate false": {
			patchFunc: mock_Range_Allocate_Allocate_false,
			expErr:    true,
		},
		"succeeded Allocate": {},
	}

	r := Range{start: uint64(0), end: uint64(512), base: big.NewInt(100), max: 200, alloc: allocator.NewAllocationMap(200, "")}
	mark := "0x100"
	// r, _ := NewAllocatorMarkRange(mark)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(&r)
			}

			err := r.Allocate(mark)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_Range_AllocateNext(t *testing.T) {
	cases := map[string]struct {
		patchFunc func(r *Range) []gomonkey.Patches
		expErr    bool
	}{
		"failed AllocateNext err": {
			patchFunc: mock_Range_AllocateNext_AllocateNext_err,
			expErr:    true,
		},
		"failed AllocateNext false": {
			patchFunc: mock_Range_AllocateNext_AllocateNext_false,
			expErr:    true,
		},
	}

	r := Range{start: uint64(0), end: uint64(512), base: big.NewInt(100), max: 200, alloc: allocator.NewAllocationMap(200, "")}
	// mark := "0x100"
	// r, _ := NewAllocatorMarkRange(mark)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(&r)
			}

			_, err := r.AllocateNext()
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_Range_Release(t *testing.T) {
	cases := map[string]struct {
		patchFunc func(r *Range) []gomonkey.Patches
		expErr    bool
	}{
		"failed Parse": {
			patchFunc: mock__Range_Release_Parse,
			expErr:    true,
		},
		"succeeded Release": {},
	}

	r := Range{start: uint64(0), end: uint64(512), base: big.NewInt(100), max: 200, alloc: allocator.NewAllocationMap(200, "")}
	mark := "0x100"
	// r, _ := NewAllocatorMarkRange(mark)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(&r)
			}

			err := r.Release(mark)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_Range_Has(t *testing.T) {
	cases := map[string]struct {
		patchFunc func(r *Range) []gomonkey.Patches
		expOK     bool
	}{
		"failed Parse": {
			patchFunc: mock__Range_Has_Parse,
			expOK:     true,
		},
		"succeeded Parse": {},
	}

	r := Range{start: uint64(0), end: uint64(512), base: big.NewInt(100), max: 200, alloc: allocator.NewAllocationMap(200, "")}
	mark := "0x100"
	// r, _ := NewAllocatorMarkRange(mark)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(&r)
			}

			b := r.Has(mark)
			if tc.expOK {
				assert.False(t, b)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_Range_contains(t *testing.T) {
	cases := map[string]struct {
		patchFunc func(r *Range) []gomonkey.Patches
		expOK     bool
	}{
		"succeeded contains": {},
	}

	r := Range{start: uint64(0), end: uint64(512), base: big.NewInt(100), max: 200, alloc: allocator.NewAllocationMap(200, "")}
	mark := uint64(100)
	// r, _ := NewAllocatorMarkRange(mark)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(&r)
			}

			b, _ := r.contains(mark)
			if tc.expOK {
				assert.False(t, b)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func mock_NewAllocatorMarkRange_RangeSize() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(RangeSize, uint64(0), uint64(0), ErrForMock)
	return []gomonkey.Patches{*patch}
}

func mock_NewAllocatorMarkRange_bigForMark() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(bigForMark, nil, ErrForMock)
	return []gomonkey.Patches{*patch}
}

func mock_RangeSize_Parse_start() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(Parse, uint64(0), ErrForMock)
	return []gomonkey.Patches{*patch}
}

func mock_RangeSize_Parse_end() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncSeq(Parse, []gomonkey.OutputCell{
		{Values: gomonkey.Params{uint64(1234567), nil}, Times: 1},
		{Values: gomonkey.Params{uint64(1234567), ErrForMock}, Times: 1},
	})
	return []gomonkey.Patches{*patch}
}

func mock_Range_Allocate_Parse(r *Range) []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(Parse, uint64(0), ErrForMock)
	return []gomonkey.Patches{*patch}
}

func mock_Range_Allocate_Allocate_err(r *Range) []gomonkey.Patches {
	patch := gomonkey.ApplyPrivateMethod(r.alloc, "Allocate", func(_ allocator.Interface) (bool, error) {
		return false, ErrForMock
	})
	return []gomonkey.Patches{*patch}
}

func mock_Range_Allocate_Allocate_false(r *Range) []gomonkey.Patches {
	patch := gomonkey.ApplyPrivateMethod(r.alloc, "Allocate", func(_ allocator.Interface) (bool, error) {
		return false, nil
	})
	return []gomonkey.Patches{*patch}
}

func mock_Range_AllocateNext_AllocateNext_err(r *Range) []gomonkey.Patches {
	patch := gomonkey.ApplyPrivateMethod(r.alloc, "AllocateNext", func(_ allocator.Interface) (int, bool, error) {
		return 0, false, ErrForMock
	})
	return []gomonkey.Patches{*patch}
}

func mock_Range_AllocateNext_AllocateNext_false(r *Range) []gomonkey.Patches {
	patch := gomonkey.ApplyPrivateMethod(r.alloc, "AllocateNext", func(_ allocator.Interface) (int, bool, error) {
		return 0, false, nil
	})
	return []gomonkey.Patches{*patch}
}

func mock__Range_Release_Parse(r *Range) []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(Parse, uint64(0), ErrForMock)
	return []gomonkey.Patches{*patch}
}

func mock__Range_Has_Parse(r *Range) []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(Parse, uint64(0), ErrForMock)
	return []gomonkey.Patches{*patch}
}
