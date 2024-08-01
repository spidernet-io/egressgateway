// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils

func EqualStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[string]int)
	for _, v := range a {
		aMap[v]++
	}

	for _, v := range b {
		if count, exists := aMap[v]; !exists || count == 0 {
			return false
		}
		aMap[v]--
	}

	return true
}
