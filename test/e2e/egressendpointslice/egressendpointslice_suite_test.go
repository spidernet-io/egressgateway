// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressendpointslice_test

import (
	"context"
	config2 "sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	econfig "github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

func TestEgressPolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Egressendpointslice Suite")
}

var (
	config       *common.Config
	egressConfig econfig.FileConfig

	cli client.Client

	nodeLabel map[string]string

	node1, node2 *corev1.Node
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

	// check nodes
	nodes := &corev1.NodeList{}
	err = cli.List(ctx, nodes)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(nodes.Items) > 2).To(BeTrue(), "test case needs at lest 3 nodes")

	//
	nodeLabel = nodes.Items[0].Labels
	node1 = &nodes.Items[0]
	node2 = &nodes.Items[1]

	// get egressgateway config
	configMap := &corev1.ConfigMap{}
	err = cli.Get(ctx, types.NamespacedName{Name: "egressgateway", Namespace: config.Namespace}, configMap)
	Expect(err).NotTo(HaveOccurred())

	raw, ok := configMap.Data["conf.yml"]
	Expect(ok).To(BeTrue(), "not found egress config file")

	err = yaml.Unmarshal([]byte(raw), &egressConfig)
	Expect(err).NotTo(HaveOccurred())
})
