// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"gopkg.in/yaml.v3"

	"github.com/spidernet-io/e2eframework/framework"
	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/test/e2e/err"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func GetEgressConfigmap(f *framework.Framework) (*config.FileConfig, error) {
	key := types.NamespacedName{
		Name:      EGRESSGATEWAY_CONFIGMAP_NAME,
		Namespace: Env[EGRESS_NAMESPACE],
	}
	cm := &corev1.ConfigMap{}
	e := f.GetResource(key, cm)
	if e != nil {
		return nil, e
	}
	if len(cm.Data) == 0 {
		return nil, err.NOT_FOUND
	}
	if _, ok := cm.Data[EGRESSGATEWAY_CONFIGMAP_KEY]; !ok {
		return nil, err.NOT_FOUND
	}
	data := cm.Data[EGRESSGATEWAY_CONFIGMAP_KEY]

	c := &config.FileConfig{}
	e = yaml.Unmarshal([]byte(data), c)
	if e != nil {
		return nil, e
	}
	return c, nil
}

func GetIPVersion(f *framework.Framework) (enableV4, enableV6 bool, e error) {
	c, e := GetEgressConfigmap(f)
	if e != nil {
		return
	}
	if !c.EnableIPv4 && !c.EnableIPv6 {
		return false, false, err.IPVERSION_ERR
	}
	return c.EnableIPv4, c.EnableIPv6, nil
}
