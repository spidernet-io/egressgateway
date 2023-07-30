// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestNamespacePredicate(t *testing.T) {
	p := nsPredicate{}
	if !p.Create(event.CreateEvent{}) {
		t.Fatal("got false")
	}

	if !p.Delete(event.DeleteEvent{}) {
		t.Fatal("got false")
	}

	if !p.Update(event.UpdateEvent{
		ObjectOld: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
				Labels: map[string]string{
					"aa": "bb",
				},
			},
		},
		ObjectNew: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
				Labels: map[string]string{
					"aa": "cc",
				},
			},
		},
	}) {
		t.Fatal("got false")
	}
}
