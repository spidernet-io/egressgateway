// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"

	"go.uber.org/zap"
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
	egress "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

const (
	indexEgressNodeEgressGateway = "egressNodeEgressGatewayIndex"
	indexNodeEgressGateway       = "nodeEgressGatewayIndex"
)

type egnReconciler struct {
	client client.Client
	log    *zap.Logger
	config *config.Config
}

func (r egnReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		r.log.Sugar().Infof("parse req(%v) with error: %v", req, err)
		return reconcile.Result{}, err
	}
	log := r.log.With(
		zap.String("namespacedName", newReq.NamespacedName.String()),
		zap.String("kind", kind),
	)
	log.Info("reconciling")
	switch kind {
	case "EgressGateway":
		return r.reconcileEG(ctx, newReq, log)
	case "EgressPolicy":
		return r.reconcileEGP(ctx, newReq, log)
	case "Node":
		return r.reconcileNode(ctx, newReq, log)
	case "EgressNode":
		return r.reconcileEN(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileNode reconcile node
func (r egnReconciler) reconcileNode(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
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

	egList := &egress.EgressGatewayList{}
	if err := r.client.List(context.Background(), egList); err != nil {
		return reconcile.Result{Requeue: true}, nil
	}

	// Node NoReady event, complete in reconcile EgressNode event
	if deleted {
		r.log.Info("request item is deleted")
		err := r.deleteNodeFromEGs(ctx, req.Name, egList)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	}

	// Checking the node label
	for _, eg := range egList.Items {
		selNode, err := metav1.LabelSelectorAsSelector(eg.Spec.NodeSelector.Selector)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}
		isMatch := selNode.Matches(labels.Set(node.Labels))
		if isMatch {
			// If the tag matches, check whether information about the node exists. If it does not exist, add an empty one
			_, isExist := GetEIPStatusByNode(node.Name, eg)
			if !isExist {
				eg.Status.NodeList = append(eg.Status.NodeList, egress.EgressIPStatus{Name: node.Name})

				r.log.Sugar().Debugf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
				err := r.client.Status().Update(ctx, &eg)
				if err != nil {
					r.log.Sugar().Errorf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
					return reconcile.Result{Requeue: true}, nil
				}
			}
		} else {
			// Labels do not match. If there is a node in status, delete the node from status and reallocate the policy
			_, isExist := GetEIPStatusByNode(node.Name, eg)
			if isExist {
				err := r.deleteNodeFromEG(ctx, node.Name, eg)
				if err != nil {
					return reconcile.Result{Requeue: true}, nil
				}
			}
		}
	}

	return reconcile.Result{}, nil
}

// reconcileEG reconcile egress gateway
func (r egnReconciler) reconcileEG(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	deleted := false
	isUpdete := false
	eg := &egress.EgressGateway{}
	err := r.client.Get(ctx, req.NamespacedName, eg)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !eg.GetDeletionTimestamp().IsZero()

	if deleted {
		log.Info("request item is deleted")
		return reconcile.Result{}, nil
	}

	if eg.Spec.NodeSelector.Selector == nil {
		log.Info("nodeSelector is nil, skip reconcile")
		return reconcile.Result{}, nil
	}

	// Obtain the latest node that meets the conditions
	newNodeList := &corev1.NodeList{}
	selNodes, err := metav1.LabelSelectorAsSelector(eg.Spec.NodeSelector.Selector)
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.client.List(ctx, newNodeList, &client.ListOptions{
		LabelSelector: selNodes,
	})
	if err != nil {
		return reconcile.Result{}, err
	}
	log.Sugar().Debugf("number of selected nodes: %d", len(newNodeList.Items))

	// Get the node you want to delete

	delNodeMap := make(map[string]egress.EgressIPStatus)
	for _, oldNode := range eg.Status.NodeList {
		delNodeMap[oldNode.Name] = oldNode
	}

	for _, newNode := range newNodeList.Items {
		delete(delNodeMap, newNode.Name)
	}

	perNodeMap := make(map[string]egress.EgressIPStatus, 0)
	for _, node := range eg.Status.NodeList {
		_, ok := delNodeMap[node.Name]
		if !ok {
			perNodeMap[node.Name] = node
		}
	}

	for _, node := range newNodeList.Items {
		_, ok := perNodeMap[node.Name]
		if !ok {
			perNodeMap[node.Name] = egress.EgressIPStatus{Name: node.Name}
		}
	}

	if len(eg.Status.NodeList) != len(newNodeList.Items) {
		isUpdete = true
	}

	log.Sugar().Infof("delete a gateway nodes: %d", delNodeMap)
	if len(delNodeMap) != 0 {
		// Select a gateway node for the policy again
		var reSetPolicies []egress.Policy
		for _, item := range delNodeMap {
			for _, eip := range item.Eips {
				reSetPolicies = append(reSetPolicies, eip.Policies...)
			}
		}

		for _, policy := range reSetPolicies {
			err = r.reAllocatorPolicy(ctx, policy, eg, perNodeMap)
			if err != nil {
				log.Sugar().Errorf("reallocator Failed to reassign a gateway node for EgressPolicy %v: %v", policy, err)
				return reconcile.Result{Requeue: true}, err
			}
		}

		isUpdete = true
	}

	if isUpdete {
		var perNodeList []egress.EgressIPStatus
		for _, node := range perNodeMap {
			perNodeList = append(perNodeList, node)
		}
		eg.Status.NodeList = perNodeList

		log.Sugar().Debugf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
		err = r.client.Status().Update(ctx, eg)
		if err != nil {
			log.Sugar().Errorf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{}, nil
}

// reconcileEG reconcile egress node
func (r egnReconciler) reconcileEN(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	deleted := false
	en := new(egress.EgressNode)
	en.Name = req.Name
	err := r.client.Get(ctx, req.NamespacedName, en)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
	}
	deleted = deleted || !en.GetDeletionTimestamp().IsZero()

	// The node deletion event has already been handled, so there is no need to do that here
	if deleted {
		log.Info("request item is deleted")
		return reconcile.Result{}, nil
	}

	// If the node is not in success state, the policy on the node is reassigned
	if en.Status.Phase != egress.EgressNodeSucceeded {
		egList := &egress.EgressGatewayList{}
		if err := r.client.List(context.Background(), egList); err != nil {
			return reconcile.Result{Requeue: true}, nil
		}
		for _, eg := range egList.Items {
			policies, isExist := GetEIPStatusByNode(en.Name, eg)
			if isExist {
				perNodeMap := make(map[string]egress.EgressIPStatus, 0)
				for _, node := range eg.Status.NodeList {
					if node.Name != en.Name {
						perNodeMap[node.Name] = node
					}
				}

				for _, policy := range policies {
					err = r.reAllocatorPolicy(ctx, policy, &eg, perNodeMap)
					if err != nil {
						log.Sugar().Errorf("reallocator Failed to reassign a gateway node for EgressPolicy %v: %v", policy, err)
						return reconcile.Result{Requeue: true}, err
					}
				}

				var perNodeList []egress.EgressIPStatus
				for _, node := range perNodeMap {
					perNodeList = append(perNodeList, node)
				}

				perNodeList = append(perNodeList, egress.EgressIPStatus{Name: en.Name})
				eg.Status.NodeList = perNodeList

				log.Sugar().Debugf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
				err = r.client.Status().Update(ctx, &eg)
				if err != nil {
					log.Sugar().Errorf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
					return reconcile.Result{Requeue: true}, err
				}
			}
		}

	}

	return reconcile.Result{}, nil
}

// reconcileEN reconcile egress gateway policy
func (r egnReconciler) reconcileEGP(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	deleted := false
	isUpdete := false
	egp := &egress.EgressPolicy{}
	err := r.client.Get(ctx, req.NamespacedName, egp)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Sugar().Errorf("get EgressPolicy %v err:%v", req.NamespacedName, err)
			return reconcile.Result{}, err
		}
		deleted = true
	}

	deleted = deleted || !egp.GetDeletionTimestamp().IsZero()

	policy := egress.Policy{Name: req.Name, Namespace: req.Namespace}
	if deleted {
		egList := &egress.EgressGatewayList{}
		if err := r.client.List(context.Background(), egList); err != nil {
			return reconcile.Result{Requeue: true}, nil
		}
		for _, eg := range egList.Items {
			_, isExist := GetEIPStatusByPolicy(policy, eg)
			if isExist {
				log.Sugar().Infof("delete policy %v from eg %v", policy, eg.Name)
				// Delete the policy from the EgressGateway. If the referenced EIP is not used by any other policy,
				// the system reclaims the EIP.
				DeletePolicyFromEG(policy, &eg)

				log.Sugar().Debugf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
				err = r.client.Status().Update(ctx, &eg)
				if err != nil {
					log.Sugar().Errorf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
					return reconcile.Result{Requeue: true}, err
				}
				return reconcile.Result{}, nil
			}
		}
		return reconcile.Result{}, nil
	}

	egName := egp.Spec.EgressGatewayName
	eg := &egress.EgressGateway{}
	err = r.client.Get(ctx, types.NamespacedName{Name: egName}, eg)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		log.Sugar().Errorf("get EgressGateway err:%v", err)
		return reconcile.Result{Requeue: true}, err
	}

	// Assigned if the policy does not have a gateway node
	eipStatus, isExist := GetEIPStatusByPolicy(policy, *eg)
	if !isExist {
		perNodeMap := make(map[string]egress.EgressIPStatus, 0)
		for _, item := range eg.Status.NodeList {
			perNodeMap[item.Name] = item
		}

		err := r.reAllocatorPolicy(ctx, policy, eg, perNodeMap)
		if err != nil {
			r.log.Sugar().Errorf("reallocator Failed to reassign a gateway node for EgressPolicy %v: %v", policy, err)
			return reconcile.Result{Requeue: true}, err
		}

		var perNodeList []egress.EgressIPStatus
		for _, node := range perNodeMap {
			perNodeList = append(perNodeList, node)
		}
		eg.Status.NodeList = perNodeList

		isUpdete = true
	} else {
		// Check whether the EIP is correct
		for i, eip := range eipStatus.Eips {
			for j, p := range eip.Policies {
				if p == policy {
					isReAllocatorPolicy := false
					if egp.Spec.EgressIP.UseNodeIP && (eip.IPv4 != "" || eip.IPv6 != "") {
						isReAllocatorPolicy = true
					} else if egp.Spec.EgressIP.IPv4 != "" && egp.Spec.EgressIP.IPv4 != eip.IPv4 {
						isReAllocatorPolicy = true
					} else if egp.Spec.EgressIP.IPv6 != "" && egp.Spec.EgressIP.IPv6 != eip.IPv6 {
						isReAllocatorPolicy = true
					}

					if isReAllocatorPolicy {
						eipStatus.Eips[i].Policies = append(eipStatus.Eips[i].Policies[:j], eipStatus.Eips[i].Policies[j+1:]...)
						perNodeMap := make(map[string]egress.EgressIPStatus, 0)
						for _, node := range eg.Status.NodeList {
							if node.Name == eipStatus.Name {
								perNodeMap[node.Name] = eipStatus
							} else {
								perNodeMap[node.Name] = node
							}
						}

						err := r.reAllocatorPolicy(ctx, policy, eg, perNodeMap)
						if err != nil {
							r.log.Sugar().Errorf("reallocator Failed to reassign a gateway node for EgressPolicy %v: %v", policy, err)
							return reconcile.Result{Requeue: true}, err
						}

						var perNodeList []egress.EgressIPStatus
						for _, node := range perNodeMap {
							perNodeList = append(perNodeList, node)
						}
						eg.Status.NodeList = perNodeList
					}

					isUpdete = true
					goto update
				}
			}
		}

	}

