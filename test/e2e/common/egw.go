// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	econfig "github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	e2eerr "github.com/spidernet-io/egressgateway/test/e2e/err"
)

func CreateGatewayNew(ctx context.Context, cli client.Client,
	name string, pool egressv1.Ippools, selector egressv1.NodeSelector) (*egressv1.EgressGateway, error) {
	res := &egressv1.EgressGateway{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       egressv1.EgressGatewaySpec{Ippools: pool, NodeSelector: selector},
	}
	err := cli.Create(ctx, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func CreateGatewayCustom(ctx context.Context, cli client.Client, setUp func(egw *egressv1.EgressGateway)) (*egressv1.EgressGateway, error) {
	name := "egw-" + uuid.NewString()
	res := &egressv1.EgressGateway{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}

	setUp(res)

	err := cli.Create(ctx, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func GetGatewayDefaultIP(ctx context.Context, cli client.Client,
	egw *egressv1.EgressGateway, cfg econfig.FileConfig) (v4 string, v6 string, err error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			err = fmt.Errorf("get default egress ip timeout")
			return
		default:
			err = cli.Get(ctx, types.NamespacedName{Name: egw.Name}, egw)
			if err != nil {
				return
			}
			v4 = egw.Spec.Ippools.Ipv4DefaultEIP
			v6 = egw.Spec.Ippools.Ipv6DefaultEIP

			if cfg.EnableIPv4 && cfg.EnableIPv6 {
				if v4 == "" || v6 == "" {
					time.Sleep(time.Second)
					continue
				} else {
					return
				}
			}

			if cfg.EnableIPv4 {
				if v4 == "" {
					time.Sleep(time.Second)
					continue
				} else {
					return
				}
			}

			if cfg.EnableIPv6 {
				if v6 == "" {
					time.Sleep(time.Second)
					continue
				} else {
					return
				}
			}

			return
		}
	}
}

func UpdateEgressGateway(ctx context.Context, cli client.Client, gateway *egressv1.EgressGateway) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("create EgressGateway timeout")
		default:
			err := cli.Update(ctx, gateway)
			if err != nil {
				if !errors.IsConflict(err) {
					return err
				}
				time.Sleep(time.Second / 2)
				tmp := new(egressv1.EgressGateway)
				err = cli.Get(ctx, types.NamespacedName{Name: gateway.Name}, tmp)
				if err != nil {
					return err
				}
				gateway.ResourceVersion = tmp.ResourceVersion
				continue
			}
			return nil
		}
	}
}

// CheckEGWSyncedWithEGP check if egw status synced with egp status when egp's allocatorPolicy is "rr"
func CheckEGWSyncedWithEGP(cli client.Client, egw *egressv1.EgressGateway, checkV4, checkV6 bool, IPNum int) (bool, error) {
	eipV4s := make(map[string]struct{})
	eipV6s := make(map[string]struct{})
	for _, eipStatus := range egw.Status.NodeList {
		if checkV4 {
			for _, eip := range eipStatus.Eips {
				if len(eip.IPv4) != 0 {
					if _, ok := eipV4s[eip.IPv4]; ok {
						return false, fmt.Errorf("ip reallocate, the egw yaml:\n%s\n", GetObjYAML(egw))
					}
					eipV4s[eip.IPv4] = struct{}{}
				}
			}
		}
		if checkV6 {
			for _, eip := range eipStatus.Eips {
				if len(eip.IPv6) != 0 {
					if _, ok := eipV6s[eip.IPv6]; ok {
						return false, fmt.Errorf("ip reallocate, the egw yaml:\n%s\n", GetObjYAML(egw))
					}
					eipV6s[eip.IPv6] = struct{}{}
				}
			}
		}
		// check egw status synced with egp status
		for _, eips := range eipStatus.Eips {
			for _, policy := range eips.Policies {
				egp := new(egressv1.EgressPolicy)
				err := cli.Get(context.TODO(), types.NamespacedName{Name: policy.Name, Namespace: policy.Namespace}, egp)
				if err != nil {
					return false, err
				}
				if egp.Status.Node != eipStatus.Name {
					e := fmt.Errorf("Node is not synced, the egp is: %s, nodeName is: %s, but egw nodeName is: %s\nthe egw yaml:\n%s\n",
						egp.Name, egp.Status.Node, eipStatus.Name, GetObjYAML(egw))
					return false, e
				}
				if egp.Status.Eip.Ipv4 != eips.IPv4 {
					e := fmt.Errorf("Eip.Ipv4 is not synced, the egp is: %s, eipV4 is: %s, but egw IPv4 is: %s\nthe egw yaml:\n%s\n",
						egp.Name, egp.Status.Eip.Ipv4, eips.IPv4, GetObjYAML(egw))
					return false, e
				}
				if egp.Status.Eip.Ipv6 != eips.IPv6 {
					e := fmt.Errorf("Eip.Ipv6 is not synced, the egp is: %s, eipV6 is: %s, but egw IPv6 is: %s\nthe egw yaml:\n%s\n",
						egp.Name, egp.Status.Eip.Ipv6, eips.IPv6, GetObjYAML(egw))
					return false, e
				}
			}
		}
	}
	if checkV4 && checkV6 {
		if len(eipV4s) != IPNum || len(eipV6s) != IPNum {
			e := fmt.Errorf("failed check ip number, expect num is %v but got eipV4s num: %v, eipV6s num: %v\n", IPNum, len(eipV4s), len(eipV6s))
			return false, e
		}
		return true, nil
	}
	if checkV4 {
		if len(eipV4s) != IPNum {
			e := fmt.Errorf("failed check ip number, expect num is %v but got eipV4s num: %v\n", IPNum, len(eipV4s))
			return false, e
		}
		return true, nil
	}
	if len(eipV6s) != IPNum {
		e := fmt.Errorf("failed check ip number, expect num is %v but got eipV6s num: %v\n", IPNum, len(eipV6s))
		return false, e
	}
	return true, nil
}

