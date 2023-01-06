// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package set

import (
	"errors"
	"fmt"
)

type Set[T any] interface {
	Len() int
	Add(T)
	AddAll(itemArray []T)
	AddSet(other Set[T])
	Discard(T)
	Clear()
	Contains(T) bool
	Iter(func(item T) error)
	Copy() Set[T]
	Equals(Set[T]) bool
	ContainsAll(Set[T]) bool
	Slice() []T
	fmt.Stringer
}

var (
	StopIteration = errors.New("stop iteration")
	RemoveItem    = errors.New("remove item")
)

type v struct{}

var emptyValue = v{}
