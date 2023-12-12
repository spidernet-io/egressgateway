// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"sort"

	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

func UpdateEgressClusterInfoNew(ctx context.Context, cli client.Client,
	obj *egressv1.EgressClusterInfo) error {

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("update %s %s/%s timeout",
				obj.GetObjectKind().GroupVersionKind().String(),
				obj.GetNamespace(), obj.GetName())
		default:
			err := cli.Update(ctx, obj)
			if err != nil {
				if !errors.IsConflict(err) {
					return err
				}
				time.Sleep(time.Second / 2)
				tmp := &egressv1.EgressClusterInfo{}
				key := types.NamespacedName{
					Namespace: obj.GetNamespace(),
					Name:      obj.GetName(),
				}
				err = cli.Get(ctx, key, tmp)
				if err != nil {
					return err
				}
				obj.ResourceVersion = tmp.ResourceVersion
				continue
			}
			return nil
		}
	}
}

func CheckEgressClusterInfoStatusSynced(
	ctx context.Context, cli client.Client,
	eci *egressv1.EgressClusterInfo,
) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	// get calico ip pool
	// get node ip list
	// get clusterIP list

	// check all internal ip synced

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("check EgressClusterInfo status synced timeout")
		default:
			key := types.NamespacedName{Name: eci.Name}
			err := cli.Get(ctx, key, eci)
			if err != nil {
				return nil
			}

			checked := 0

			exp := 0
			if eci.Spec.AutoDetect.ClusterIP {
				exp++
				// do check
				clusterIPv4CIDRs, clusterIPv6CIDRs, err := GetClusterCIDR(ctx, cli)
				if err != nil {
					return err
				}
				if eci.Status.ClusterIP == nil {
					return fmt.Errorf("error: empty  eci.Status.ClusterIP")
				}
				err1 := diffTwoSlice(eci.Status.ClusterIP.IPv4, clusterIPv4CIDRs)
				err2 := diffTwoSlice(eci.Status.ClusterIP.IPv6, clusterIPv6CIDRs)
				if err1 == nil && err2 == nil {
					checked++
				}
			}
			if eci.Spec.AutoDetect.NodeIP {
				exp++
				// do check
				nodeIPv4List, nodeIPv6List, err := GetAllNodesIPNew(ctx, cli)
				if err != nil {
					return err
				}

				list := make([]string, 0)
				for _, v := range eci.Status.NodeIP {
					list = append(list, v.IPv4...)
					list = append(list, v.IPv6...)
				}

				err = diffTwoSlice(list, append(nodeIPv4List, nodeIPv6List...))
				if err == nil {
					checked++
				}
			}
			if eci.Spec.AutoDetect.PodCidrMode == "calico" {
				exp++
				// do check
				res, err := ListCalicoIPPool(ctx, cli)
				if err != nil {
					return err
				}

				list := make([]string, 0)
				for _, v := range eci.Status.PodCIDR {
					list = append(list, v.IPv4...)
					list = append(list, v.IPv6...)
				}

				err = diffTwoSlice(res, list)
				if err == nil {
					checked++
				}
			}
			if tools.IsSameSlice(eci.Spec.ExtraCidr, eci.Status.ExtraCidr) {
				exp++
				checked++
			}

			if checked == exp {
				return nil
			}
			return nil
		}
	}
}

func diffTwoSlice(a []string, b []string) error {
	if len(a) != len(b) {
		return fmt.Errorf("arrays have different lengths")
	}

	// Sort the arrays in ascending order
	sort.Strings(a)
	sort.Strings(b)

	// Compare each element in the sorted arrays
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return fmt.Errorf("arrays are not identical")
		}
	}

	return nil
}
