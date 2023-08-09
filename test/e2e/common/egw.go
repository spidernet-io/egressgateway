// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-faker/faker/v4"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	econfig "github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
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
	name := "egw-" + faker.Word()
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

// CheckEgressGatewayStatusSynced after some operations that affect the gateway, do a final verification of EgressGatewayStatus
func CheckEgressGatewayStatusSynced(ctx context.Context, cli client.Client, egw *egressv1.EgressGateway, expectStatus *egressv1.EgressGatewayStatus, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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
			if reflect.DeepEqual(*expectStatus, egw.Status) {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}
