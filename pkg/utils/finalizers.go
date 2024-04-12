// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func RemoveFinalizers(objMeta *metav1.ObjectMeta, finalizers ...string) {
	for _, f := range finalizers {
		for i := len(objMeta.Finalizers) - 1; i >= 0; i-- {
			if objMeta.Finalizers[i] == f {
				objMeta.Finalizers = append(objMeta.Finalizers[:i], objMeta.Finalizers[i+1:]...)
				break
			}
		}
	}
}
