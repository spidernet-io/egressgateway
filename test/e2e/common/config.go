// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	KwokTaintKey    = "kwok.x-k8s.io/node"
	KwokTaintEffect = "NoSchedule"
)

var (
	KwokNodeLabel = map[string]string{"type": "kwok"}
	KwokNodeTaint = corev1.Taint{Effect: KwokTaintEffect, Key: KwokTaintKey}
)

const (
	EGRESS_VXLAN_INTERFACE_NAME = "egress.vxlan"

	AVERAGE_SELECTION = "AverageSelection"
)

// egressClusterInfo
const (
	Calico = "calico"
	K8s    = "k8s"
	Auto   = "auto"
)

type Config struct {
	Image   string `mapstructure:"IMAGE"`
	TcpPort int    `mapstructure:"TCP_PORT"`
	UdpPort int    `mapstructure:"UDP_PORT"`
	WebPort int    `mapstructure:"WEB_PORT"`
	Mod     string `mapstructure:"MOD"`

	// for A/B testing
	ServerAIPv4 string `mapstructure:"SERVER_A_IPV4"`
	ServerAIPv6 string `mapstructure:"SERVER_A_IPV6"`
	ServerBIPv4 string `mapstructure:"SERVER_B_IPV4"`
	ServerBIPv6 string `mapstructure:"SERVER_B_IPV6"`

	Namespace      string       `mapstructure:"E2E_NAMESPACE"`
	KubeConfigPath string       `mapstructure:"KUBECONFIG"`
	KubeConfigFile *rest.Config `json:"-"`
}

func ReadConfig() (*Config, error) {
	config := &Config{
		Image:       "",
		TcpPort:     63380,
		UdpPort:     63381,
		WebPort:     63382,
		Mod:         "all",
		ServerAIPv4: "",
		ServerAIPv6: "",
		ServerBIPv4: "",
		ServerBIPv6: "",
		Namespace:   "egressgateway",
	}

	// map environment variables to struct objects
	envKeysMap := &map[string]interface{}{}
	err := mapstructure.Decode(config, &envKeysMap)
	if err != nil {
		return nil, err
	}
	for k := range *envKeysMap {
		if err := viper.BindEnv(k); err != nil {
			return nil, err
		}
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	config.KubeConfigFile, err = ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	return config, nil
}
