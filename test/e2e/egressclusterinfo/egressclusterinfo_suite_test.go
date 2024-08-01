// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo_test

import (
	"context"
	config2 "sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	econfig "github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

func TestEgressgateway(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Egressgateway Suite")
}

var (
	config       *common.Config
	egressConfig econfig.FileConfig

	cli client.Client
)

var _ = BeforeSuite(func() {
	GinkgoRecover()

	var err error
	config, err = common.ReadConfig()
	Expect(err).NotTo(HaveOccurred())

	cfg := config2.GetConfigOrDie()
	cfg.QPS = 100
	cfg.Burst = 100
	cli, err = client.New(cfg, client.Options{
		Scheme: schema.GetScheme(),
	})
	Expect(err).NotTo(HaveOccurred())

	ctx := context.Background()

	// get egressgateway config
	configMap := &corev1.ConfigMap{}
	err = cli.Get(ctx, types.NamespacedName{Name: "egressgateway", Namespace: config.Namespace}, configMap)
	Expect(err).NotTo(HaveOccurred())

	raw, ok := configMap.Data["conf.yml"]
	Expect(ok).To(BeTrue(), "not found egress config file")

	err = yaml.Unmarshal([]byte(raw), &egressConfig)
	Expect(err).NotTo(HaveOccurred())
})
