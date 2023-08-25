// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/e2eframework/framework"
	egcitools "github.com/spidernet-io/egressgateway/pkg/controller/egress_cluster_info"
	egressv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
	"github.com/spidernet-io/egressgateway/test/e2e/err"
)

const egciName = "default"

var er error

func GetEgressClusterInfo(f *framework.Framework, egressClusterInfo *egressv1beta1.EgressClusterInfo) error {
	key := client.ObjectKey{
		Name: egciName,
	}
	return f.GetResource(key, egressClusterInfo)
}

func UpdateEgressClusterInfo(f *framework.Framework, egressClusterInfo *egressv1beta1.EgressClusterInfo, timeout time.Duration, opts ...client.UpdateOption) error {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	egci := new(egressv1beta1.EgressClusterInfo)
	for {
		select {
		case <-ctx.Done():
			return ERR_TIMEOUT
		default:
			er = GetEgressClusterInfo(f, egci)
			if er != nil {
				return er
			}
			egressClusterInfo.ResourceVersion = egci.ResourceVersion
			er = f.UpdateResource(egressClusterInfo, opts...)
			if er == nil {
				GinkgoWriter.Printf("the latest resourceVersion is: %s\n", egci.ResourceVersion)
				return nil
			}
			if !errors.IsConflict(er) {
				return er
			}
			GinkgoWriter.Printf("conflict, need retry, now the request resourceVersion is: %s\n", egci.ResourceVersion)
			time.Sleep(time.Second)
		}
	}
}

