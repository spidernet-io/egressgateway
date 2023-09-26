// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClusterCIDR(ctx context.Context, cli client.Client) (ipv4, ipv6 []string, err error) {
	key := types.NamespacedName{Namespace: "kube-system", Name: "kubeadm-config"}
	configMap := &corev1.ConfigMap{}
	err = cli.Get(ctx, key, configMap)
	if err != nil {
		return
	}

	v, ok := configMap.Data["ClusterConfiguration"]
	if !ok {
		err = fmt.Errorf("can get kube-system configmap")
		return
	}

	c := &KubeAdmConfig{}
	err = yaml.Unmarshal([]byte(v), c)
	if err != nil {
		return
	}

	res := strings.Split(c.Networking.ServiceSubnet, ",")
	if len(res) == 2 {
		ipv4 = []string{res[0]}
		ipv6 = []string{res[1]}
	} else if strings.Contains(c.Networking.ServiceSubnet, ":") {
		ipv6 = []string{c.Networking.ServiceSubnet}
	} else if strings.Contains(c.Networking.ServiceSubnet, ",") {
		ipv4 = []string{c.Networking.ServiceSubnet}
	}

	return
}

type KubeAdmConfig struct {
	Networking Networking `yaml:"networking"`
}

type Networking struct {
	ServiceSubnet string `yaml:"serviceSubnet"`
}
