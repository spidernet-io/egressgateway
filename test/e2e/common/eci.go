// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"github.com/spidernet-io/egressgateway/pkg/controller/clusterinfo"
	"github.com/spidernet-io/egressgateway/pkg/utils"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
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

func CheckEgressClusterInfoStatusSynced(ctx context.Context,
	cli client.Client, eci *egressv1.EgressClusterInfo,
) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	clusterIPReady := false
	nodeIPReady := false
	calicoReady := false
	extraCIDRReady := false

	clusterIPReadyStr := ""

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("check EgressClusterInfo status synced timeout "+
				"clusterIPReady=%v nodeIPReady=%v calicoReady=%v extraCIDRReady=%v \n%s",
				clusterIPReady, nodeIPReady, calicoReady, extraCIDRReady, clusterIPReadyStr)
		default:
			key := types.NamespacedName{Name: eci.Name}
			err := cli.Get(ctx, key, eci)
			if err != nil {
				return fmt.Errorf("failed go get egress cluster info %w "+
					"clusterIPReady=%v nodeIPReady=%v calicoReady=%v extraCIDRReady=%v",
					err, clusterIPReady, nodeIPReady, calicoReady, extraCIDRReady)
			}

			if !clusterIPReady {
				if eci.Spec.AutoDetect.ClusterIP {
					clusterIPv4CIDRs, clusterIPv6CIDRs, err := clusterinfo.GetClusterCIDR(ctx, cli)
					if err != nil {
						continue
					}
					if eci.Status.ClusterIP == nil {
						continue
					}
					pass1 := utils.EqualStringSlice(eci.Status.ClusterIP.IPv4, clusterIPv4CIDRs)
					pass2 := utils.EqualStringSlice(eci.Status.ClusterIP.IPv6, clusterIPv6CIDRs)
					if pass1 && pass2 {
						clusterIPReady = true
					}

					clusterIPReadyStr = fmt.Sprintf("eci.Status.ClusterIP.IPv4=%v, clusterIPv4CIDRs=%v, "+
						"eci.Status.ClusterIP.IPv6=%v, clusterIPv6CIDRs=%v",
						eci.Status.ClusterIP.IPv4, clusterIPv4CIDRs,
						eci.Status.ClusterIP.IPv6, clusterIPv6CIDRs)

				} else {
					clusterIPReady = true
				}
			}

			if !nodeIPReady {
				if eci.Spec.AutoDetect.NodeIP {
					nodeIPv4List, nodeIPv6List, err := GetAllNodesIPNew(ctx, cli)
					if err != nil {
						return err
					}

					list := make([]string, 0)
					for _, v := range eci.Status.NodeIP {
						list = append(list, v.IPv4...)
						list = append(list, v.IPv6...)
					}

					if utils.EqualStringSlice(list, append(nodeIPv4List, nodeIPv6List...)) {
						nodeIPReady = true
					}
				} else {
					nodeIPReady = true
				}
			}

			if !calicoReady {
				if eci.Spec.AutoDetect.PodCidrMode == "calico" {
					res, err := ListCalicoIPPool(ctx, cli)
					if err != nil {
						return err
					}

					list := make([]string, 0)
					for _, v := range eci.Status.PodCIDR {
						list = append(list, v.IPv4...)
						list = append(list, v.IPv6...)
					}

					if utils.EqualStringSlice(res, list) {
						calicoReady = true
					}
				} else {
					calicoReady = true
				}
			}

			if !extraCIDRReady {
				if utils.EqualStringSlice(eci.Spec.ExtraCidr, eci.Status.ExtraCidr) {
					extraCIDRReady = true
				}
			}

			if clusterIPReady && nodeIPReady && calicoReady && extraCIDRReady {
				return nil
			}
			time.Sleep(time.Second * 4)
		}
	}
}
