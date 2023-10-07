// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEgressClusterInfo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EgressClusterInfo Suite UT")
}

var egciName = "default"

var _ = BeforeSuite(func() {

	DeferCleanup(func() {

	})
})
