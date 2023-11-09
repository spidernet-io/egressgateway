// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/constant"
	egress "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
)

type egnReconciler struct {
	client client.Client
	log    logr.Logger
	config *config.Config
}

type policyInfo struct {
	egw             string
	ipv4            string
	ipv6            string
	node            string
	policy          egress.Policy
	isUseNodeIP     bool
	allocatorPolicy string
}

func (r egnReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		return reconcile.Result{}, err
	}

	log := r.log.WithValues("kind", kind)

	switch kind {
	case "EgressGateway":
		return r.reconcileEGW(ctx, newReq, log)
	case "EgressClusterPolicy":
		fallthrough
	case "EgressPolicy":
		return r.reconcileEGP(ctx, newReq, log)
	case "Node":
		return r.reconcileNode(ctx, newReq, log)
	case "EgressTunnel":
		return r.reconcileEGT(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileNode reconcile node
func (r egnReconciler) reconcileNode(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	node := new(corev1.Node)
	err := r.client.Get(ctx, req.NamespacedName, node)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !node.GetDeletionTimestamp().IsZero()

	egwList := &egress.EgressGatewayList{}
	if err := r.client.List(ctx, egwList); err != nil {
		return reconcile.Result{Requeue: true}, nil
	}

	// Node NoReady event, complete in reconcile EgressTunnel event
	if deleted {
		r.log.Info("request item is deleted")
		err := r.deleteNodeFromEGs(ctx, log, req.Name, egwList)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	}

	// Checking the node label
	for _, egw := range egwList.Items {
		selNode, err := metav1.LabelSelectorAsSelector(egw.Spec.NodeSelector.Selector)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}
		isMatch := selNode.Matches(labels.Set(node.Labels))
		if isMatch {
			// If the tag matches, check whether information about the node exists. If it does not exist, add an empty one
			_, isExist := GetPoliciesByNode(node.Name, egw)
			if !isExist {
				egt := new(egress.EgressTunnel)
				err := r.client.Get(ctx, types.NamespacedName{Name: node.Name}, egt)
				if err == nil {
					egw.Status.NodeList = append(egw.Status.NodeList, egress.EgressIPStatus{Name: node.Name, Status: string(egt.Status.Phase)})
				} else {
					egw.Status.NodeList = append(egw.Status.NodeList, egress.EgressIPStatus{Name: node.Name, Status: string(egress.EgressTunnelFailed)})
				}

				r.log.V(1).Info("update egress gateway status", "status", egw.Status)
				err = r.client.Status().Update(ctx, &egw)
				if err != nil {
					r.log.Error(err, "update egress gateway status", "status", egw.Status)
					return reconcile.Result{Requeue: true}, nil
				}
			}
		} else {
			// Labels do not match. If there is a node in status, delete the node from status and reallocate the policy
			_, isExist := GetPoliciesByNode(node.Name, egw)
			if isExist {
				err := r.deleteNodeFromEG(ctx, log, node.Name, egw)
				if err != nil {
					return reconcile.Result{Requeue: true}, nil
				}
			}
		}
	}

	return reconcile.Result{}, nil
}

// reconcileEGW reconcile egress gateway
func (r egnReconciler) reconcileEGW(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	isUpdate := false
	egw := &egress.EgressGateway{}
	err := r.client.Get(ctx, req.NamespacedName, egw)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !egw.GetDeletionTimestamp().IsZero()

	if deleted {
		log.Info("request item is deleted")
		return reconcile.Result{}, nil
	}

	if egw.Spec.NodeSelector.Selector == nil {
		log.Info("nodeSelector is nil, skip reconcile")
		return reconcile.Result{}, nil
	}

	// Obtain the latest node that meets the conditions
	newNodeList := &corev1.NodeList{}
	selNodes, err := metav1.LabelSelectorAsSelector(egw.Spec.NodeSelector.Selector)
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.client.List(ctx, newNodeList, &client.ListOptions{
		LabelSelector: selNodes,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	log.Info("obtained nodes",
		"numberOfNodes", len(newNodeList.Items),
		"selector", egw.Spec.NodeSelector.Selector.String())

	// Get the node you want to delete
	delNodeMap := make(map[string]egress.EgressIPStatus)
	for _, oldNode := range egw.Status.NodeList {
		delNodeMap[oldNode.Name] = oldNode
	}

	for _, newNode := range newNodeList.Items {
		delete(delNodeMap, newNode.Name)
	}

	perNodeMap := make(map[string]egress.EgressIPStatus)
	for _, node := range egw.Status.NodeList {
		_, ok := delNodeMap[node.Name]
		if !ok {
			perNodeMap[node.Name] = node
		}
	}

	for _, node := range newNodeList.Items {
		_, ok := perNodeMap[node.Name]
		if !ok {
			egt := new(egress.EgressTunnel)
			err := r.client.Get(ctx, types.NamespacedName{Name: node.Name}, egt)
			if err == nil {
				perNodeMap[node.Name] = egress.EgressIPStatus{Name: node.Name, Status: string(egt.Status.Phase)}
			} else {
				perNodeMap[node.Name] = egress.EgressIPStatus{Name: node.Name, Status: string(egress.EgressTunnelFailed)}
			}
			isUpdate = true
		}
	}

	if len(egw.Status.NodeList) != len(newNodeList.Items) {
		isUpdate = true
	}

	log.Info("deleted gateway nodes", "delNodeMap", delNodeMap)

	if len(delNodeMap) != 0 {
		// Select a gateway node for the policy again
		var reSetPolicies []egress.Policy
		for _, item := range delNodeMap {
			for _, eip := range item.Eips {
				reSetPolicies = append(reSetPolicies, eip.Policies...)
			}
		}

		for _, policy := range reSetPolicies {
			if err = r.reAllocatorPolicy(ctx, log, policy, egw, perNodeMap); err != nil {
				log.Error(err, "failed to reallocate a gateway node for EgressPolicy",
					"policy", policy,
					"egressGateway", egw.Name,
					"namespace", egw.Namespace)
				return reconcile.Result{Requeue: true}, err
			}
		}

		isUpdate = true
	}

	// When the first gateway node of an egw recovers, you need to rebind the policy that references the egw
	readyNum := 0
	policyNum := 0
	for _, node := range perNodeMap {
		if node.Status == string(egress.EgressTunnelReady) {
			readyNum++
			policyNum += len(node.Eips)
		}
	}
	if readyNum == 1 && policyNum == 0 {
		var policies []egress.Policy
		egpList := &egress.EgressPolicyList{}
		if err := r.client.List(ctx, egpList); err != nil {
			log.Error(err, "list EgressPolicy failed")
			return reconcile.Result{Requeue: true}, err
		}

		for _, egp := range egpList.Items {
			if egp.Spec.EgressGatewayName == egw.Name {
				policies = append(policies, egress.Policy{Name: egp.Name, Namespace: egp.Namespace})
			}
		}

		egcpList := &egress.EgressClusterPolicyList{}
		if err := r.client.List(ctx, egcpList); err != nil {
			log.Error(err, "list EgressClusterPolicy failed")
			return reconcile.Result{Requeue: true}, err
		}

		for _, egcp := range egcpList.Items {
			if egcp.Spec.EgressGatewayName == egw.Name {
				policies = append(policies, egress.Policy{Name: egcp.Name})
			}
		}

		for _, policy := range policies {
			err = r.reAllocatorPolicy(ctx, log, policy, egw, perNodeMap)
			if err != nil {
				log.Error(err, "failed to reassign a gateway node for EgressPolicy", "policy", policy)
				return reconcile.Result{Requeue: true}, err
			}
		}

		isUpdate = true
	}

	if isUpdate {
		var perNodeList []egress.EgressIPStatus
		for _, node := range perNodeMap {
			perNodeList = append(perNodeList, node)
		}
		egw.Status.NodeList = perNodeList

		log.V(1).Info("update egress gateway status", "status", egw.Status)
		err = r.client.Status().Update(ctx, egw)
		if err != nil {
			log.Error(err, "update egress gateway status", "status", egw.Status)
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{}, nil
}

// reconcileEG reconcile egress tunnel
func (r egnReconciler) reconcileEGT(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	egt := new(egress.EgressTunnel)
	egt.Name = req.Name
	err := r.client.Get(ctx, req.NamespacedName, egt)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
	}
	deleted = deleted || !egt.GetDeletionTimestamp().IsZero()

	// The node deletion event has already been handled, so there is no need to do that here
	if deleted {
		log.Info("request item is deleted")
		return reconcile.Result{}, nil
	}

	egwList := &egress.EgressGatewayList{}

	if err := r.client.List(context.Background(), egwList); err != nil {
		return reconcile.Result{Requeue: true}, nil
	}

	for _, item := range egwList.Items {
		policies, isExist := GetPoliciesByNode(egt.Name, item)
		if isExist {
			perNodeMap := make(map[string]egress.EgressIPStatus)
			egw := item.DeepCopy()

			// If the node is not in success state, the policy on the node is reassigned
			if egt.Status.Phase != egress.EgressTunnelReady {
				for _, node := range egw.Status.NodeList {
					if node.Name != egt.Name {
						perNodeMap[node.Name] = node
					} else {
						perNodeMap[node.Name] = egress.EgressIPStatus{Name: node.Name, Status: string(egt.Status.Phase)}
					}
				}

				for _, policy := range policies {
					err = r.reAllocatorPolicy(ctx, log, policy, egw, perNodeMap)
					if err != nil {
						log.Error(err, "failed to reassign a gateway node for EgressPolicy", "policy", policy)
						return reconcile.Result{Requeue: true}, err
					}
				}
			} else {
				for _, node := range egw.Status.NodeList {
					if node.Name == egt.Name {
						for _, node := range egw.Status.NodeList {
							perNodeMap[node.Name] = node
						}

						if node.Status != string(egress.EgressTunnelReady) {
							perNodeMap[node.Name] = egress.EgressIPStatus{Name: node.Name, Eips: node.Eips, Status: string(egress.EgressTunnelReady)}

							// When the first gateway node of an egw recovers, you need to rebind the policy that references the egw
							readyNum := 0
							policyNum := 0
							for _, node := range perNodeMap {
								if node.Status == string(egress.EgressTunnelReady) {
									readyNum++
									policyNum += len(node.Eips)
								}
							}
							if readyNum == 1 && policyNum == 0 {
								var policies []egress.Policy
								egpList := &egress.EgressPolicyList{}
								if err := r.client.List(ctx, egpList); err != nil {
									log.Error(err, "list EgressPolicy failed")
									return reconcile.Result{Requeue: true}, err
								}

								for _, egp := range egpList.Items {
									if egp.Spec.EgressGatewayName == egw.Name {
										policies = append(policies, egress.Policy{Name: egp.Name, Namespace: egp.Namespace})
									}
								}

								egcpList := &egress.EgressClusterPolicyList{}
								if err := r.client.List(ctx, egpList); err != nil {
									log.Error(err, "list EgressClusterPolicy failed")
									return reconcile.Result{Requeue: true}, err
								}

								for _, egcp := range egcpList.Items {
									if egcp.Spec.EgressGatewayName == egw.Name {
										policies = append(policies, egress.Policy{Name: egcp.Name})
									}
								}

								for _, policy := range policies {
									err = r.reAllocatorPolicy(ctx, log, policy, egw, perNodeMap)
									if err != nil {
										log.Error(err, "failed to reassign a gateway node for EgressPolicy", "policy", policy)
										return reconcile.Result{Requeue: true}, err
									}
								}
							}
							break
						}
						return reconcile.Result{Requeue: false}, nil
					}
				}
			}

			var perNodeList []egress.EgressIPStatus
			for _, node := range perNodeMap {
				perNodeList = append(perNodeList, node)
			}

			egw.Status.NodeList = perNodeList

			log.V(1).Info("update egress gateway status", "status", egw.Status)
			err = r.client.Status().Update(ctx, egw)
			if err != nil {
				log.Error(err, "update egress gateway status", "status", egw.Status)
				return reconcile.Result{Requeue: true}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

// reconcileEN reconcile EgressPolicy and EgressClusterPolicy
func (r egnReconciler) reconcileEGP(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	if req.Namespace == "" {
		log = log.WithValues("name", req.Name)
	} else {
		log = log.WithValues("name", req.Name, "namespace", req.Namespace)
	}
	log.V(1).Info("reconciling")

	deleted := false
	isUpdate := false
	egp := &egress.EgressPolicy{}
	egcp := &egress.EgressClusterPolicy{}
	pi := policyInfo{}

	if len(req.Namespace) == 0 {
		err := r.client.Get(ctx, req.NamespacedName, egcp)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err, "retrieves an obj from the k8s")
				return reconcile.Result{}, err
			}
			deleted = true
		}
		deleted = deleted || !egcp.GetDeletionTimestamp().IsZero()
		pi.policy = egress.Policy{Name: req.Name}
		if !deleted {
			if len(egcp.Spec.EgressIP.IPv4) != 0 {
				pi.ipv4 = egcp.Spec.EgressIP.IPv4
			} else {
				pi.ipv4 = egcp.Status.Eip.Ipv4
			}

			if len(egcp.Spec.EgressIP.IPv6) != 0 {
				pi.ipv6 = egcp.Spec.EgressIP.IPv6
			} else {
				pi.ipv6 = egcp.Status.Eip.Ipv6
			}

			pi.isUseNodeIP = egcp.Spec.EgressIP.UseNodeIP
			pi.egw = egcp.Spec.EgressGatewayName
		}
	} else {
		err := r.client.Get(ctx, req.NamespacedName, egp)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err, "retrieves an obj from the k8s")
				return reconcile.Result{}, err
			}
			deleted = true
		}
		deleted = deleted || !egp.GetDeletionTimestamp().IsZero()
		pi.policy = egress.Policy{Name: req.Name, Namespace: req.Namespace}
		if !deleted {
			if len(egp.Spec.EgressIP.IPv4) != 0 {
				pi.ipv4 = egp.Spec.EgressIP.IPv4
			} else {
				pi.ipv4 = egp.Status.Eip.Ipv4
			}

			if len(egp.Spec.EgressIP.IPv6) != 0 {
				pi.ipv6 = egp.Spec.EgressIP.IPv6
			} else {
				pi.ipv6 = egp.Status.Eip.Ipv6
			}

			pi.isUseNodeIP = egp.Spec.EgressIP.UseNodeIP
			pi.egw = egp.Spec.EgressGatewayName
		}
	}

	policy := pi.policy
	if deleted {
		egwList := &egress.EgressGatewayList{}
		if err := r.client.List(ctx, egwList); err != nil {
			return reconcile.Result{Requeue: true}, nil
		}
		for _, egw := range egwList.Items {
			_, isExist := GetEIPStatusByPolicy(policy, egw)
			if isExist {
				log.Info("delete policy", "policy", policy, "egw", egw.Name)
				// Delete the policy from the EgressGateway. If the referenced EIP is not used by any other policy,
				// the system reclaims the EIP.
				DeletePolicyFromEG(log, policy, &egw)

				log.V(1).Info("update egress gateway status", "status", egw.Status)
				err := r.client.Status().Update(ctx, &egw)
				if err != nil {
					log.Error(err, "update egress gateway status", "status", egw.Status)
					return reconcile.Result{Requeue: true}, err
				}
				return reconcile.Result{}, nil
			}
		}
		return reconcile.Result{}, nil
	}

	egwName := pi.egw
	egw := &egress.EgressGateway{}
	err := r.client.Get(ctx, types.NamespacedName{Name: egwName}, egw)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		log.Error(err, "get EgressGateway")
		return reconcile.Result{Requeue: true}, err
	}

	// Assigned if the policy does not have a gateway node
	eipStatus, isExist := GetEIPStatusByPolicy(policy, *egw)
	if !isExist {
		perNodeMap := make(map[string]egress.EgressIPStatus)
		for _, item := range egw.Status.NodeList {
			perNodeMap[item.Name] = item
		}

		err := r.reAllocatorPolicy(ctx, log, policy, egw, perNodeMap)
		if err != nil {
			r.log.Error(err, "reallocator Failed to reassign a gateway node for EgressPolicy", "policy", policy)
			return reconcile.Result{Requeue: true}, err
		}

		var perNodeList []egress.EgressIPStatus
		for _, node := range perNodeMap {
			perNodeList = append(perNodeList, node)
		}
		egw.Status.NodeList = perNodeList

		isUpdate = true
	} else {
		// Check whether the EIP is correct
		for i, eip := range eipStatus.Eips {
			for j, p := range eip.Policies {
				if p == policy {
					isReAllocatorPolicy := false
					if pi.isUseNodeIP && (eip.IPv4 != "" || eip.IPv6 != "") {
						isReAllocatorPolicy = true
					} else if pi.ipv4 != "" && pi.ipv4 != eip.IPv4 {
						log.Info("policy", policy, ", pi.ipv4=", pi.ipv4, ", eip.IPv4", "=", eip.IPv4)
						isReAllocatorPolicy = true
					} else if pi.ipv6 != "" && pi.ipv6 != eip.IPv6 {
						log.Info("policy", policy, ", pi.ipv6=", pi.ipv6, ", eip.IPv6", "=", eip.IPv6)
						isReAllocatorPolicy = true
					}

					if isReAllocatorPolicy {
						eipStatus.Eips[i].Policies = append(eipStatus.Eips[i].Policies[:j], eipStatus.Eips[i].Policies[j+1:]...)
						perNodeMap := make(map[string]egress.EgressIPStatus)
						for _, node := range egw.Status.NodeList {
							if node.Name == eipStatus.Name {
								perNodeMap[node.Name] = eipStatus
							} else {
								perNodeMap[node.Name] = node
							}
						}
						err := r.reAllocatorPolicy(ctx, log, policy, egw, perNodeMap)
						if err != nil {
							log.Error(err, "failed to reassign a gateway node for EgressPolicy",
								"policy", policy,
								"egressGateway", egw.Name,
								"namespace", egw.Namespace)

							return reconcile.Result{Requeue: true}, err
						}

						var perNodeList []egress.EgressIPStatus
						for _, node := range perNodeMap {
							perNodeList = append(perNodeList, node)
						}
						egw.Status.NodeList = perNodeList
					} else {
						// check policy status
						var policyStatus egress.EgressPolicyStatus
						policyStatus.Eip.Ipv4 = eip.IPv4
						policyStatus.Eip.Ipv6 = eip.IPv6
						policyStatus.Node = eipStatus.Name

						if len(policy.Namespace) == 0 {
							if len(egcp.Status.Node) == 0 {
								egcp.Status = policyStatus
								log.V(1).Info("update egressclusterpolicy status", "status", egcp.Status)
								err = r.client.Status().Update(ctx, egcp)
								if err != nil {
									log.Error(err, "update egressclusterpolicy status", "status", egcp.Status)
									return reconcile.Result{Requeue: true}, err
								}
							}
						} else {
							if len(egp.Status.Node) == 0 {
								egp.Status = policyStatus
								log.V(1).Info("update egresspolicy status", "status", egp.Status)
								err = r.client.Status().Update(ctx, egp)
								if err != nil {
									log.Error(err, "update egresspolicy status", "status", egp.Status)
									return reconcile.Result{Requeue: true}, err
								}
							}
						}
					}

					isUpdate = true
					goto update
				}
			}
		}

	}

update:
	if isUpdate {
		r.log.V(1).Info("update egress gateway status", "status", egw.Status)
		err = r.client.Status().Update(ctx, egw)
		if err != nil {
			r.log.Error(err, "update egress gateway status", "status", egw.Status)
			return reconcile.Result{Requeue: true}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r egnReconciler) deleteNodeFromEGs(ctx context.Context, log logr.Logger, nodeName string, egwList *egress.EgressGatewayList) error {
	for _, egw := range egwList.Items {
		for _, eipStatus := range egw.Status.NodeList {
			if nodeName == eipStatus.Name {
				err := r.deleteNodeFromEG(ctx, log, nodeName, egw)
				if err != nil {
					return err
				}
				break
			}
		}
	}

	return nil
}

// Delete the node from the EgressGateway
func (r egnReconciler) deleteNodeFromEG(ctx context.Context, log logr.Logger, nodeName string, egw egress.EgressGateway) error {
	// Get the policy that needs to be reassigned
	policies, isExist := GetPoliciesByNode(nodeName, egw)

	if isExist {
		perNodeMap := make(map[string]egress.EgressIPStatus)
		for _, item := range egw.Status.NodeList {
			if nodeName != item.Name {
				perNodeMap[item.Name] = item
			}
		}

		// Redistribute network gateway nodes
		for _, policy := range policies {
			err := r.reAllocatorPolicy(ctx, log, policy, &egw, perNodeMap)
			if err != nil {
				r.log.Error(err, "failed to reassign a gateway node for EgressPolicy", "policy", policy)
				return err
			}
		}

		var perNodeList []egress.EgressIPStatus
		for _, node := range perNodeMap {
			perNodeList = append(perNodeList, node)
		}

		egw.Status.NodeList = perNodeList
		r.log.V(1).Info("update egress gateway status", "status", egw.Status)
		err := r.client.Status().Update(ctx, &egw)
		if err != nil {
			r.log.Error(err, "update egress gateway status", "status", egw.Status)
			return err
		}
	}

	return nil
}

func (r egnReconciler) reAllocatorPolicy(ctx context.Context, log logr.Logger, policy egress.Policy, egw *egress.EgressGateway, nodeMap map[string]egress.EgressIPStatus) error {
	var perNode string
	var ipv4, ipv6 string
	var err error
	pi := policyInfo{}
	pi.policy = policy

	if len(nodeMap) == 0 {
		r.log.Info("egw: ", egw.Name, " does not have a matching node")
		return nil
	}

	if len(pi.policy.Namespace) == 0 {
		egcp := &egress.EgressClusterPolicy{}
		err := r.client.Get(ctx, types.NamespacedName{Name: pi.policy.Name}, egcp)
		if err != nil {
			return err
		}

		if len(egcp.Spec.EgressIP.IPv4) != 0 {
			pi.ipv4 = egcp.Spec.EgressIP.IPv4
		} else {
			pi.ipv4 = egcp.Status.Eip.Ipv4
		}

		if len(egcp.Spec.EgressIP.IPv6) != 0 {
			pi.ipv6 = egcp.Spec.EgressIP.IPv6
		} else {
			pi.ipv6 = egcp.Status.Eip.Ipv6
		}

		pi.isUseNodeIP = egcp.Spec.EgressIP.UseNodeIP
		pi.egw = egcp.Spec.EgressGatewayName
		pi.allocatorPolicy = egcp.Spec.EgressIP.AllocatorPolicy
	} else {
		egp := &egress.EgressPolicy{}
		err := r.client.Get(ctx, types.NamespacedName{Namespace: pi.policy.Namespace, Name: pi.policy.Name}, egp)
		if err != nil {
			return err
		}

		if len(egp.Spec.EgressIP.IPv4) != 0 {
			pi.ipv4 = egp.Spec.EgressIP.IPv4
		} else {
			pi.ipv4 = egp.Status.Eip.Ipv4
		}

		if len(egp.Spec.EgressIP.IPv6) != 0 {
			pi.ipv6 = egp.Spec.EgressIP.IPv6
		} else {
			pi.ipv6 = egp.Status.Eip.Ipv6
		}

		pi.isUseNodeIP = egp.Spec.EgressIP.UseNodeIP
		pi.egw = egp.Spec.EgressGatewayName
		pi.allocatorPolicy = egp.Spec.EgressIP.AllocatorPolicy
	}

	ipv4 = pi.ipv4
	if len(ipv4) != 0 {
		perNode = GetNodeByIP(ipv4, *egw)
		if nodeMap[perNode].Status != string(egress.EgressTunnelReady) {
			perNode = ""
		}

		if len(perNode) == 0 {
			perNode, err = r.allocatorNode("rr", nodeMap)
			if err != nil {
				return err
			}
		}

		ipv4, ipv6, err = r.allocatorEIP("", perNode, pi, *egw)
		if err != nil {
			return err
		}
	} else {
		allocatorPolicy := pi.allocatorPolicy
		if allocatorPolicy == egress.EipAllocatorRR {
			perNode, err = r.allocatorNode("rr", nodeMap)
			if err != nil {
				return err
			}

			ipv4, ipv6, err = r.allocatorEIP("", perNode, pi, *egw)
			if err != nil {
				return err
			}
		} else {
			ipv4 = egw.Spec.Ippools.Ipv4DefaultEIP
			ipv6 = egw.Spec.Ippools.Ipv6DefaultEIP

			perNode = GetNodeByIP(ipv4, *egw)
			if nodeMap[perNode].Status != string(egress.EgressTunnelReady) {
				perNode = ""
			}

			if len(perNode) == 0 {
				perNode, err = r.allocatorNode("rr", nodeMap)
				if err != nil {
					return err
				}
			}
		}
	}

	log.Info("reAllocatorPolicy", " policy=", pi.policy, " perNode=", perNode, " ipv4=", ipv4, " ipv6=", ipv6)

	err = setEipStatus(ipv4, ipv6, perNode, pi.policy, nodeMap)
	if err != nil {
		return err
	}

	return nil
}

func (r egnReconciler) allocatorNode(selNodePolicy string, nodeMap map[string]egress.EgressIPStatus) (string, error) {

	if len(nodeMap) == 0 {
		err := fmt.Errorf("nodeList is empty")
		return "", err
	}

	var perNode string
	perNodePolicyNum := 0
	i := 0
	for _, node := range nodeMap {
		if node.Status != string(egress.EgressTunnelReady) {
			continue
		}

		policyNum := 0
		for _, eip := range node.Eips {
			policyNum += len(eip.Policies)
		}

		if i == 0 {
			i++
			perNode = node.Name
			perNodePolicyNum = policyNum
		} else if policyNum <= perNodePolicyNum {
			perNode = node.Name
			perNodePolicyNum = policyNum
		}
	}

	return perNode, nil
}

func (r egnReconciler) allocatorEIP(selEipLolicy string, nodeName string, pi policyInfo, egw egress.EgressGateway) (string, string, error) {

	if pi.isUseNodeIP || len(nodeName) == 0 {
		return "", "", nil
	}
	var perIpv4 string
	var perIpv6 string
	rander := rand.New(rand.NewSource(time.Now().UnixNano()))

	if r.config.FileConfig.EnableIPv4 {
		var useIpv4s []net.IP

		ipv4Ranges, _ := ip.MergeIPRanges(constant.IPv4, egw.Spec.Ippools.IPv4)
		perIpv4 = pi.ipv4
		if len(perIpv4) != 0 {
			result, err := ip.IsIPIncludedRange(constant.IPv4, perIpv4, ipv4Ranges)
			if err != nil {
				return "", "", err
			}
			if !result {
				return "", "", fmt.Errorf("%v is not within the EIP range of EgressGateway %v", perIpv4, egw.Name)
			}
		} else {
			for _, node := range egw.Status.NodeList {
				for _, eip := range node.Eips {
					if len(eip.IPv4) != 0 {
						useIpv4s = append(useIpv4s, net.ParseIP(eip.IPv4))
					}
				}
			}

			ipv4s, _ := ip.ParseIPRanges(constant.IPv4, ipv4Ranges)
			freeIpv4s := ip.IPsDiffSet(ipv4s, useIpv4s, false)

			if len(freeIpv4s) == 0 {
				return "", "", fmt.Errorf("No Egress IPV4 is available; policy=%v egw=%v", pi.policy, egw.Name)

				// save it for later policy
				// var useIpv4sByNode []net.IP
				// for _, node := range egw.Status.NodeList {
				// 	if node.Name == nodeName {
				// 		for _, eip := range node.Eips {
				// 			if len(eip.IPv4) != 0 {
				// 				useIpv4sByNode = append(useIpv4sByNode, net.ParseIP(eip.IPv4))
				// 			}
				// 		}
				// 	}
				// }

				// if len(useIpv4sByNode) == 0 {
				// 	return "", "", fmt.Errorf("No EIP meeting requirements is found on node %v; EG %v", nodeName, egw.Name)
				// }

				// perIpv4 = useIpv4sByNode[rander.Intn(len(useIpv4sByNode))].String()
			} else {
				perIpv4 = freeIpv4s[rander.Intn(len(freeIpv4s))].String()
			}
		}
	}

	if r.config.FileConfig.EnableIPv6 {
		if len(perIpv4) != 0 && len(GetEipByIPV4(perIpv4, egw).IPv6) != 0 {
			return perIpv4, GetEipByIPV4(perIpv4, egw).IPv6, nil
		}

		var useIpv6s []net.IP

		ipv6Ranges, _ := ip.MergeIPRanges(constant.IPv6, egw.Spec.Ippools.IPv6)

		perIpv6 = pi.ipv6
		if len(perIpv6) != 0 {
			result, err := ip.IsIPIncludedRange(constant.IPv6, perIpv6, ipv6Ranges)
			if err != nil {
				return "", "", err
			}
			if !result {
				return "", "", fmt.Errorf("%v is not within the EIP range of EgressGateway %v", perIpv6, egw.Name)
			}
		} else {
			for _, node := range egw.Status.NodeList {
				for _, eip := range node.Eips {
					if len(eip.IPv6) != 0 {
						useIpv6s = append(useIpv6s, net.ParseIP(eip.IPv6))
					}
				}
			}

			ipv6s, _ := ip.ParseIPRanges(constant.IPv6, ipv6Ranges)
			freeIpv6s := ip.IPsDiffSet(ipv6s, useIpv6s, false)

			if len(freeIpv6s) == 0 {
				return "", "", fmt.Errorf("No Egress IPV4 is available; policy=%v egw=%v", pi.policy, egw.Name)

				// save it for later policy
				// var useIpv6sByNode []net.IP
				// for _, node := range egw.Status.NodeList {
				// 	if node.Name == nodeName {
				// 		for _, eip := range node.Eips {
				// 			if len(eip.IPv6) != 0 {
				// 				useIpv6sByNode = append(useIpv6sByNode, net.ParseIP(eip.IPv6))
				// 			}
				// 		}
				// 	}
				// }

				// if len(useIpv6sByNode) == 0 {
				// 	return "", "", fmt.Errorf("No EIP meeting requirements is found on node %v; EG %v", nodeName, egw.Name)
				// }
				// perIpv6 = useIpv6sByNode[rander.Intn(len(useIpv6sByNode))].String()
			} else {
				perIpv6 = freeIpv6s[rander.Intn(len(freeIpv6s))].String()
			}
		}
	}

	return perIpv4, perIpv6, nil
}

func NewEgressGatewayController(mgr manager.Manager, log logr.Logger, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("cfg can not be nil")
	}
	r := &egnReconciler{
		client: mgr.GetClient(),
		log:    log,
		config: cfg,
	}

	c, err := controller.New("egressGateway", mgr,
		controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err = c.Watch(source.Kind(mgr.GetCache(), &egress.EgressGateway{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGateway"))); err != nil {
		return fmt.Errorf("failed to watch EgressGateway: %w", err)
	}

	if err = c.Watch(source.Kind(mgr.GetCache(), &corev1.Node{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("Node"))); err != nil {
		return fmt.Errorf("failed to watch Node: %w", err)
	}

	if err = c.Watch(source.Kind(mgr.GetCache(), &egress.EgressPolicy{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressPolicy"))); err != nil {
		return fmt.Errorf("failed to watch EgressPolicy: %w", err)
	}

	if err = c.Watch(source.Kind(mgr.GetCache(), &egress.EgressClusterPolicy{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressClusterPolicy"))); err != nil {
		return fmt.Errorf("failed to watch EgressClusterPolicy: %w", err)
	}

	if err = c.Watch(source.Kind(mgr.GetCache(), &egress.EgressTunnel{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressTunnel"))); err != nil {
		return fmt.Errorf("failed to watch EgressTunnel: %w", err)
	}

	return nil
}

func GetEipByIPV4(ipv4 string, egw egress.EgressGateway) egress.Eips {
	var eipInfo egress.Eips
	for _, node := range egw.Status.NodeList {
		for _, eip := range node.Eips {
			if eip.IPv4 == ipv4 {
				eipInfo = eip
			}
		}
	}

	return eipInfo
}

func GetEipByIPV6(ipv6 string, egw egress.EgressGateway) egress.Eips {
	var eipInfo egress.Eips
	for _, node := range egw.Status.NodeList {
		for _, eip := range node.Eips {
			if eip.IPv6 == ipv6 {
				eipInfo = eip
			}
		}
	}

	return eipInfo
}

func GetNodeByIP(ipv4 string, egw egress.EgressGateway) string {
	var nodeName string
	for _, node := range egw.Status.NodeList {
		for _, eip := range node.Eips {
			if eip.IPv4 == ipv4 {
				nodeName = node.Name
			}
		}
	}

	return nodeName
}

func setEipStatus(ipv4, ipv6 string, nodeName string, policy egress.Policy, nodeMap map[string]egress.EgressIPStatus) error {
	if len(nodeName) == 0 {
		return nil
	}

	eipStatus, ok := nodeMap[nodeName]
	if !ok {
		return fmt.Errorf("the %v node is not a gateway node", nodeName)
	}
	isExist := false
	newEipStatus := egress.EgressIPStatus{}

	for _, eip := range eipStatus.Eips {
		if ipv4 == eip.IPv4 {
			eip.Policies = append(eip.Policies, policy)

			isExist = true
		}
		newEipStatus.Eips = append(newEipStatus.Eips, eip)
	}

	if !isExist {
		newEip := egress.Eips{}
		newEip.IPv4 = ipv4
		newEip.IPv6 = ipv6
		newEip.Policies = append(newEip.Policies, policy)
		eipStatus.Eips = append(eipStatus.Eips, newEip)
		nodeMap[nodeName] = eipStatus
	} else {
		newEipStatus.Name = nodeName
		newEipStatus.Status = eipStatus.Status
		nodeMap[nodeName] = newEipStatus
	}

	return nil
}

func GetPoliciesByNode(nodeName string, egw egress.EgressGateway) ([]egress.Policy, bool) {

	var eipStatus egress.EgressIPStatus
	var policies []egress.Policy
	isExist := false
	for _, node := range egw.Status.NodeList {
		if node.Name == nodeName {
			eipStatus = node
			isExist = true
		}
	}

	if isExist {
		for _, eip := range eipStatus.Eips {
			policies = append(policies, eip.Policies...)
		}
	}

	return policies, isExist
}

func GetEIPStatusByPolicy(policy egress.Policy, egw egress.EgressGateway) (egress.EgressIPStatus, bool) {
	var eipStatus egress.EgressIPStatus
	isExist := false

	for _, item := range egw.Status.NodeList {
		for _, eip := range item.Eips {
			for _, p := range eip.Policies {
				if p == policy {
					eipStatus = item
					isExist = true
				}
			}
		}
	}

	return eipStatus, isExist
}

func DeletePolicyFromEG(log logr.Logger, policy egress.Policy, egw *egress.EgressGateway) {
	var policies []egress.Policy
	var eips []egress.Eips
	for i, node := range egw.Status.NodeList {
		for j, eip := range node.Eips {
			for k, item := range eip.Policies {
				if item == policy {
					policies = append(eip.Policies[:k], eip.Policies[k+1:]...)

					if len(policies) == 0 {
						// Release EIP
						for x, e := range node.Eips {
							if (len(eip.IPv4) != 0 && eip.IPv4 == e.IPv4) || (len(eip.IPv6) != 0 && eip.IPv6 == e.IPv6) {
								eips = append(node.Eips[:x], node.Eips[x+1:]...)
								log.Info("release", " EIP= ", node.Eips[x], " policy=", policy)
								break
							}
						}
						egw.Status.NodeList[i].Eips = eips
					} else {
						egw.Status.NodeList[i].Eips[j].Policies = policies
					}
					goto breakHere
				}
			}
		}
	}
breakHere:
	return
}
