// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

func YamlMarshal(in interface{}) []byte {
	out, err := yaml.Marshal(in)
	Expect(err).NotTo(HaveOccurred())
	return out
}
