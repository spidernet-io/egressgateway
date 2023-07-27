// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

//const egciName = "default"

//func GetEgressClusterInfo(f *framework.Framework, name string, egressClusterInfo *egressv1beta1.EgressClusterInfo) error {
//	key := client.ObjectKey{
//		Name: name,
//	}
//	return f.GetResource(key, egressClusterInfo)
//}
//
//func WaitEgressClusterInfoPodCidrUpdated(f *framework.Framework, podType string, timeout time.Duration) (*egressv1beta1.EgressClusterInfo, error) {
//	var podV4Cidr, podV6Cidr []string
//	var v4ok, v6ok bool
//	switch podType {
//	case CALICO:
//		podV4Cidr, podV6Cidr = GetCalicoIPPoolsCidr(f)
//	}
//
//	ctx, cancel := context.WithTimeout(context.Background(), timeout)
//	defer cancel()
//
//	eci := new(egressv1beta1.EgressClusterInfo)
//	for {
//		select {
//		case <-ctx.Done():
//			return nil, e.TIME_OUT
//		default:
//			err := GetEgressClusterInfo(f, egciName, eci)
//			if err != nil {
//				return nil, err
//			}
//
//			eciPodCidrV4 := eci.Status.EgressIgnoreCIDR.PodCIDR.IPv4
//			eciPodCidrV6 := eci.Status.EgressIgnoreCIDR.PodCIDR.IPv6
//
//			if len(podV4Cidr) == 0 && eciPodCidrV4 == nil {
//				v4ok = true
//			} else {
//				GinkgoWriter.Printf("podV4Cidr: %v, eciPodCidrV4: %v\n", podV4Cidr, eciPodCidrV4)
//				v4ok, err = utils.IsSameIPCidrs(podV4Cidr, eciPodCidrV4)
//				Expect(err).NotTo(HaveOccurred())
//			}
//			if len(podV6Cidr) == 0 && eciPodCidrV6 == nil {
//				v6ok = true
//			} else {
//				GinkgoWriter.Printf("podV6Cidr: %v, eciPodCidrV6: %v\n", podV6Cidr, eciPodCidrV6)
//				v6ok, err = utils.IsSameIPCidrs(podV6Cidr, eciPodCidrV6)
//				Expect(err).NotTo(HaveOccurred())
//			}
//
//			if v4ok && v6ok {
//				return eci, nil
//			}
//			time.Sleep(time.Millisecond * 100)
//		}
//	}
//}
//
//func WaitEgressClusterInfoNodeIPUpdated(f *framework.Framework, timeout time.Duration) (*egressv1beta1.EgressClusterInfo, error) {
//	nodesIPv4, nodesIPv6 := GetAllNodesIP(f)
//	var v4ok, v6ok bool
//
//	ctx, cancel := context.WithTimeout(context.Background(), timeout)
//	defer cancel()
//
//	eci := new(egressv1beta1.EgressClusterInfo)
//	for {
//		select {
//		case <-ctx.Done():
//			return nil, e.TIME_OUT
//		default:
//			err := GetEgressClusterInfo(f, egciName, eci)
//			Expect(err).NotTo(HaveOccurred())
//
//			eciNodesIPv4 := eci.Status.EgressIgnoreCIDR.NodeIP.IPv4
//			eciNodesIPv6 := eci.Status.EgressIgnoreCIDR.NodeIP.IPv6
//
//			if len(nodesIPv4) == 0 && eciNodesIPv4 == nil {
//				v4ok = true
//			} else {
//				GinkgoWriter.Printf("nodesIPv4: %v, eciNodesIPv4: %v\n", nodesIPv4, eciNodesIPv4)
//				v4ok, err = utils.IsSameIPs(nodesIPv4, eciNodesIPv4)
//				Expect(err).NotTo(HaveOccurred())
//			}
//			if len(nodesIPv6) == 0 && eciNodesIPv6 == nil {
//				v6ok = true
//			} else {
//				GinkgoWriter.Printf("nodesIPv6: %v, eciNodesIPv6: %v\n", nodesIPv6, eciNodesIPv6)
//				v6ok, err = utils.IsSameIPs(nodesIPv6, eciNodesIPv6)
//				Expect(err).NotTo(HaveOccurred())
//			}
//
//			if v4ok && v6ok {
//				return eci, nil
//			}
//			time.Sleep(time.Millisecond * 100)
//		}
//	}
//}
//
//func WaitEgressClusterInfoClusterIPUpdated(f *framework.Framework, timeout time.Duration) (*egressv1beta1.EgressClusterInfo, error) {
//	clusterIPv4, clusterIPv6 := GetClusterIpCidr(f)
//	var v4ok, v6ok bool
//
//	ctx, cancel := context.WithTimeout(context.Background(), timeout)
//	defer cancel()
//
//	eci := new(egressv1beta1.EgressClusterInfo)
//	for {
//		select {
//		case <-ctx.Done():
//			return nil, e.TIME_OUT
//		default:
//			err := GetEgressClusterInfo(f, egciName, eci)
//			Expect(err).NotTo(HaveOccurred())
//
//			eciClusterIPv4 := eci.Status.EgressIgnoreCIDR.ClusterIP.IPv4
//			eciClusterIPv6 := eci.Status.EgressIgnoreCIDR.ClusterIP.IPv6
//
//			if len(clusterIPv4) == 0 && eciClusterIPv4 == nil {
//				v4ok = true
//			} else {
//				GinkgoWriter.Printf("clusterIPv4: %v, eciClusterIPv4: %v\n", clusterIPv4, eciClusterIPv4)
//				v4ok, err = utils.IsSameIPCidrs(clusterIPv4, eciClusterIPv4)
//				Expect(err).NotTo(HaveOccurred())
//			}
//			if len(clusterIPv6) == 0 && eciClusterIPv6 == nil {
//				v6ok = true
//			} else {
//				GinkgoWriter.Printf("clusterIPv6: %v, eciClusterIPv6: %v\n", clusterIPv6, eciClusterIPv6)
//				v6ok, err = utils.IsSameIPCidrs(clusterIPv6, eciClusterIPv6)
//				Expect(err).NotTo(HaveOccurred())
//			}
//
//			if v4ok && v6ok {
//				return eci, nil
//			}
//			time.Sleep(time.Millisecond * 100)
//		}
//	}
//}
//
//func CheckEgressIgnoreCIDRFields(f *framework.Framework, timeout time.Duration) {
//	GinkgoWriter.Println("check EgressIgnoreCIDR Fields")
//	ignore, err := GetEgressIgnoreCIDR(f)
//	Expect(err).NotTo(HaveOccurred(), "failed to GetEgressIgnoreCIDR")
//	Expect(ignore).NotTo(BeNil(), "the config.EgressIgnoreCIDR is nil")
//
//	if ignore.NodeIP {
//		// check EgressClusterInfoNodeIP
//		_, err := WaitEgressClusterInfoNodeIPUpdated(f, timeout)
//		Expect(err).NotTo(HaveOccurred(), "failed check EgressClusterInfoNodeIP")
//	}
//
//	if ignore.ClusterIP {
//		// check EgressClusterInfoClusterIP
//		_, err := WaitEgressClusterInfoClusterIPUpdated(f, timeout)
//		Expect(err).NotTo(HaveOccurred(), "failed check EgressClusterInfoClusterIP")
//	}
//
//	switch ignore.PodCIDR {
//	case CALICO:
//		// check EgressClusterInfoPodCidr
//		_, err := WaitEgressClusterInfoPodCidrUpdated(f, CALICO, timeout)
//		Expect(err).NotTo(HaveOccurred(), "failed check EgressClusterInfoPodCidr")
//	}
//}
