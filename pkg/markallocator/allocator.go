// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package markallocator

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/cilium/ipam/service/allocator"
)

var (
	ErrFull      = errors.New("range is full")
	ErrAllocated = errors.New("provided mark is already allocated")
)

type Interface interface {
	Allocate(mark string) error
	AllocateNext() (string, error)
	Release(mark string) error
	ForEach(func(mark string))

	// Has function for testing
	Has(mark string) bool
}

type Range struct {
	base *big.Int

	start, end uint64

	// max is the maximum size of the usable addresses in the range
	max int

	alloc allocator.Interface
}

func NewAllocatorCIDRRange(mask string) (Interface, error) {
	start, end, err := rangeSize(mask)
	if err != nil {
		return nil, err
	}
	base, err := bigForMark(mask)
	if err != nil {
		return nil, err
	}
	r := &Range{
		max:   int(end - start - 1),
		base:  base.Add(base, big.NewInt(1)),
		start: start,
		end:   end,
	}
	r.alloc = allocator.NewAllocationMap(r.max, "")
	return r, err
}

func bigForMark(mark string) (*big.Int, error) {
	val, err := Parse(mark)
	if err != nil {
		return nil, err
	}
	return big.NewInt(0).SetUint64(val), nil
}

func rangeSize(mask string) (uint64, uint64, error) {
	start, err := Parse(mask)
	if err != nil {
		return 0, 0, err
	}
	endStr := ""
	for i := len(mask) - 1; i >= 0; i-- {
		item := mask[i]
		if item == 48 {
			endStr = "f" + endStr
		} else {
			endStr = mask[:i+1] + endStr
			break
		}
	}
	end, err := Parse(endStr)
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

func Parse(mark string) (uint64, error) {
	tmp := strings.ReplaceAll(mark, "0x", "")
	return strconv.ParseUint(tmp, 16, 64)
}

// Allocate attempts to reserve the provided mark. ErrNotInRange or
// ErrAllocated will be returned if the mark is not valid for this range
// or has already been reserved. ErrFull will be returned if there
// are no addresses left.
func (r *Range) Allocate(mark string) error {
	m, err := Parse(mark)
	if err != nil {
		return err
	}

	ok, offset := r.contains(m)
	if !ok {
		return fmt.Errorf("%s not in range [%x, %x]", mark, r.start, r.end)
	}

	allocated, err := r.alloc.Allocate(offset)
	if err != nil {
		return err
	}
	if !allocated {
		return ErrAllocated
	}
	return nil
}

// AllocateNext reserves one of the mark from the pool. ErrFull may
// be returned if there are no addresses left.
func (r *Range) AllocateNext() (string, error) {
	offset, ok, err := r.alloc.AllocateNext()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrFull
	}
	return addMarkOffset(r.base, offset), nil
}

// Release releases the mark back to the pool. Releasing an
// unallocated mark or a mark out of the range is a no-op and
// returns no error.
func (r *Range) Release(mark string) error {
	m, err := Parse(mark)
	if err != nil {
		return err
	}

	ok, offset := r.contains(m)
	if !ok {
		return fmt.Errorf("%s not in range [%x, %x]", mark, r.start, r.end)
	}

	return r.alloc.Release(offset)
}

// ForEach calls the provided function for each allocated mark.
func (r *Range) ForEach(fn func(mark string)) {
	r.alloc.ForEach(func(offset int) {
		mark, _ := GetIndexedMark(r.base, offset)
		fn(mark)
	})
}

// GetIndexedMark returns a string that is r.base + index in the contiguous mark space.
func GetIndexedMark(base *big.Int, index int) (string, error) {
	mark := addMarkOffset(base, index)
	return mark, nil
}

// Has returns true if the provided mark is already allocated and a call
// to Allocate(mark) would fail with ErrAllocated.
func (r *Range) Has(mark string) bool {
	m, err := Parse(mark)
	if err != nil {
		return false
	}

	ok, offset := r.contains(m)
	if !ok {
		return false
	}

	return r.alloc.Has(offset)
}

// addMarkOffset adds the provided integer offset to a base big.Int representing a mark
func addMarkOffset(base *big.Int, offset int) string {
	r := big.NewInt(0).Add(base, big.NewInt(int64(offset))).Uint64()
	return "0x" + strconv.FormatUint(r, 16)
}

// contains returns true and the offset if the mrk is in the range, and false
// and nil otherwise. The first and last addresses of the mask are omitted.
func (r *Range) contains(mark uint64) (bool, int) {
	if r.start > mark || r.end < mark {
		return false, 0
	}

	offset := calculateMarkOffset(r.base, mark)
	if offset < 0 || offset >= r.max {
		return false, 0
	}
	return true, offset
}

// calculateMarkOffset calculates the integer offset of mark from base such that
// base + offset = mark. It requires mark >= base.
func calculateMarkOffset(base *big.Int, mark uint64) int {
	return int(big.NewInt(0).Sub(big.NewInt(0).SetUint64(mark), base).Int64())
}
