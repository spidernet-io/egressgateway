// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-faker/faker/v4"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateEgressGatewayAndPodsBeforeEach(ctx context.Context, cli client.Client, enableIPv4, enableIPv6 bool, nodeNameList []string, podImg string, IPNum int64, increase uint8) (*egressv1.EgressGateway, []*corev1.Pod, error) {
	// create egressGateway
	pool, err := GenIPPools(ctx, cli, enableIPv4, enableIPv6, IPNum, increase)
	if err != nil {
		return nil, nil, err
	}

	labels := map[string]string{"ip-test": faker.Word()}
	err = LabelNodes(ctx, cli, nodeNameList, labels)
	if err != nil {
		return nil, nil, err
	}
	nodeSelector := egressv1.NodeSelector{Selector: &metav1.LabelSelector{MatchLabels: labels}}

	egw, err := CreateGatewayNew(ctx, cli, "egw-"+faker.Word(), pool, nodeSelector)
	if err != nil {
		return nil, nil, err
	}
	// create pods
	pods := CreatePods(ctx, cli, podImg, int(IPNum))
	return egw, pods, nil
}
