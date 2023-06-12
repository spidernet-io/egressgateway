// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1"
	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := egressv1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := calicov1.AddToScheme(scheme); err != nil {
		panic(err)
	}
}

// GetScheme returns scheme
func GetScheme() *runtime.Scheme {
	return scheme
}
