// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	"github.com/spidernet-io/e2eframework/framework"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"

	"github.com/spidernet-io/egressgateway/pkg/utils"
	e "github.com/spidernet-io/egressgateway/test/e2e/err"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

func GetCalicoIPPool(f *framework.Framework, name string) (*calicov1.IPPool, error) {
	key := types.NamespacedName{Name: name}
	ippool := new(calicov1.IPPool)
	err := f.GetResource(key, ippool)
	if err != nil {
		return nil, err
	}
	return ippool, nil
}

func GetCalicoIPPools(f *framework.Framework) []calicov1.IPPool {
	IPPoolList := &calicov1.IPPoolList{}
	err := f.ListResource(IPPoolList)
	Expect(err).NotTo(HaveOccurred())
	Expect(IPPoolList).NotTo(BeNil())
	ippools := make([]calicov1.IPPool, 0)
	ippools = append(ippools, IPPoolList.Items...)
	return ippools
}

func GetCalicoIPPoolsCidr(f *framework.Framework) (v4Cidrs, v6Cidrs []string) {
	ippools := GetCalicoIPPools(f)
	v4Cidrs, v6Cidrs = make([]string, 0), make([]string, 0)
	for _, ippool := range ippools {
		cidr := ippool.Spec.CIDR
		isV4Cidr, err := utils.IsIPv4Cidr(cidr)
		Expect(err).NotTo(HaveOccurred())

		isV6Cidr, err := utils.IsIPv6Cidr(cidr)
		Expect(err).NotTo(HaveOccurred())

		if isV4Cidr {
			v4Cidrs = append(v4Cidrs, cidr)
		}
		if isV6Cidr {
			v6Cidrs = append(v6Cidrs, cidr)
		}
	}
	return
}

func CreateCalicoIPPool(f *framework.Framework, namePrefix, cidrPrefix string, GenerateRandomCidr func(_ string) string, opts ...client.CreateOption) *calicov1.IPPool {
	ippool := GenerateCalicoIPPoolYaml(namePrefix, cidrPrefix, GenerateRandomCidr)
	Expect(f.CreateResource(ippool, opts...)).NotTo(HaveOccurred())
	return ippool
}

func GenerateCalicoIPPoolYaml(namePrefix, cidrPrefix string, GenerateRandomCidr func(prefix string) string) *calicov1.IPPool {
	name := namePrefix + tools.GenerateRandomNumber(10000)
	return &calicov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: calicov1.IPPoolSpec{
			CIDR: GenerateRandomCidr(cidrPrefix),
		},
	}
}

func WaitCalicoIPPoolCreated(f *framework.Framework, name string, timeout time.Duration) (*calicov1.IPPool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return nil, e.TIME_OUT
		default:
			ippool, err := GetCalicoIPPool(f, name)
			if err != nil {
				time.Sleep(time.Millisecond * 100)
			}
			return ippool, nil
		}
	}
}

func DeleteCalicoIPPool(f *framework.Framework, ippool *calicov1.IPPool, opts ...client.DeleteOption) error {
	return f.DeleteResource(ippool, opts...)
}

func WaitCalicoIPPoolDeleted(f *framework.Framework, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return e.TIME_OUT
		default:
			_, err := GetCalicoIPPool(f, name)
			if err != nil {
				if errors.IsNotFound(err) {
					return nil
				}
				return err
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func UpdateCalicoIPPoolCidr(f *framework.Framework, ippool *calicov1.IPPool, cidrPrefix string, GenerateRandomCidr func(_ string) string, opts ...client.UpdateOption) (*calicov1.IPPool, error) {
	ippool.Spec.CIDR = GenerateRandomCidr(cidrPrefix)
	err := f.UpdateResource(ippool, opts...)
	if err != nil {
		return nil, err
	}
	return ippool, nil
}

func WaitCalicoIPPoolCidrUpdated(f *framework.Framework, updatedIPPool *calicov1.IPPool, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return e.TIME_OUT
		default:
			ippool, err := GetCalicoIPPool(f, updatedIPPool.Name)
			if err != nil {
				return err
			}
			if ippool.Spec.CIDR == updatedIPPool.Spec.CIDR {
				return nil
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}
