// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	goerrors "errors"
	"github.com/go-faker/faker/v4"
	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func CreateCalicoIPPool(ctx context.Context, cli client.Client,
	prefix, cidrPrefix string,
	genRandomCIDR func(string) string) (*calicov1.IPPool, error) {

	pool := &calicov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: prefix + faker.Word(),
		},
		Spec: calicov1.IPPoolSpec{
			CIDR: genRandomCIDR(cidrPrefix),
		},
	}
	err := cli.Create(ctx, pool)
	if err != nil {
		return nil, err
	}
	return pool, err
}

func UpdateCalicoIPPoolCIDR(ctx context.Context, cli client.Client,
	pool *calicov1.IPPool,
	cidrPrefix string,
	genRandomCIDR func(string) string,
) error {
	pool.Spec.CIDR = genRandomCIDR(cidrPrefix)
	err := cli.Update(ctx, pool)
	if err != nil {
		return err
	}
	return nil
}

func ListCalicoIPPool(ctx context.Context, cli client.Client) ([]string, error) {
	res := make([]string, 0)

	list := &calicov1.IPPoolList{}
	err := cli.List(ctx, list)
	if err != nil {
		rdfErr := &apiutil.ErrResourceDiscoveryFailed{}
		if !goerrors.As(err, &rdfErr) {
			return res, err
		}
	}

	for _, item := range list.Items {
		res = append(res, item.Spec.CIDR)
	}
	return res, err
}
