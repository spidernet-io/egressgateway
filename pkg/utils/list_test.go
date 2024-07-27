// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils

import "testing"

func TestEqualStringSlice(t *testing.T) {
	cases := []struct {
		name   string
		a      []string
		b      []string
		expect bool
	}{
		{"equal slices", []string{"a", "b", "c"}, []string{"a", "b", "c"}, true},
		{"different elements", []string{"a", "c", "b"}, []string{"a", "b", "c"}, true},
		{"different lengths", []string{"a", "b", "c"}, []string{"a", "b"}, false},
		{"different elements", []string{"a", "b", "c"}, []string{"a", "b", "d"}, false},
		{"empty slices", []string{}, []string{}, true},
		{"one empty slice", []string{"a"}, []string{}, false},
	}

	for _, v := range cases {
		t.Run(v.name, func(t *testing.T) {
			out := EqualStringSlice(v.a, v.b)
			if out != v.expect {
				t.Errorf("EqualStringSlice() got = %v, want = %v", out, v.expect)
			}
		})
	}
}
