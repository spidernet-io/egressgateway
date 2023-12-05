// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package slice

func RemoveElement[T int | string](slice []T, element T) []T {
	i := indexOf(slice, element)
	if i == -1 {
		return slice
	}
	return append(slice[:i], slice[i+1:]...)
}

func indexOf[T int | string](slice []T, element T) int {
	for i, v := range slice {
		if v == element {
			return i
		}
	}
	return -1
}
