// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package reliability_test

import (
	"context"
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

func TestReliability(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reliability Suite")
}

var (
	config       *common.Config
	egressConfig econfig.FileConfig

	cli client.Client

	nodeNameList, workerNodes []string
	nodeLabel                 map[string]string
)

var _ = BeforeSuite(func() {
	GinkgoRecover()

	var err error
	config, err = common.ReadConfig()
	Expect(err).NotTo(HaveOccurred())

	cli, err = client.New(config.KubeConfigFile, client.Options{Scheme: schema.GetScheme()})
	Expect(err).NotTo(HaveOccurred())

	ctx := context.Background()

	// check nodes
	nodes := &corev1.NodeList{}
	err = cli.List(ctx, nodes)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(nodes.Items) > 2).To(BeTrue(), "test case needs at lest 3 nodes")

	for _, item := range nodes.Items {
		nodeNameList = append(nodeNameList, item.Name)
		if _, ok := item.Labels["node-role.kubernetes.io/control-plane"]; !ok {
			workerNodes = append(workerNodes, item.Name)
		}
	}
	Expect(len(workerNodes) > 1).To(BeTrue(), "this test case needs at lest 2 worker nodes")

	nodeLabel = nodes.Items[0].Labels

	// get egressgateway config
	configMap := &corev1.ConfigMap{}
	err = cli.Get(ctx, types.NamespacedName{Name: "egressgateway", Namespace: config.Namespace}, configMap)
	Expect(err).NotTo(HaveOccurred())

	raw, ok := configMap.Data["conf.yml"]
	Expect(ok).To(BeTrue(), "not found egress config file")

	err = yaml.Unmarshal([]byte(raw), &egressConfig)
	Expect(err).NotTo(HaveOccurred())
})