// WaitEGWSyncedWithEGP
func WaitEGWSyncedWithEGP(cli client.Client, egw *egressv1.EgressGateway, checkV4, checkV6 bool, IPNum int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	var e error
	for {
		select {
		case <-ctx.Done():
			if e == nil {
				return e2eerr.ErrTimeout
			}
			return fmt.Errorf("egressGateway failed synced with egressPolicy, error: %v\n", e)
		default:
			err := cli.Get(ctx, types.NamespacedName{Name: egw.Name}, egw)
			if err != nil {
				time.Sleep(time.Second / 2)
				continue
			}
			ok, err := CheckEGWSyncedWithEGP(cli, egw, checkV4, checkV6, IPNum)
			if ok {
				return nil
			}
			if err != nil {
				e = err
				time.Sleep(time.Second / 2)
				continue
			}
		}
	}
}

// CheckEgressGatewayStatusSynced after some operations that affect the gateway, do a final verification of EgressGatewayStatus
func CheckEgressGatewayStatusSynced(ctx context.Context, cli client.Client, egw *egressv1.EgressGateway, expectStatus *egressv1.EgressGatewayStatus, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	expectStatusCP := expectStatus.DeepCopy()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("check EgressGatewayStatus synced timeout")
		default:
			key := types.NamespacedName{Name: egw.Name}
			err := cli.Get(ctx, key, egw)
			if err != nil {
				return nil
			}

			egwStatusCp := egw.Status.DeepCopy()
			expectNodeList := SortEgressGatewayNodeListByName(expectStatusCP.NodeList)
			egwNodeList := SortEgressGatewayNodeListByName(egwStatusCp.NodeList)

			if reflect.DeepEqual(expectNodeList, egwNodeList) && reflect.DeepEqual(expectStatusCP.IPUsage, egwStatusCp.IPUsage) {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func DeleteEgressGateway(ctx context.Context, cli client.Client, egw *egressv1.EgressGateway, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := DeleteObj(ctx, cli, egw)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return e2eerr.ErrTimeout
		default:
			err = cli.Get(ctx, types.NamespacedName{Name: egw.Name}, egw)
			if apierrors.IsNotFound(err) {
				return nil
			}
			time.Sleep(time.Second / 2)
		}
	}
}

func GetClusterDefualtEgressGateway(ctx context.Context, cli client.Client) (*egressv1.EgressGateway, error) {
	egwList := new(egressv1.EgressGatewayList)
	err := cli.List(ctx, egwList)
	if err != nil {
		return nil, err
	}
	for _, v := range egwList.Items {
		if v.Spec.ClusterDefault {
			return &v, nil
		}
	}
	return nil, nil
}

func SortEgressGatewayNodeListByName(nodes []egressv1.EgressIPStatus) []egressv1.EgressIPStatus {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})
	return nodes
}
