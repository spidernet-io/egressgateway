// Copyright 2017 The Kubernetes Authors.
// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/spidernet-io/egressgateway/pkg/ipset"
)

// FakeIPSet is a no-op implementation of ipset Interface
type FakeIPSet struct {
	// version of ipset util
	Version string
	// The key of Sets maps is the ip set name
	Sets map[string]*ipset.IPSet
	// The key of Entries maps is the ip set name where the entries exist
	Entries map[string]sets.Set[string]
}

// NewFake create a new fake ipset interface - it initialize the FakeIPSet.
func NewFake(version string) *FakeIPSet {
	return &FakeIPSet{
		Version: version,
		Sets:    make(map[string]*ipset.IPSet),
		Entries: make(map[string]sets.Set[string]),
	}
}

// GetVersion is part of interface.
func (f *FakeIPSet) GetVersion() (string, error) {
	return f.Version, nil
}

// FlushSet is part of interface.  It deletes all entries from a named set but keeps the set itself.
func (f *FakeIPSet) FlushSet(set string) error {
	if f.Entries == nil {
		return fmt.Errorf("entries map can't be nil")
	}

	// delete all entry elements
	//nolint
	for true {
		if _, has := f.Entries[set].PopAny(); has {
			continue
		}
		break
	}
	return nil
}

// DestroySet is part of interface.  It deletes both the entries and the set itself.
func (f *FakeIPSet) DestroySet(set string) error {
	delete(f.Sets, set)
	delete(f.Entries, set)
	return nil
}

// DestroyAllSets is part of interface.
func (f *FakeIPSet) DestroyAllSets() error {
	f.Sets = nil
	f.Entries = nil
	return nil
}

// CreateSet is part of interface.
func (f *FakeIPSet) CreateSet(set *ipset.IPSet, ignoreExistErr bool) error {
	if f.Sets[set.Name] != nil {
		if !ignoreExistErr {
			// already exists
			return fmt.Errorf("set cannot be created: set with the same name already exists")
		}
		return nil
	}
	f.Sets[set.Name] = set
	// initialize entry map
	f.Entries[set.Name] = sets.New[string]()
	return nil
}

// AddEntry is part of interface.
func (f *FakeIPSet) AddEntry(entry string, set *ipset.IPSet, ignoreExistErr bool) error {
	if f.Entries[set.Name].Has(entry) {
		if !ignoreExistErr {
			// already exists
			return ipset.ErrAlreadyAddedEntry
		}
		return nil
	}
	f.Entries[set.Name].Insert(entry)
	return nil
}

// DelEntry is part of interface.
func (f *FakeIPSet) DelEntry(entry string, set string) error {
	if f.Entries == nil {
		return fmt.Errorf("entries map can't be nil")
	}
	f.Entries[set].Delete(entry)
	return nil
}

// TestEntry is part of interface.
func (f *FakeIPSet) TestEntry(entry string, set string) (bool, error) {
	if f.Entries == nil {
		return false, fmt.Errorf("entries map can't be nil")
	}
	found := f.Entries[set].Has(entry)
	return found, nil
}

// ListEntries is part of interface.
func (f *FakeIPSet) ListEntries(set string) ([]string, error) {
	if f.Entries == nil {
		return nil, fmt.Errorf("entries map can't be nil")
	}
	return f.Entries[set].UnsortedList(), nil
}

// ListSets is part of interface.
func (f *FakeIPSet) ListSets() ([]string, error) {
	res := make([]string, 0)
	for set := range f.Sets {
		res = append(res, set)
	}
	return res, nil
}

var _ = ipset.Interface(&FakeIPSet{})
