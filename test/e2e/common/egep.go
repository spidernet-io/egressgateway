// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"time"

	apiserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	e2eerr "github.com/spidernet-io/egressgateway/test/e2e/err"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

func CheckEgressEndPointSliceStatus(ctx context.Context, cli client.Client, egp *egressv1.EgressPolicy) (bool, error) {
	ep, err := GetEgressEndPointSliceByEgressPolicy(ctx, cli, egp)
	if err != nil {
		return false, err
	}
	pl, err := GetPodsReferencedEgressPolicy(ctx, cli, egp)
	if err != nil {
		return false, err
	}
	if len(ep.Endpoints) != len(pl.Items) {
		return false, nil
	}
	pod2Ep := getEgressEndPointMap(ep)

	ok := 0
	for _, v := range pl.Items {
		if pod2Ep[v.Name].Pod != v.Name ||
			pod2Ep[v.Name].Namespace != v.Namespace ||
			pod2Ep[v.Name].Node != v.Spec.NodeName {
			ok++
		}
		ipv4s, ipv6s := GetPodIPs(&v)
		if !tools.IsSameSlice(pod2Ep[v.Name].IPv4, ipv4s) || !tools.IsSameSlice(pod2Ep[v.Name].IPv6, ipv6s) {
			ok++
		}
		if ok != 0 {
			break
		}
	}
	return ok == 0, nil
}

func CheckEgressClusterEndPointSliceStatus(ctx context.Context, cli client.Client, egcp *egressv1.EgressClusterPolicy) (bool, error) {
	ep, err := GetEgressClusterEndPointSliceByEgressClusterPolicy(ctx, cli, egcp)
	if err != nil {
		return false, err
	}
	pl, err := GetPodsReferencedEgressClusterPolicy(ctx, cli, egcp)
	if err != nil {
		return false, err
	}
	if len(ep.Endpoints) != len(pl.Items) {
		return false, nil
	}
	pod2Ep := getEgressClusterEndPointMap(ep)

	ok := 0
	for _, v := range pl.Items {
		if pod2Ep[v.Name].Pod != v.Name ||
			pod2Ep[v.Name].Namespace != v.Namespace ||
			pod2Ep[v.Name].Node != v.Spec.NodeName {
			ok++
		}
		ipv4s, ipv6s := GetPodIPs(&v)
		if !tools.IsSameSlice(pod2Ep[v.Name].IPv4, ipv4s) || !tools.IsSameSlice(pod2Ep[v.Name].IPv6, ipv6s) {
			ok++
		}
		if ok != 0 {
			break
		}
	}
	return ok == 0, nil
}

func GetEgressEndPointSliceByEgressPolicy(ctx context.Context, cli client.Client, egp *egressv1.EgressPolicy) (*egressv1.EgressEndpointSlice, error) {
	list := new(egressv1.EgressEndpointSliceList)
	err := cli.List(ctx, list)
	if err != nil {
		return nil, err
	}

	res := new(egressv1.EgressEndpointSlice)
	for _, e := range list.Items {
		if e.GenerateName == egp.Name+"-" {
			res = &e
			break
		}
	}

	return res, nil
}

func GetEgressClusterEndPointSliceByEgressClusterPolicy(ctx context.Context, cli client.Client, egcp *egressv1.EgressClusterPolicy) (*egressv1.EgressClusterEndpointSlice, error) {
	list := new(egressv1.EgressClusterEndpointSliceList)
	err := cli.List(ctx, list)
	if err != nil {
		return nil, err
	}

	res := new(egressv1.EgressClusterEndpointSlice)
	for _, e := range list.Items {
		if e.GenerateName == egcp.Name+"-" {
			res = &e
			break
		}
	}

	return res, nil
}

type pod2Endpoints map[string]egressv1.EgressEndpoint

func getEgressEndPointMap(egep *egressv1.EgressEndpointSlice) pod2Endpoints {
	res := make(pod2Endpoints, 0)
	for _, v := range egep.Endpoints {
		res[v.Pod] = v
	}
	return res
}

func getEgressClusterEndPointMap(egcep *egressv1.EgressClusterEndpointSlice) pod2Endpoints {
	res := make(pod2Endpoints, 0)
	for _, v := range egcep.Endpoints {
		res[v.Pod] = v
	}
	return res
}

func WaitForEgressEndPointSliceStatusSynced(ctx context.Context, cli client.Client, egp *egressv1.EgressPolicy, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return e2eerr.ErrTimeout
		default:
			ok, err := CheckEgressEndPointSliceStatus(ctx, cli, egp)
			if err == nil && ok {
				return nil
			}
			time.Sleep(time.Second * 2)
		}
	}
}

func WaitForEgressClusterEndPointSliceStatusSynced(ctx context.Context, cli client.Client, egcp *egressv1.EgressClusterPolicy, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("check egress cluster endpoint slice status synced timeout")
		default:
			ok, err := CheckEgressClusterEndPointSliceStatus(ctx, cli, egcp)
			if err == nil && ok {
				return nil
			}
			time.Sleep(time.Second * 2)
		}
	}
}

func WaitEgressEndPointSliceDeleted(ctx context.Context, cli client.Client, egep *egressv1.EgressEndpointSlice, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := DeleteObj(ctx, cli, egep)
	if err != nil {
		if apiserrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return e2eerr.ErrTimeout
		default:
			err = cli.Get(ctx, types.NamespacedName{Name: egep.Name, Namespace: egep.Namespace}, &egressv1.EgressEndpointSlice{})
			if !apiserrors.IsNotFound(err) {
				time.Sleep(time.Second / 2)
				continue
			}
			return nil
		}
	}
}

func WaitEgressClusterEndPointSliceDeleted(ctx context.Context, cli client.Client, egcep *egressv1.EgressClusterEndpointSlice, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := DeleteObj(ctx, cli, egcep)
	if err != nil {
		if apiserrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return e2eerr.ErrTimeout
		default:
			err = cli.Get(ctx, types.NamespacedName{Name: egcep.Name, Namespace: egcep.Namespace}, &egressv1.EgressClusterEndpointSlice{})
			if !apiserrors.IsNotFound(err) {
				time.Sleep(time.Second / 2)
				continue
			}
			return nil
		}
	}
}
