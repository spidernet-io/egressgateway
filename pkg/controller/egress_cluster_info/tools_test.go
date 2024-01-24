// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo

import (
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_GetPodByLabel(t *testing.T) {
	t.Run("failed List", func(t *testing.T) {
		c := fake.NewFakeClient()
		patch := gomonkey.ApplyMethodReturn(c, "List", ErrForMock)
		defer patch.Reset()

		_, err := GetPodByLabel(c, map[string]string{"test": "GetPodByLabel"})
		assert.Error(t, err)
	})
}
