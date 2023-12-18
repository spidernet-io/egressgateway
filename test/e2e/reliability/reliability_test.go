// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package reliability_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
	"github.com/spidernet-io/egressgateway/test/e2e/constant"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

var _ = Describe("Reliability", Serial, Label("Reliability"), func() {
	Context("Test the drift of the EIP", func() {
		var (
			ctx       context.Context
			egw       *egressv1.EgressGateway
			daemonSet *appsv1.DaemonSet
			policy    *egressv1.EgressPolicy
			egNodes   []string
			labels    map[string]string
			ipNum     int
			pool      egressv1.Ippools
		)

		BeforeEach(func() {
			var err error
			ctx = context.Background()

			egNodes = workerNodes
			labels = map[string]string{"eg-reliability": "true"}
			selector := egressv1.NodeSelector{Selector: &v1.LabelSelector{MatchLabels: labels}}

			err = common.LabelNodes(ctx, cli, egNodes, labels)
			Expect(err).NotTo(HaveOccurred())

			ipNum = 3
			pool, err = common.GenIPPools(ctx, cli, egressConfig.EnableIPv4, egressConfig.EnableIPv6, int64(ipNum), 2)
			Expect(err).NotTo(HaveOccurred())

			egw, err = common.CreateGatewayNew(ctx, cli, "egw-"+uuid.NewString(), pool, selector)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("succeeded to create the gateway: %s\n", egw.Name)

			// check default eip
			v4DefaultEip, v6DefaultEip, err := common.GetGatewayDefaultIP(ctx, cli, egw, egressConfig)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("v4DefaultEip: %s, v6DefaultEip: %s\n", v4DefaultEip, v6DefaultEip)

			// daemonSet
			daemonSet, err = common.CreateDaemonSet(ctx, cli, "ds-reliability-"+faker.Word(), config.Image)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("succeeded to create DaemonSet: %s\n", daemonSet.Name)

			// policy
			policy, err = common.CreateEgressPolicyNew(ctx, cli, egressConfig, egw.Name, daemonSet.Labels, "")
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("succeeded to create policy: %s\n", policy.Name)

			// check eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, daemonSet,
				policy.Status.Eip.Ipv4, policy.Status.Eip.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("succeeded to check the export IP of the daemonSet: %s\n", daemonSet.Name)
		})

		AfterEach(func() {
			GinkgoWriter.Println("delete daemonSet")
			err := common.DeleteObj(ctx, cli, daemonSet)
			Expect(err).NotTo(HaveOccurred())

			GinkgoWriter.Println("delete policy")
			err = common.DeleteObj(ctx, cli, policy)
			Expect(err).NotTo(HaveOccurred())

			GinkgoWriter.Println("delete gateway")
			err = common.DeleteObj(ctx, cli, egw)
			Expect(err).NotTo(HaveOccurred())

			// start up all nodes if some nodes not ready
			GinkgoWriter.Println("PowerOnNodesUntilClusterReady")
			Expect(common.PowerOnNodesUntilClusterReady(ctx, cli, workerNodes, time.Minute, time.Minute)).NotTo(HaveOccurred())

			// unlabel nodes
			GinkgoWriter.Println("unLabel nodes")
			Expect(common.UnLabelNodes(ctx, cli, egNodes, labels)).NotTo(HaveOccurred())
		})

		/*
			When a node with EIP is shut down, the EIP will take effect on another node that matches the NodeSelector.
			Additionally, the egressGatewayStatus will be updated as expected.
		*/
		It("Test EIP drift after the eip-node shut down", Serial, Label("R00005"), func() {
			// check gateway status
			GinkgoWriter.Println("check egress gateway status")
			GinkgoWriter.Printf("the gateway node is: %s\n", policy.Status.Node)
			gatewayNode := policy.Status.Node
			otherNodes := tools.SubtractionSlice(workerNodes, []string{gatewayNode})
			checkGatewayStatus(ctx, cli, pool, ipNum, gatewayNode, otherNodes, []string{}, policy, egw, time.Second*5)

			// shut down the eip node
			notReadyNodes := gatewayNode
			GinkgoWriter.Printf("shut down node: %s\n", gatewayNode)
			Expect(common.PowerOffNodeUntilNotReady(ctx, cli, gatewayNode, time.Minute, time.Minute)).NotTo(HaveOccurred())

			// check if eip drift after node shut down
			GinkgoWriter.Println("Check if eip drift after node shut down")
			Eventually(ctx, func(ctx context.Context, cli client.Client) (string, error) {
				err := cli.Get(ctx, types.NamespacedName{Name: policy.Name, Namespace: policy.Namespace}, policy)
				return policy.Status.Node, err
			}).WithArguments(ctx, cli).WithTimeout(time.Minute).ShouldNot(Equal(gatewayNode))

			// check gateway status
			GinkgoWriter.Println("check egress gateway status")
			GinkgoWriter.Printf("after shut down the gatewayNode, now the gateway node is: %s\n", policy.Status.Node)
			gatewayNode = policy.Status.Node
			otherNodes = tools.SubtractionSlice(workerNodes, []string{gatewayNode, notReadyNodes})
			checkGatewayStatus(ctx, cli, pool, ipNum, gatewayNode, otherNodes, []string{notReadyNodes}, policy, egw, time.Second*5)

			// check the running pod's export IP is eip
			GinkgoWriter.Println("Check the eip in running pods after shut down the eip node")
			checkNodes := tools.SubtractionSlice(nodeNameList, []string{notReadyNodes})
			err := common.CheckEgressIPOfNodesPodList(ctx, cli, config, egressConfig, daemonSet.Spec.Template.Labels,
				checkNodes, policy.Status.Eip.Ipv4, policy.Status.Eip.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("succeeded to check the export IP of pods running on ready nodes: %v\n", checkNodes)
		})

		/*
			1. Shut down all matching egress nodes and expect the pod's egress IP to become non-EIP, with the gateway's status being updated correctly.
			2. Power on one of the nodes and expect the EIP to take effect on that node, resulting in the pod's egress IP becoming the EIP.
			3. Power on all nodes and expect no EIP drift to occur.
		*/
		It("The EIP does not drift when each node is powered on sequentially", Serial, Label("R00006"), func() {
			// shut down all gateway node
			for _, node := range workerNodes {

				GinkgoWriter.Printf("shut down node: %s\n", node)
				Expect(common.PowerOffNodeUntilNotReady(ctx, cli, node, time.Minute, time.Minute)).NotTo(HaveOccurred())
				GinkgoWriter.Printf("succeeded to shut down the node: %s\n", node)
			}
			// check gateway status
			GinkgoWriter.Println("check egress gateway status")
			checkGatewayStatus(ctx, cli, pool, ipNum, "", []string{}, workerNodes, policy, egw, time.Second*5)

			// check eip
			GinkgoWriter.Println("Check the eip in running pods after shut down all gateway nodes")
			checkNodes := tools.SubtractionSlice(nodeNameList, workerNodes)
			err := common.CheckEgressIPOfNodesPodList(ctx, cli, config, egressConfig, daemonSet.Spec.Template.Labels,
				checkNodes, policy.Status.Eip.Ipv4, policy.Status.Eip.Ipv6, false)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("succeeded to check the export IP of pods running on ready nodes: %v\n", checkNodes)

			// check policy status
			GinkgoWriter.Println("Check the policy status")
			Eventually(ctx, func(ctx context.Context, cli client.Client) bool {
				p := new(egressv1.EgressPolicy)
				_ = cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, p)
				return p.Status.Node == "" && p.Status.Eip == egressv1.Eip{}
			}).WithArguments(ctx, cli).WithTimeout(time.Second*5).Should(BeTrue(), "Failed to check policy status")

			// start up one node
			gatewayNode := workerNodes[0]
			GinkgoWriter.Printf("PowerOnNodeUntilReady: %s\n", gatewayNode)
			Expect(common.PowerOnNodeUntilReady(ctx, cli, gatewayNode, time.Minute, time.Minute)).NotTo(HaveOccurred())

			// wait the policy status updated
			GinkgoWriter.Printf("wait the policy: %s status updated\n", policy.Name)
			Eventually(ctx, func(ctx context.Context, cli client.Client) bool {
				_ = cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, policy)
				return policy.Status.Node == gatewayNode
			}).WithArguments(ctx, cli).WithTimeout(time.Second * 10).Should(BeTrue())

			// check egress gateway status
			GinkgoWriter.Println("check egress gateway status")
			GinkgoWriter.Printf("after start up node: %s, now the gateway node is: %s\n", gatewayNode, gatewayNode)
			notReadyNodes := workerNodes[1:]
			checkGatewayStatus(ctx, cli, pool, ipNum, gatewayNode, []string{}, notReadyNodes, policy, egw, time.Second*5)

			// wait pods running after its node restarted
			err = common.WaitForNodesPodListRestarted(ctx, cli, daemonSet.Spec.Template.Labels, []string{gatewayNode}, time.Minute*3)
			Expect(err).NotTo(HaveOccurred())
			// check the running pod's export IP is eip
			GinkgoWriter.Println("Check the eip in running pods after start up one node")
			checkNodes = tools.SubtractionSlice(nodeNameList, notReadyNodes)
			err = common.CheckEgressIPOfNodesPodList(ctx, cli, config, egressConfig, daemonSet.Spec.Template.Labels,
				checkNodes, policy.Status.Eip.Ipv4, policy.Status.Eip.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("succeeded to check the export IP of pods running on ready nodes: %v\n", checkNodes)

			// power on the node and wait cluster ready
			GinkgoWriter.Println("PowerOnNodesUntilClusterReady")
			Expect(common.PowerOnNodesUntilClusterReady(ctx, cli, workerNodes, time.Minute, time.Minute)).NotTo(HaveOccurred())

			// expect the eip will not drift
			GinkgoWriter.Println("check the egress gateway status; the EIP should not drift")
			checkGatewayStatus(ctx, cli, pool, ipNum, gatewayNode, workerNodes[1:], []string{}, policy, egw, time.Second*5)
		})

		/*
			restart the components such as calico, etcd and kube-proxy
			check the eip of the pods
			check the status of the egress gateway crs
		*/
		DescribeTable("restart components", Serial, Label("R00007"), func(labels map[string]string, timeout time.Duration) {
			// get gateway
			beforeEgw := new(egressv1.EgressGateway)
			err := cli.Get(ctx, types.NamespacedName{Name: egw.Name}, beforeEgw)
			Expect(err).NotTo(HaveOccurred())

			// get policy
			beforPolicy := new(egressv1.EgressPolicy)
			err = cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, beforPolicy)
			Expect(err).NotTo(HaveOccurred())

			// get egressClusterInfo
			beforeEgci := new(egressv1.EgressClusterInfo)
			err = cli.Get(ctx, types.NamespacedName{Name: "default"}, beforeEgci)
			Expect(err).NotTo(HaveOccurred())

			// get egressEndPoints
			beforeEgep, err := common.GetEgressEndPointSliceByEgressPolicy(ctx, cli, policy)
			Expect(err).NotTo(HaveOccurred())

			var wg sync.WaitGroup
			var isRerun atomic.Bool
			wg.Add(1)

			go func() {
				defer GinkgoRecover()
				defer wg.Done()

				err := common.DeletePodsUntilReady(ctx, cli, labels, timeout)
				Expect(err).NotTo(HaveOccurred())
				isRerun.Store(true)
			}()

			for !isRerun.Load() {
				err := common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, daemonSet,
					policy.Status.Eip.Ipv4, policy.Status.Eip.Ipv6, true)
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Printf("succeeded to check the export IP of the daemonSet: %s\n", daemonSet.Name)
				time.Sleep(time.Second / 2)
			}
			wg.Wait()

			// check the gateway status
			GinkgoWriter.Println("check egress gateway status")
			nowEgw := new(egressv1.EgressGateway)
			err = cli.Get(ctx, types.NamespacedName{Name: egw.Name}, nowEgw)
			Expect(err).NotTo(HaveOccurred())
			Expect(nowEgw.Status).Should(Equal(beforeEgw.Status), fmt.Sprintf("expect %v\ngot: %v\n", beforeEgw.Status, nowEgw.Status))

			// check the policy status
			GinkgoWriter.Println("check egress policy status")
			nowPolicy := new(egressv1.EgressPolicy)
			err = cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, nowPolicy)
			Expect(err).NotTo(HaveOccurred())
			Expect(nowPolicy.Status).Should(Equal(beforPolicy.Status), fmt.Sprintf("expect:\n%v\ngot:\n %v\n", beforPolicy.Status, nowPolicy.Status))

			// check the egressClusterInfo status
			nowEgci := new(egressv1.EgressClusterInfo)
			err = cli.Get(ctx, types.NamespacedName{Name: "default"}, nowEgci)
			Expect(err).NotTo(HaveOccurred())
			Expect(nowEgci.Status).Should(Equal(beforeEgci.Status), fmt.Sprintf("expect:\n%v\ngot:\n %v\n", beforeEgci.Status, nowEgci.Status))

			// check the egressEndPointSlice
			// get egressEndPoints
			nowEgep, err := common.GetEgressEndPointSliceByEgressPolicy(ctx, cli, policy)
			Expect(err).NotTo(HaveOccurred())
			Expect(nowEgep.Endpoints).Should(Equal(beforeEgep.Endpoints), fmt.Sprintf("expect:\n%v\ngot:\n %v\n", beforeEgep.Endpoints, nowEgep.Endpoints))
		},
			Entry("restart kube-controller-manager", constant.KubeControllerManagerLabel, time.Minute),
			Entry("restart kube-apiserver", constant.KubeApiServerLabel, time.Minute),
			Entry("restart etcd", constant.KubeEtcdLabel, time.Minute),
			Entry("restart kube-scheduler", constant.KubeSchedulerLabel, time.Minute),
			Entry("restart kube-proxy", constant.KubeProxyLabel, time.Minute),
			Entry("restart calico-node", constant.CalicoNodeLabel, time.Minute),
			Entry("restart calico-kube-controllers", constant.CalicoControllerLabel, time.Minute),
		)
	})
})

