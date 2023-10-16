// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

func GetObjYAML(obj runtime.Object) string {
	o := obj.DeepCopyObject()
	a, err := meta.Accessor(o)
	if err != nil {
		return ""
	}
	a.SetManagedFields(nil)
	res, _ := yaml.Marshal(o)
	return string(res)
}
