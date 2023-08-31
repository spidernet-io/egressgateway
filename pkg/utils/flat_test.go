// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func TestKindToMapFlat(t *testing.T) {

	kinds := []string{"", "egress"}

	for _, v := range kinds {
		t.Run("kind", func(t *testing.T) {
			mf := utils.KindToMapFlat(v)
			ctx := context.TODO()
			eg := new(egressv1.EgressGateway)
			eg.Name = "test"
			eg.Namespace = "test"
			_ = mf(ctx, eg)
			eg.Namespace = ""
			_ = mf(ctx, eg)

		})
	}

}

func TestParseKindWithReq(t *testing.T) {

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test",
			Name:      "test",
		},
	}
	_, _, _ = utils.ParseKindWithReq(req)

	req = reconcile.Request{
		NamespacedName: types.NamespacedName{},
	}
	_, _, _ = utils.ParseKindWithReq(req)
}
