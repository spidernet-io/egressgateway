// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteObj(ctx context.Context, cli client.Client, obj client.Object) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if !reflect.ValueOf(obj).IsNil() {
		err := cli.Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj)
		if err == nil {
			err := cli.Delete(ctx, obj)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
