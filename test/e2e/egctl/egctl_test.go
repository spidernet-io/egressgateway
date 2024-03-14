// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egctl

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/egressgateway/cmd/egctl/cmd"
	econfig "github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var (
	config       *common.Config
	egressConfig econfig.FileConfig

	cli client.Client

	nodeLabel map[string]string

	node1, node2 *corev1.Node
)

func TestEgctl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "egctl Suite")
}

var _ = BeforeSuite(func() {
	GinkgoRecover()

	var (
		egw   *egressv1.EgressGateway
		egp   *egressv1.EgressPolicy
		ipNum int
		pool  egressv1.Ippools
	)
	ctx := context.Background()

	BeforeEach(func() {
		var err error
		ipNum = 3

		// create EgressGateway
		pool, err = common.GenIPPools(ctx, cli, egressConfig.EnableIPv4, egressConfig.EnableIPv6, int64(ipNum), 10)
		Expect(err).NotTo(HaveOccurred())
		nodeSelector := egressv1.NodeSelector{Selector: &metav1.LabelSelector{MatchLabels: nodeLabel}}

		egw, err = common.CreateGatewayNew(ctx, cli, "egw-"+uuid.NewString(), pool, nodeSelector)
		Expect(err).NotTo(HaveOccurred())
		GinkgoWriter.Printf("Create EgressGateway: %s\n", egw.Name)

		egp, err = common.CreateEgressPolicyNew(ctx, cli, egressConfig, egw.Name, map[string]string{"app": "test"}, "")
		Expect(err).NotTo(HaveOccurred())
		GinkgoWriter.Printf("Create EgressPolicy: %s\n", egp.Name)

		DeferCleanup(func() {
			if egp != nil {
				err = common.DeleteEgressPolicies(ctx, cli, []*egressv1.EgressPolicy{egp})
				Expect(err).NotTo(HaveOccurred())
			}

			if egw != nil {
				err = common.DeleteEgressGateway(ctx, cli, egw, time.Minute/2)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	targetNode := node1.Name
	if egp.Status.Node == node1.Name {
		targetNode = node2.Name
	}

	vip := egp.Status.Eip.Ipv4
	if egp.Status.Eip.Ipv4 == "" {
		vip = egp.Status.Eip.Ipv6
	}

	err := cmd.MoveEgressIP(egw.Name, vip, targetNode)
	Expect(err).NotTo(HaveOccurred())

	err = checkPolicyReady(ctx, cli, egp, time.Minute/2, targetNode)
	Expect(err).NotTo(HaveOccurred())
})

func checkPolicyReady(ctx context.Context, cli client.Client, egp *egressv1.EgressPolicy,
	timeout time.Duration, expNode string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("")
		default:
			err := cli.Get(ctx, types.NamespacedName{Namespace: egp.Namespace, Name: egp.Name}, egp)
			if err != nil {
				return err
			}
			if egp.Status.Node != expNode {
				time.Sleep(time.Second / 2)
				continue
			}
			return nil
		}
	}
}

var _ = Describe("egctl", Serial, Label("C00001"), func() {
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
