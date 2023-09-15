// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("Test default egress gateway", Label("DefaultEgressGateway", "G00017"), Ordered, func() {
	var clusterDefaultEgw *egressv1.EgressGateway
	var nsDefaultEgw *egressv1.EgressGateway
	var policy1 *egressv1.EgressPolicy
	var policy2 *egressv1.EgressPolicy

	BeforeAll(func() {
		ipPool := egressv1.Ippools{}
		if egressConfig.EnableIPv4 {
			ipPool.IPv4 = []string{"10.99.0.1"}
		}
		if egressConfig.EnableIPv6 {
			ipPool.IPv6 = []string{"4c83:a33d:f5e5:d0b5:b76e:117c:ba98:7518"}
		}
		clusterDefaultEgw = &egressv1.EgressGateway{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-default",
			},
			Spec: egressv1.EgressGatewaySpec{
				ClusterDefault: true,
				Ippools:        ipPool,
				NodeSelector: egressv1.NodeSelector{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/hostname": "egressgateway-control-plane",
						}}}},
		}

		nsDefaultEgw = clusterDefaultEgw.DeepCopy()
		nsDefaultEgw.Name = "ns-default"
		nsDefaultEgw.Spec.ClusterDefault = false
		if egressConfig.EnableIPv4 {
			nsDefaultEgw.Spec.Ippools.IPv4 = []string{"10.99.0.2"}
		}
		if egressConfig.EnableIPv6 {
			nsDefaultEgw.Spec.Ippools.IPv6 = []string{"4c83:a11d:f5e5:d0b5:b76e:117c:ba98:7518"}
		}

		policy1 = &egressv1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-default-egw-policy1",
				Namespace: "default",
			},
			Spec: egressv1.EgressPolicySpec{
				EgressIP: egressv1.EgressIP{},
				AppliedTo: egressv1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "mock-app"},
					},
				},
				DestSubnet: []string{"10.6.0.92/32"},
			},
		}
		policy2 = policy1.DeepCopy()
		policy2.Name = "test-default-egw-policy2"

		DeferCleanup(func() {
			ctx := context.Background()

			err := common.DeleteObj(ctx, cli, policy2)
			Expect(err).NotTo(HaveOccurred())

			err = common.DeleteObj(ctx, cli, policy1)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(time.Second * 3)

			err = common.DeleteObj(ctx, cli, nsDefaultEgw)
			Expect(err).NotTo(HaveOccurred())

			err = common.DeleteObj(ctx, cli, clusterDefaultEgw)
			Expect(err).NotTo(HaveOccurred())

			ns := &corev1.Namespace{}
			key := types.NamespacedName{Name: "default"}
			err = cli.Get(ctx, key, ns)
			Expect(err).NotTo(HaveOccurred())
			_, ok := ns.Labels[egressv1.LabelNamespaceEgressGatewayDefault]
			if ok {
				delete(ns.Labels, egressv1.LabelNamespaceEgressGatewayDefault)
				err = cli.Update(ctx, ns)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	It("test cluster default egress gateway", func() {
		// create global egress gateway
		ctx := context.Background()
		err := cli.Create(ctx, clusterDefaultEgw)
		Expect(err).NotTo(HaveOccurred())

		// create egress policy1
		err = cli.Create(ctx, policy1)
		Expect(err).NotTo(HaveOccurred())

		// check policy1 is bind to cluster default
		key := types.NamespacedName{Namespace: policy1.Namespace, Name: policy1.Name}
		err = cli.Get(ctx, key, policy1)
		Expect(err).NotTo(HaveOccurred())
		Expect(policy1.Spec.EgressGatewayName).To(Equal(clusterDefaultEgw.Name))
	})

	It("test namespace default egress gateway", func() {
		// create the default egress gateway of default ns
		ctx := context.Background()
		err := cli.Create(ctx, nsDefaultEgw)
		Expect(err).NotTo(HaveOccurred())

		ns := &corev1.Namespace{}
		key := types.NamespacedName{Name: "default"}
		err = cli.Get(ctx, key, ns)
		Expect(err).NotTo(HaveOccurred())

		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		val, ok := ns.Labels[egressv1.LabelNamespaceEgressGatewayDefault]
		if !ok || val != nsDefaultEgw.Name {
			ns.Labels[egressv1.LabelNamespaceEgressGatewayDefault] = nsDefaultEgw.Name
			err = cli.Update(ctx, ns)
			Expect(err).NotTo(HaveOccurred())
		}
		err = cli.Create(ctx, policy2)
		Expect(err).NotTo(HaveOccurred())

		// check policy2 is bind to namespace default
		key = types.NamespacedName{Namespace: policy2.Namespace, Name: policy2.Name}
		err = cli.Get(ctx, key, policy2)
		Expect(err).NotTo(HaveOccurred())
		Expect(policy2.Spec.EgressGatewayName).To(Equal(nsDefaultEgw.Name))
	})

})