func checkGatewayStatus(ctx context.Context, cli client.Client, pool egressv1.Ippools, ipNum int, gatewayNode string, otherNodes, notReadyNodes []string, policy *egressv1.EgressPolicy, egw *egressv1.EgressGateway, timeout time.Duration) {
	expectGatewayStatus := new(egressv1.EgressGatewayStatus)
	// ipUsage
	ipUsage := egressv1.IPUsage{}
	if gatewayNode != "" {
		if len(pool.IPv4) > 0 {
			ipUsage.IPv4Free = ipNum - 1
			ipUsage.IPv4Total = ipNum
		}
		if len(pool.IPv6) > 0 {
			ipUsage.IPv6Free = ipNum - 1
			ipUsage.IPv6Total = ipNum
		}
	} else {
		if len(pool.IPv4) > 0 {
			ipUsage.IPv4Free = ipNum
			ipUsage.IPv4Total = ipNum
		}
		if len(pool.IPv6) > 0 {
			ipUsage.IPv6Free = ipNum
			ipUsage.IPv6Total = ipNum
		}
	}
	expectGatewayStatus.IPUsage = ipUsage

	// NodeList
	nodeList := make([]egressv1.EgressIPStatus, 0)
	if gatewayNode != "" {
		gatewayNodeStatus := egressv1.EgressIPStatus{
			Name: gatewayNode,
			Eips: []egressv1.Eips{
				{
					IPv4: policy.Status.Eip.Ipv4,
					IPv6: policy.Status.Eip.Ipv6,
					Policies: []egressv1.Policy{
						{Name: policy.Name, Namespace: policy.Namespace},
					},
				},
			},
			Status: string(egressv1.EgressTunnelReady),
		}
		nodeList = append(nodeList, gatewayNodeStatus)
	}

	for _, node := range otherNodes {
		item := egressv1.EgressIPStatus{
			Name:   node,
			Status: string(egressv1.EgressTunnelReady),
		}
		nodeList = append(nodeList, item)
	}
	for _, node := range notReadyNodes {
		item := egressv1.EgressIPStatus{
			Name:   node,
			Status: string(egressv1.EgressTunnelNodeNotReady),
		}
		nodeList = append(nodeList, item)
	}
	expectGatewayStatus.NodeList = nodeList

	err := common.CheckEgressGatewayStatusSynced(ctx, cli, egw, expectGatewayStatus, timeout)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("expect: %v\ngot: %v\n", *expectGatewayStatus, egw.Status))
	GinkgoWriter.Println("succeeded to check gateway status")
}
