// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package slice

import (
	"reflect"
	"testing"
)

func TestRemoveElement(t *testing.T) {
	cases := []struct {
		name   string
		in     []string
		el     string
		expect []string
	}{
		{"when included", []string{"a", "b", "c"}, "a", []string{"b", "c"}},
		{"when not included", []string{"a", "b", "c"}, "d", []string{"a", "b", "c"}},
		{"when empty slice", []string{}, "a", []string{}},
	}

	for _, v := range cases {
		t.Run(v.name, func(t *testing.T) {
			out := RemoveElement[string](v.in, v.el)
			if !reflect.DeepEqual(out, v.expect) {
				t.Errorf("RemoveElement() got = %v, want = %v", out, v.expect)
			}
		})
	}
}