func WaitEgressClusterInfoPodCidrAndModeUpdated(f *framework.Framework, podType string, timeout time.Duration) (*egressv1beta1.EgressClusterInfo, error) {
	var podV4Cidr, podV6Cidr []string
	var mode egressv1beta1.PodCidrMode
	var v4ok, v6ok bool
	switch podType {
	case Calico:
		podV4Cidr, podV6Cidr = GetCalicoIPPoolsCidr(f)
		mode = Calico
	case Auto:
		podV4Cidr, podV6Cidr = GetCalicoIPPoolsCidr(f)
		mode = Calico
	case K8s:
		podV4Cidr, podV6Cidr, er = egcitools.GetClusterCidr(f.KClient)
		if er != nil {
			return nil, er
		}
		mode = K8s
	case "":
		podV4Cidr = []string{}
		podV6Cidr = []string{}
		mode = ""
	default:
		return nil, err.ErrInvalidPodCidrMode
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	eci := new(egressv1beta1.EgressClusterInfo)
	for {
		select {
		case <-ctx.Done():
			return nil, err.TIME_OUT
		default:
			er := GetEgressClusterInfo(f, eci)
			if er != nil {
				return nil, er
			}

			eciPodCidrV4, eciPodCidrV6, podCidrMode := getEgressClusterInfoPodCidrAndMode(f)

			if len(podV4Cidr) == 0 && len(eciPodCidrV4) == 0 {
				v4ok = true
			} else {
				GinkgoWriter.Printf("podV4Cidr: %v, eciPodCidrV4: %v\n", podV4Cidr, eciPodCidrV4)
				v4ok, er = ip.IsSameIPCidrs(podV4Cidr, eciPodCidrV4)
				Expect(er).NotTo(HaveOccurred())
			}
			if len(podV6Cidr) == 0 && len(eciPodCidrV6) == 0 {
				v6ok = true
			} else {
				GinkgoWriter.Printf("podV6Cidr: %v, eciPodCidrV6: %v\n", podV6Cidr, eciPodCidrV6)
				v6ok, er = ip.IsSameIPCidrs(podV6Cidr, eciPodCidrV6)
				Expect(er).NotTo(HaveOccurred())
			}

			if v4ok && v6ok && podCidrMode == mode {
				return eci, nil
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func WaitEgressClusterInfoNodeIPUpdated(f *framework.Framework, timeout time.Duration) (*egressv1beta1.EgressClusterInfo, error) {
	nodesIPv4, nodesIPv6 := GetAllNodesIP(f)
	var v4ok, v6ok bool

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	eci := new(egressv1beta1.EgressClusterInfo)
	for {
		select {
		case <-ctx.Done():
			return nil, err.TIME_OUT
		default:
			er := GetEgressClusterInfo(f, eci)
			Expect(er).NotTo(HaveOccurred())

			eciNodesIPv4, eciNodesIPv6 := getEgressClusterInfoNodeIps(f)

			if len(nodesIPv4) == 0 && len(eciNodesIPv4) == 0 {
				v4ok = true
			} else {
				GinkgoWriter.Printf("nodesIPv4: %v, eciNodesIPv4: %v\n", nodesIPv4, eciNodesIPv4)
				v4ok, er = ip.IsSameIPs(nodesIPv4, eciNodesIPv4)
				Expect(er).NotTo(HaveOccurred())
			}
			if len(nodesIPv6) == 0 && len(eciNodesIPv6) == 0 {
				v6ok = true
			} else {
				GinkgoWriter.Printf("nodesIPv6: %v, eciNodesIPv6: %v\n", nodesIPv6, eciNodesIPv6)
				v6ok, er = ip.IsSameIPs(nodesIPv6, eciNodesIPv6)
				Expect(er).NotTo(HaveOccurred())
			}

			if v4ok && v6ok {
				return eci, nil
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func WaitEgressClusterInfoClusterIPUpdated(f *framework.Framework, timeout time.Duration) (*egressv1beta1.EgressClusterInfo, error) {
	clusterIPv4, clusterIPv6 := GetClusterIpCidr(f)
	var v4ok, v6ok bool

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	eci := new(egressv1beta1.EgressClusterInfo)
	for {
		select {
		case <-ctx.Done():
			return nil, err.TIME_OUT
		default:
			er := GetEgressClusterInfo(f, eci)
			Expect(er).NotTo(HaveOccurred())

			eciClusterIPv4, eciClusterIPv6 := getEgressClusterInfoClusterIps(f)

			if len(clusterIPv4) == 0 && len(eciClusterIPv4) == 0 {
				v4ok = true
			} else {
				GinkgoWriter.Printf("clusterIPv4: %v, eciClusterIPv4: %v\n", clusterIPv4, eciClusterIPv4)
				v4ok, er = ip.IsSameIPCidrs(clusterIPv4, eciClusterIPv4)
				Expect(er).NotTo(HaveOccurred())
			}
			if len(clusterIPv6) == 0 && len(eciClusterIPv6) == 0 {
				v6ok = true
			} else {
				GinkgoWriter.Printf("clusterIPv6: %v, eciClusterIPv6: %v\n", clusterIPv6, eciClusterIPv6)
				v6ok, er = ip.IsSameIPCidrs(clusterIPv6, eciClusterIPv6)
				Expect(er).NotTo(HaveOccurred())
			}

			if v4ok && v6ok {
				return eci, nil
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func WaitEgressClusterInfoExtraCidrUpdated(f *framework.Framework, timeout time.Duration) (*egressv1beta1.EgressClusterInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	eci := new(egressv1beta1.EgressClusterInfo)
	for {
		select {
		case <-ctx.Done():
			return nil, err.TIME_OUT
		default:
			if tools.IsSameSlice(eci.Spec.ExtraCidr, eci.Status.ExtraCidr) {
				return eci, nil
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func CheckEgressClusterInfoStatus(f *framework.Framework, timeout time.Duration) {
	GinkgoWriter.Println("check EgressClusterInfo status")

	egci := new(egressv1beta1.EgressClusterInfo)
	Expect(GetEgressClusterInfo(f, egci)).NotTo(HaveOccurred())

	if egci.Spec.AutoDetect.NodeIP {
		// check EgressClusterInfoNodeIP
		_, er := WaitEgressClusterInfoNodeIPUpdated(f, timeout)
		Expect(er).NotTo(HaveOccurred(), "failed check EgressClusterInfoNodeIP")
	}

	if egci.Spec.AutoDetect.ClusterIP {
		// check EgressClusterInfoClusterIP
		_, er := WaitEgressClusterInfoClusterIPUpdated(f, timeout)
		Expect(er).NotTo(HaveOccurred(), "failed check EgressClusterInfoClusterIP")
	}

	switch egci.Spec.AutoDetect.PodCidrMode {
	case Calico:
		// check EgressClusterInfoPodCidr
		_, er = WaitEgressClusterInfoPodCidrAndModeUpdated(f, Calico, timeout)
		Expect(er).NotTo(HaveOccurred(), "failed check EgressClusterInfoCalicoPodCidr")
	case K8s:
		// check EgressClusterInfoPodCidr
		_, er = WaitEgressClusterInfoPodCidrAndModeUpdated(f, K8s, timeout)
		Expect(er).NotTo(HaveOccurred(), "failed check EgressClusterInfoCalicoPodCidr")
	}

	_, er = WaitEgressClusterInfoExtraCidrUpdated(f, timeout)
	Expect(er).NotTo(HaveOccurred())
}

// getEgressClusterInfoPodCidr get egressClusterInfo podCidr slices
func getEgressClusterInfoPodCidrAndMode(f *framework.Framework) (v4PodCidr, v6PodCidr []string, mode egressv1beta1.PodCidrMode) {
	GinkgoWriter.Println("getEgressClusterInfoPodCidr")

	egci := new(egressv1beta1.EgressClusterInfo)
	Expect(GetEgressClusterInfo(f, egci)).NotTo(HaveOccurred())

	v4PodCidr = make([]string, 0)
	v6PodCidr = make([]string, 0)
	mode = egci.Status.PodCidrMode

	if egci.Status.PodCIDR == nil {
		return
	}

	for _, pair := range egci.Status.PodCIDR {
		v4PodCidr = append(v4PodCidr, pair.IPv4...)
		v6PodCidr = append(v6PodCidr, pair.IPv6...)
	}
	return
}

// getEgressClusterInfoNodeIps get egressClusterInfo nodeIps
func getEgressClusterInfoNodeIps(f *framework.Framework) (v4NodeIps, v6NodeIps []string) {
	GinkgoWriter.Println("getEgressClusterInfoNodeIps")

	egci := new(egressv1beta1.EgressClusterInfo)
	Expect(GetEgressClusterInfo(f, egci)).NotTo(HaveOccurred())

	v4NodeIps = make([]string, 0)
	v6NodeIps = make([]string, 0)

	if egci.Status.NodeIP == nil {
		return
	}

	for _, pair := range egci.Status.NodeIP {
		if len(pair.IPv4) != 0 {
			v4NodeIps = append(v4NodeIps, pair.IPv4...)
		}
		if len(pair.IPv6) != 0 {
			v6NodeIps = append(v6NodeIps, pair.IPv6...)
		}
	}
	return
}

// getEgressClusterInfoClusterIps get egressClusterInfo clusterCidr slices
func getEgressClusterInfoClusterIps(f *framework.Framework) (v4ClusterIps, v6ClusterIps []string) {
	GinkgoWriter.Println("getEgressClusterInfoClusterIps")

	egci := new(egressv1beta1.EgressClusterInfo)
	Expect(GetEgressClusterInfo(f, egci)).NotTo(HaveOccurred())

	v4ClusterIps = make([]string, 0)
	v6ClusterIps = make([]string, 0)

	if egci.Status.ClusterIP == nil {
		return
	}

	v4ClusterIps = append(v4ClusterIps, egci.Status.ClusterIP.IPv4...)
	v6ClusterIps = append(v6ClusterIps, egci.Status.ClusterIP.IPv6...)
	return
}
