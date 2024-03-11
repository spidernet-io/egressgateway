// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestRemoveFinalizers tests the RemoveFinalizers function.
func TestRemoveFinalizers(t *testing.T) {
	tests := []struct {
		name       string
		objMeta    metav1.ObjectMeta
		finalizers []string
		want       []string
	}{
		{
			name: "remove single finalizer",
			objMeta: metav1.ObjectMeta{
				Finalizers: []string{"finalize1", "finalize2", "finalize3"},
			},
			finalizers: []string{"finalize2"},
			want:       []string{"finalize1", "finalize3"},
		},
		{
			name: "remove multiple finalizers",
			objMeta: metav1.ObjectMeta{
				Finalizers: []string{"finalize1", "finalize2", "finalize3", "finalize4"},
			},
			finalizers: []string{"finalize2", "finalize4"},
			want:       []string{"finalize1", "finalize3"},
		},
		{
			name: "remove non-existent finalizer",
			objMeta: metav1.ObjectMeta{
				Finalizers: []string{"finalize1", "finalize2"},
			},
			finalizers: []string{"finalize3"},
			want:       []string{"finalize1", "finalize2"},
		},
		{
			name: "remove all finalizers",
			objMeta: metav1.ObjectMeta{
				Finalizers: []string{"finalize1", "finalize2"},
			},
			finalizers: []string{"finalize1", "finalize2"},
			want:       []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RemoveFinalizers(&tt.objMeta, tt.finalizers...)
			if !reflect.DeepEqual(tt.objMeta.Finalizers, tt.want) {
				t.Errorf("RemoveFinalizers() got = %v, want %v", tt.objMeta.Finalizers, tt.want)
			}
		})
	}
}
