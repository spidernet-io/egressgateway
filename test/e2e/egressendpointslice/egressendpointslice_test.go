// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressendpointslice_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("Egressendpointslice", func() {
	/*
		Testing the impact of adding and deleting Pods on EndpointSlices and Pod egress IP.

		1. Create a gateway.
		2. Create a policy.
		3. Create and delete 10 Pods.
		4. Verify the status of EndpointSlices and the egress IP of Pods.
		5. When deleting all Pods, EndpointSlices should be deleted as well.
	*/
	Context("After performing several rounds of creating and deleting pods, check the status of the egress endpointSlice and the exported IP of the pod", Serial, Label("S00001"), func() {
		// deploy
		var (
			deploy *appsv1.Deployment
			podNum int
		)
		// gateway
		var egw *egressv1.EgressGateway
		// policy
		var egp *egressv1.EgressPolicy
		var egcp *egressv1.EgressClusterPolicy
		// error
		var err error
		// context
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
			egwName := "egw-" + uuid.NewString()

			// podNum
			podNum = 10

			// create egressGateway
			GinkgoWriter.Printf("create gateway %s\n", egwName)
			pool, err := common.GenIPPools(ctx, cli, egressConfig.EnableIPv4, egressConfig.EnableIPv6, 3, 1)
			Expect(err).NotTo(HaveOccurred(), "failed to generate pool")

			nodeSelector := egressv1.NodeSelector{Selector: &metav1.LabelSelector{MatchLabels: nodeLabel}}

			egw, err = common.CreateGatewayNew(ctx, cli, egwName, pool, nodeSelector)
			Expect(err).NotTo(HaveOccurred(), "failed to create egressGateway %s\n", egwName)

			DeferCleanup(func() {
				// delete the deploy if its exists
				if deploy != nil {
					GinkgoWriter.Printf("delete the deploy %s if its exists\n", deploy.Name)
					Expect(common.DeleteObj(ctx, cli, deploy)).NotTo(HaveOccurred())
				}

				// delete the policy if its exists
				if egp != nil {
					GinkgoWriter.Printf("delete the policy %s if its exists\n", egp.Name)
					Expect(common.WaitEgressPoliciesDeleted(ctx, cli, []*egressv1.EgressPolicy{egp}, time.Second*10)).NotTo(HaveOccurred())
				}

				// delete the cluster policy if its exists
				if egcp != nil {
					GinkgoWriter.Printf("delete the cluster policy %s if its exists\n", egcp.Name)
					Expect(common.WaitEgressClusterPoliciesDeleted(ctx, cli, []*egressv1.EgressClusterPolicy{egcp}, time.Second*10)).NotTo(HaveOccurred())
				}

				// delete the egressGateway if its exists
				if egw != nil {
					GinkgoWriter.Printf("delete the gatgeway %s if its exists\n", egw.Name)
					Expect(common.DeleteEgressGateway(ctx, cli, egw, time.Minute/2)).NotTo(HaveOccurred())
				}
			})
		})

		It("test the namespace-level policy", func() {
			// create deploy
			deployName := "deploy-" + uuid.NewString()
			deploy, err = common.CreateDeploy(ctx, cli, deployName, config.Image, podNum, time.Second*20)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to create deploy %s\n", deployName))
			GinkgoWriter.Printf("succeeded to create deploy %s\n", deploy.Name)

			// create policy
			egp, err = common.CreateEgressPolicyNew(ctx, cli, egressConfig, egw.Name, deploy.Labels, "")
			Expect(err).NotTo(HaveOccurred(), "failed to create policy")
			GinkgoWriter.Printf("succeeded to create policy %s\n", egp.Name)

			// delete deploy
			err := common.WaitDeployDeleted(ctx, cli, deploy, time.Second*10)
			Expect(err).NotTo(HaveOccurred())

			// create deploy again
			deploy, err = common.CreateDeploy(ctx, cli, deployName, config.Image, podNum, time.Second*20)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to create deploy %s\n", deployName))
			GinkgoWriter.Printf("succeeded to create deploy %s\n", deploy.Name)

			// check egressEndpointSlice synced
			GinkgoWriter.Println("check the egressEndpointSlice synced with the pod list")
			err = common.WaitForEgressEndPointSliceStatusSynced(ctx, cli, egp, time.Second*20)
			Expect(err).NotTo(HaveOccurred())

			// check the egress ip of pods
			GinkgoWriter.Println("check the egress ip of pods")
			err = common.CheckDeployEgressIP(ctx, cli, config, egressConfig, deploy, egp.Status.Eip.Ipv4, egp.Status.Eip.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())

			// delete the deploy, and we expect the egressEndpointSlice to be deleted as well
			err = common.WaitDeployDeleted(ctx, cli, deploy, time.Second*10)
			Expect(err).NotTo(HaveOccurred())

			Eventually(ctx, func() string {
				ees, _ := common.GetEgressEndPointSliceByEgressPolicy(ctx, cli, egp)
				return ees.Name
			}).WithTimeout(time.Second * 10).WithPolling(time.Second * 2).Should(BeEmpty())
		})

		It("test the cluster-level policy", func() {
			// create deploy
			deployName := "deploy-" + uuid.NewString()
			deploy, err = common.CreateDeploy(ctx, cli, deployName, config.Image, podNum, time.Second*20)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to create deploy %s\n", deployName))
			GinkgoWriter.Printf("succeeded to create deploy %s\n", deploy.Name)

			// create cluster policy
			egcp, err = common.CreateEgressClusterPolicy(ctx, cli, egressConfig, egw.Name, deploy.Labels)
			Expect(err).NotTo(HaveOccurred(), "failed to create cluster policy")
			GinkgoWriter.Printf("succeeded to create cluster policy %s\n", egcp.Name)

			// delete deploy
			err := common.WaitDeployDeleted(ctx, cli, deploy, time.Second*10)
			Expect(err).NotTo(HaveOccurred())

			// create deploy again
			deploy, err = common.CreateDeploy(ctx, cli, deployName, config.Image, podNum, time.Second*20)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to create deploy %s\n", deployName))
			GinkgoWriter.Printf("succeeded to create deploy %s\n", deploy.Name)

			// check egressClusterEndpointSlice synced
			GinkgoWriter.Println("check the egressClusterEndpointSlice synced with the pod list")
			err = common.WaitForEgressClusterEndPointSliceStatusSynced(ctx, cli, egcp, time.Second*20)
			Expect(err).NotTo(HaveOccurred())

			// check the egress ip of pods
			GinkgoWriter.Println("check the egress ip of pods")
			err = common.CheckDeployEgressIP(ctx, cli, config, egressConfig, deploy, egcp.Status.Eip.Ipv4, egcp.Status.Eip.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())

			// delete the deploy, and we expect the egressClusterEndpointSlice to be deleted as well
			err = common.WaitDeployDeleted(ctx, cli, deploy, time.Second*10)
			Expect(err).NotTo(HaveOccurred())
			Eventually(ctx, func() string {
				eces, _ := common.GetEgressClusterEndPointSliceByEgressClusterPolicy(ctx, cli, egcp)
				return eces.Name
			}).WithTimeout(time.Second * 10).WithPolling(time.Second * 2).Should(BeEmpty())
		})
	})
})
