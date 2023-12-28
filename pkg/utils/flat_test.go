// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	"reflect"
	"testing"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	tests := []struct {
		name      string
		req       reconcile.Request
		wantKind  string
		wantReq   reconcile.Request
		wantError error
	}{
		{
			name: "Valid input",
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "kind/namespace",
					Name:      "name",
				},
			},
			wantKind: "kind",
			wantReq: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "namespace",
					Name:      "name",
				},
			},
			wantError: nil,
		},
		{
			name: "Invalid input - missing slash",
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "namespace",
					Name:      "name",
				},
			},
			wantKind:  "",
			wantReq:   reconcile.Request{},
			wantError: utils.ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKind, gotReq, gotError := utils.ParseKindWithReq(tt.req)
			if gotKind != tt.wantKind {
				t.Errorf("ParseKindWithReq() gotKind = %v, want %v", gotKind, tt.wantKind)
			}
			if !reflect.DeepEqual(gotReq, tt.wantReq) {
				t.Errorf("ParseKindWithReq() gotReq = %v, want %v", gotReq, tt.wantReq)
			}
			if !reflect.DeepEqual(gotError, tt.wantError) {
				t.Errorf("ParseKindWithReq() gotError = %v, want %v", gotError, tt.wantError)
			}
		})
	}
}
