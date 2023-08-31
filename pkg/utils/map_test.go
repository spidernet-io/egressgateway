// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"github.com/spidernet-io/egressgateway/pkg/utils"
	"testing"
)

func TestSyncMap(t *testing.T) {

	sMap := utils.NewSyncMap[string, string]()
	sMap.Store("a", "b")
	sMap.Store("c", "d")
	_, _ = sMap.Load("a")
	_, _ = sMap.Load("b")
	sMap.Delete("a")

	sMap.Range(func(key string, val string) bool {
		return true
	})
}
