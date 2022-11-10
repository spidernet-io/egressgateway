// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"
	"testing"

	"github.com/spidernet-io/egressgateway/pkg/config"
)

func TestController(t *testing.T) {
	testCases := map[string]struct {
		name   string
		cfg    *config.Config
		expect bool
	}{
		"": {},
	}
	for _, testCase := range testCases {
		fmt.Println(testCase)
	}
}
