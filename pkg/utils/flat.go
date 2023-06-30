// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"errors"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var ErrInvalidRequest = errors.New("error invalid request")

func KindToMapFlat(kind string) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		ns := obj.GetNamespace()
		if ns == "" {
			ns = kind + "/"
		} else {
			ns = path.Join(ns, kind)
		}
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: ns,
					Name:      obj.GetName(),
				},
			},
		}
	}
}

func ParseKindWithReq(req reconcile.Request) (string, reconcile.Request, error) {
	arr := strings.Split(req.Namespace, "/")
	if len(arr) != 2 {
		return "", reconcile.Request{}, ErrInvalidRequest
	}
	return arr[0], reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: arr[1],
			Name:      req.Name,
		},
	}, nil
}
