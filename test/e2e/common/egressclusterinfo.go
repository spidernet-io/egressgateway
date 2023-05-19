// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"

	"github.com/spidernet-io/egressgateway/pkg/utils"
	e "github.com/spidernet-io/egressgateway/test/e2e/err"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	"github.com/spidernet-io/e2eframework/framework"
	egressv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
)

func GetEgressClusterInfo(f *framework.Framework, name string, egressClusterInfo *egressv1beta1.EgressClusterInfo) error {
	key := client.ObjectKey{
		Name: name,
	}
	return f.GetResource(key, egressClusterInfo)
}

func WaitEgressClusterInfoPodCidrUpdated(f *framework.Framework, oldEci *egressv1beta1.EgressClusterInfo, podType string, timeout time.Duration) (*egressv1beta1.EgressClusterInfo, error) {
	var podV4Cidr, podV6Cidr []string
	var v4ok, v6ok bool
	switch podType {
	case CALICO:
		podV4Cidr, podV6Cidr = GetCalicoIPPoolsCidr(f)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	eci := new(egressv1beta1.EgressClusterInfo)
	for {
		select {
		case <-ctx.Done():
			return nil, e.TIME_OUT
		default:
			err := GetEgressClusterInfo(f, oldEci.Name, eci)
			if err != nil {
				return nil, err
			}

			if newPodCidrV4 := eci.Status.EgressIgnoreCIDR.PodCIDR.IPv4; newPodCidrV4 != nil {
				v4ok, err = utils.IsSameIPCidrs(podV4Cidr, newPodCidrV4)
				if err != nil {
					return nil, err
				}
			} else {
				if len(podV4Cidr) == 0 {
					v4ok = true
				}
			}
			if newPodCidrV6 := eci.Status.EgressIgnoreCIDR.PodCIDR.IPv6; newPodCidrV6 != nil {
				v6ok, err = utils.IsSameIPCidrs(podV6Cidr, newPodCidrV6)
				if err != nil {
					return nil, err
				}
			} else {
				if len(podV6Cidr) == 0 {
					v6ok = true
				}
			}

			if oldEci.ResourceVersion != eci.ResourceVersion && v4ok && v6ok {
				return eci, nil
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}