update:
	if isUpdete {
		r.log.Sugar().Debugf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
		err = r.client.Status().Update(ctx, eg)
		if err != nil {
			r.log.Sugar().Errorf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r egnReconciler) deleteNodeFromEGs(ctx context.Context, nodeName string, egList *egress.EgressGatewayList) error {
	for _, eg := range egList.Items {
		for _, eipStatus := range eg.Status.NodeList {
			if nodeName == eipStatus.Name {
				err := r.deleteNodeFromEG(ctx, nodeName, eg)
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
func (r egnReconciler) deleteNodeFromEG(ctx context.Context, nodeName string, eg egress.EgressGateway) error {
	// Get the policy that needs to be reassigned
	policies, isExist := GetEIPStatusByNode(nodeName, eg)

	if isExist {
		perNodeMap := make(map[string]egress.EgressIPStatus, 0)
		for _, item := range eg.Status.NodeList {
			if nodeName != item.Name {
				perNodeMap[item.Name] = item
			}
		}

		// Redistribute network gateway nodes
		for _, policy := range policies {
			err := r.reAllocatorPolicy(ctx, policy, &eg, perNodeMap)
			if err != nil {
				r.log.Sugar().Errorf("reallocator Failed to reassign a gateway node for EgressPolicy %v: %v", policy, err)
				return err
			}
		}

		var perNodeList []egress.EgressIPStatus
		for _, node := range perNodeMap {
			perNodeList = append(perNodeList, node)
		}

		eg.Status.NodeList = perNodeList
		r.log.Sugar().Debugf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
		err := r.client.Status().Update(ctx, &eg)
		if err != nil {
			r.log.Sugar().Errorf("update egress gateway status\n%s", mustMarshalJson(eg.Status))
			return err
		}
	}

	return nil
}

func (r egnReconciler) reAllocatorPolicy(ctx context.Context, policy egress.Policy, eg *egress.EgressGateway, nodeMap map[string]egress.EgressIPStatus) error {
	egp := &egress.EgressPolicy{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, egp)
	if err != nil {
		return err
	}

	perNode, err := r.allocatorNode("rr", nodeMap)
	if err != nil {
		return err
	}

	ipv4, ipv6, err := r.allocatorEIP("", perNode, *egp, *eg)
	if err != nil {
		return err
	}

	err = setEipStatus(ipv4, ipv6, perNode, policy, nodeMap)
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

func (r egnReconciler) allocatorEIP(selEipLolicy string, nodeName string, egp egress.EgressPolicy, eg egress.EgressGateway) (string, string, error) {

	if egp.Spec.EgressIP.UseNodeIP {
		return "", "", nil
	}

	var perIpv4 string
	var perIpv6 string

	if r.config.FileConfig.EnableIPv4 {
		var useIpv4s []net.IP
		var useIpv4sByNode []net.IP

		ipv4Ranges, _ := utils.MergeIPRanges(constant.IPv4, eg.Spec.Ranges.IPv4)

		perIpv4 = egp.Spec.EgressIP.IPv4
		if len(perIpv4) != 0 {
			result, err := utils.IsIPIncludedRange(constant.IPv4, perIpv4, ipv4Ranges)
			if err != nil {
				return "", "", err
			}
			if !result {
				return "", "", fmt.Errorf("%v is not within the EIP range of EgressGateway %v", perIpv4, eg.Name)
			}
		} else {
			for _, node := range eg.Status.NodeList {
				for _, eip := range node.Eips {
					if len(eip.IPv4) != 0 {
						useIpv4s = append(useIpv4s, net.ParseIP(eip.IPv4))
					}
				}
			}

			ipv4s, _ := utils.ParseIPRanges(constant.IPv4, ipv4Ranges)
			freeIpv4s := utils.IPsDiffSet(ipv4s, useIpv4s, false)
			rander := rand.New(rand.NewSource(time.Now().UnixNano()))

			if len(freeIpv4s) == 0 {
				for _, node := range eg.Status.NodeList {
					if node.Name == nodeName {
						for _, eip := range node.Eips {
							if len(eip.IPv4) != 0 {
								useIpv4sByNode = append(useIpv4sByNode, net.ParseIP(eip.IPv4))
							}
						}
					}
				}

				if len(useIpv4sByNode) == 0 {
					return "", "", fmt.Errorf("No EIP meeting requirements is found on node %v; EG %v", nodeName, eg.Name)
				}

				perIpv4 = useIpv4sByNode[rander.Intn(len(useIpv4sByNode))].String()
			} else {
				perIpv4 = freeIpv4s[rander.Intn(len(freeIpv4s))].String()
			}
		}
	}

	if r.config.FileConfig.EnableIPv6 {
		if len(perIpv4) != 0 && len(GetEipByIP(perIpv4, eg).IPv6) != 0 {
			return perIpv4, GetEipByIP(perIpv4, eg).IPv6, nil
		}

		var useIpv6s []net.IP
		var useIpv6sByNode []net.IP

		ipv6Ranges, _ := utils.MergeIPRanges(constant.IPv6, eg.Spec.Ranges.IPv6)

		perIpv6 = egp.Spec.EgressIP.IPv6
		if len(perIpv6) != 0 {
			result, err := utils.IsIPIncludedRange(constant.IPv6, perIpv6, ipv6Ranges)
			if err != nil {
				return "", "", err
			}
			if !result {
				return "", "", fmt.Errorf("%v is not within the EIP range of EgressGateway %v", perIpv6, eg.Name)
			}
		} else {
			for _, node := range eg.Status.NodeList {
				for _, eip := range node.Eips {
					if len(eip.IPv6) != 0 {
						useIpv6s = append(useIpv6s, net.ParseIP(eip.IPv6))
					}
				}
			}

			ipv6s, _ := utils.ParseIPRanges(constant.IPv6, ipv6Ranges)
			freeIpv6s := utils.IPsDiffSet(ipv6s, useIpv6s, false)
			rander := rand.New(rand.NewSource(time.Now().UnixNano()))

			if len(freeIpv6s) == 0 {
				for _, node := range eg.Status.NodeList {
					if node.Name == nodeName {
						for _, eip := range node.Eips {
							if len(eip.IPv6) != 0 {
								useIpv6sByNode = append(useIpv6sByNode, net.ParseIP(eip.IPv6))
							}
						}
					}
				}

				if len(useIpv6sByNode) == 0 {
					return "", "", fmt.Errorf("No EIP meeting requirements is found on node %v; EG %v", nodeName, eg.Name)
				}
				perIpv6 = useIpv6sByNode[rander.Intn(len(useIpv6sByNode))].String()
			} else {
				perIpv6 = freeIpv6s[rander.Intn(len(freeIpv6s))].String()
			}
		}
	}

	return perIpv4, perIpv6, nil
}

func NewEgressGatewayController(mgr manager.Manager, log *zap.Logger, cfg *config.Config) error {
	if log == nil {
		return fmt.Errorf("log can not be nil")
	}
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

	if err := c.Watch(&source.Kind{Type: &egress.EgressGateway{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGateway"))); err != nil {
		return fmt.Errorf("failed to watch EgressGateway: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &corev1.Node{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("Node"))); err != nil {
		return fmt.Errorf("failed to watch Node: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &egress.EgressPolicy{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressPolicy"))); err != nil {
		return fmt.Errorf("failed to watch EgressPolicy: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &egress.EgressNode{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressNode"))); err != nil {
		return fmt.Errorf("failed to watch EgressNode: %w", err)
	}

	return nil
}

func GetEipByIP(ipv4 string, eg egress.EgressGateway) egress.Eips {
	var eipInfo egress.Eips
	for _, node := range eg.Status.NodeList {
		for _, eip := range node.Eips {
			if eip.IPv4 == ipv4 {
				eipInfo = eip
			}
		}
	}

	return eipInfo
}

func setEipStatus(ipv4, ipv6 string, nodeName string, policy egress.Policy, nodeMap map[string]egress.EgressIPStatus) error {
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
		nodeMap[nodeName] = newEipStatus
	}

	return nil
}

func mustMarshalJson(obj interface{}) string {
	raw, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(raw)
}

func GetEIPStatusByNode(nodeName string, eg egress.EgressGateway) ([]egress.Policy, bool) {

	var eipStatus egress.EgressIPStatus
	var policies []egress.Policy
	isExist := false
	for _, node := range eg.Status.NodeList {
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

func GetEIPStatusByPolicy(policy egress.Policy, eg egress.EgressGateway) (egress.EgressIPStatus, bool) {
	var eipStatus egress.EgressIPStatus
	isExist := false

	for _, item := range eg.Status.NodeList {
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

func DeletePolicyFromEG(policy egress.Policy, eg *egress.EgressGateway) {
	var policies []egress.Policy
	var eips []egress.Eips

	for i, node := range eg.Status.NodeList {
		for j, eip := range node.Eips {
			for k, item := range eip.Policies {
				if item == policy {
					policies = append(eip.Policies[:k], eip.Policies[k+1:]...)

					if len(policies) == 0 {
						// Release EIP
						for x, e := range node.Eips {
							if eip.IPv4 == e.IPv4 || eip.IPv6 == e.IPv6 {
								eips = append(node.Eips[:x], node.Eips[x+1:]...)
								break
							}
						}
						eg.Status.NodeList[i].Eips = eips
					} else {
						eg.Status.NodeList[i].Eips[j].Policies = policies
					}
					goto breakHere
				}
			}
		}
	}
breakHere:
	return
}
