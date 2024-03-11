// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/constant"
	egress "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
	"github.com/spidernet-io/egressgateway/pkg/utils/slice"
)

type egnReconciler struct {
	client client.Client
	log    logr.Logger
	config *config.Config
	cli    client.Client
}

func (r *egnReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		return reconcile.Result{}, err
	}

	log := r.log.WithValues("kind", kind)

	switch kind {
	case "EgressGateway":
		return r.reconcileGateway(ctx, newReq, log)
	case "EgressClusterPolicy":
		return r.reconcileEgressClusterPolicy(ctx, newReq, log)
	case "EgressPolicy":
		return r.reconcileEgressPolicy(ctx, newReq, log)
	case "Node":
		return r.reconcileNode(ctx, newReq, log)
	case "EgressTunnel":
		return r.reconcileTunnel(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

func (r *egnReconciler) reconcileTunnel(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	tunnel := &egress.EgressTunnel{}
	err := r.client.Get(ctx, req.NamespacedName, tunnel)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !tunnel.GetDeletionTimestamp().IsZero()

	if deleted {
		// case 1, tunnel delete
		//         move ip

		egwList := new(egress.EgressGatewayList)
		err := r.cli.List(ctx, egwList)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}

		for _, egw := range egwList.Items {
			var needMoveIPs []egress.Eips
			needUpdate := false
			for nodeIndex, node := range egw.Status.NodeList {
				if node.Name == req.Name {
					needUpdate = true
					needMoveIPs = append(needMoveIPs, node.Eips...)
					egw.Status.NodeList = append(egw.Status.NodeList[:nodeIndex], egw.Status.NodeList[nodeIndex+1:]...)
					break
				}
			}
			if len(needMoveIPs) > 0 {
				moveEipToReadyNode(&egw, needMoveIPs)
			}
			if needUpdate {
				err := updateGatewayStatusWithUsage(ctx, r.client, &egw)
				if err != nil {
					return reconcile.Result{Requeue: true}, err
				}
				// sync all policy status
				err = updateAllPolicyStatus(ctx, r.client, &egw)
				if err != nil {
					return reconcile.Result{Requeue: true}, err
				}
			}
		}
		return reconcile.Result{}, nil
	}

	fmt.Println("update tunnel")

	egwList := new(egress.EgressGatewayList)
	err = r.cli.List(ctx, egwList)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	for _, egw := range egwList.Items {
		var needMoveIPs []egress.Eips
		needUpdate := false
		for nodeIndex, node := range egw.Status.NodeList {
			// case 2, tunnel update
			if node.Name == req.Name {
				if tunnel.Status.Phase == egress.EgressTunnelReady {
					// case 2.1: other status (e.g. NodeNotReady) -> Ready
					if tunnel.Status.Phase.IsNotEqual(node.Status) {
						needUpdate = true
						egw.Status.NodeList[nodeIndex].Status = tunnel.Status.Phase.String()
						if egw.Status.ReadyCount() == 1 {
							// if it is the first tunnel in the node list,
							// we need do more (list all policy, recheck all)
							res, err := r.checkAndUpdateAllPolicyIfNeedWhenFirstNodeReady(ctx, req, log, &egw)
							if err != nil {
								return res, err
							}
						}
					}
				} else {
					// case 2.2: ready -> not ready / statue not sync
					//           move ip
					if len(node.Eips) > 0 {
						// need move ip
						needUpdate = true
						needMoveIPs = append(needMoveIPs, node.Eips...)
						egw.Status.NodeList[nodeIndex].Eips = make([]egress.Eips, 0)
					}
					// check state are sync
					if tunnel.Status.Phase.IsNotEqual(node.Status) {
						needUpdate = true
						egw.Status.NodeList[nodeIndex].Status = tunnel.Status.Phase.String()
					}
				}
				break
			}
		}
		if len(needMoveIPs) > 0 {
			moveEipToReadyNode(&egw, needMoveIPs)
		}
		if needUpdate {
			err := updateGatewayStatusWithUsage(ctx, r.client, &egw)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			// sync all policy status
			err = updateAllPolicyStatus(ctx, r.client, &egw)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func (r *egnReconciler) checkAndUpdateAllPolicyIfNeedWhenFirstNodeReady(ctx context.Context,
	req reconcile.Request, log logr.Logger, egw *egress.EgressGateway) (reconcile.Result, error) {

	clusterPolicy := new(egress.EgressClusterPolicyList)
	err := r.client.List(ctx, clusterPolicy)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	for _, p := range clusterPolicy.Items {
		if p.Spec.EgressGatewayName == egw.Name {
			res, err := r.reAssignEgressClusterPolicyIP(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: p.Namespace, Name: p.Name},
			}, egw, &p)
			if err != nil {
				return res, err
			}
		}
	}
	policy := new(egress.EgressPolicyList)
	err = r.client.List(ctx, policy)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	for _, p := range policy.Items {
		if p.Spec.EgressGatewayName == egw.Name {
			res, err := r.reAssignEgressPolicyIP(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: p.Namespace, Name: p.Name},
			}, egw, &p)
			if err != nil {
				return res, err
			}
		}
	}
	return reconcile.Result{}, nil
}

func (r *egnReconciler) reAssignEgressPolicyIP(ctx context.Context,
	req reconcile.Request, gateway *egress.EgressGateway, policy *egress.EgressPolicy) (reconcile.Result, error) {

	var err error
	assignedIP := getAssignedIP(gateway, req.Namespace, req.Name)
	if assignedIP == nil {
		assignedIP, err = assignIP(gateway, req, policy.Spec.EgressIP)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		if assignedIP == nil {
			return reconcile.Result{Requeue: true}, fmt.Errorf("not enough ip")
		}
		err = updateGatewayStatusWithUsage(ctx, r.client, gateway)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		//err = updateEgressPolicyIfNeed(ctx, r.client, policy, assignedIP)
		//if err != nil {
		//	return reconcile.Result{Requeue: true}, err
		//}
		err := updateEgressPolicyStatusIfNeed(ctx, r.client, policy, assignedIP)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	} else {
		err := updateEgressPolicyStatusIfNeed(ctx, r.client, policy, assignedIP)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *egnReconciler) reAssignEgressClusterPolicyIP(ctx context.Context,
	req reconcile.Request, gateway *egress.EgressGateway, policy *egress.EgressClusterPolicy) (reconcile.Result, error) {

	var err error

	assignedIP := getAssignedIP(gateway, req.Namespace, req.Name)
	if assignedIP == nil {
		assignedIP, err = assignIP(gateway, req, policy.Spec.EgressIP)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		if assignedIP == nil {
			return reconcile.Result{Requeue: true}, fmt.Errorf("not enough ip")
		}
		err := updateEgressClusterPolicyStatusIfNeed(ctx, r.client, policy, assignedIP)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	} else {
		err := updateEgressClusterPolicyStatusIfNeed(ctx, r.client, policy, assignedIP)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *egnReconciler) reconcileNode(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	node := &corev1.Node{}
	err := r.client.Get(ctx, req.NamespacedName, node)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !node.GetDeletionTimestamp().IsZero()

	if deleted {
		// case1: do nothing
		return reconcile.Result{}, nil
	}

	// case2: node label update
	egwList := new(egress.EgressGatewayList)
	err = r.cli.List(ctx, egwList)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	for _, egw := range egwList.Items {
		selector, err := metav1.LabelSelectorAsSelector(egw.Spec.NodeSelector.Selector)
		if err != nil {
			return reconcile.Result{}, err
		}
		//
		needUpdate := false
		if selector.Matches(labels.Set(node.Labels)) {
			// case2.1: label match
			// case2.1.1: not in list, add it
			// case2.1.1: already int list, do nothing
			var find bool
			for _, item := range egw.Status.NodeList {
				if item.Name == node.Name {
					find = true
					break
				}
			}
			if !find {
				needUpdate = true
				status := egress.EgressTunnelPending.String()
				tunnel := new(egress.EgressTunnel)
				err := r.client.Get(ctx, types.NamespacedName{Name: node.Name}, tunnel)
				if err != nil {
					if !errors.IsNotFound(err) {
						return reconcile.Result{}, err
					}
				} else {
					status = tunnel.Status.Phase.String()
				}
				egw.Status.NodeList = append(egw.Status.NodeList, egress.EgressIPStatus{
					Name:   node.Name,
					Eips:   make([]egress.Eips, 0),
					Status: status,
				})
				// if it is the first ready
				// if it is the first tunnel in the node list,
				// we need do more (list all policy, recheck all)
				res, err := r.checkAndUpdateAllPolicyIfNeedWhenFirstNodeReady(ctx, req, log, &egw)
				if err != nil {
					return res, err
				}
			}
		} else {
			// case2.2: label not match
			// case2.2.1: not in list, do nothing
			// case2.2.1: already int list, delete it
			var needMoveIPs []egress.Eips
			for nodeIndex, item := range egw.Status.NodeList {
				if item.Name == node.Name {
					needUpdate = true
					needMoveIPs = item.Eips
					egw.Status.NodeList = append(egw.Status.NodeList[:nodeIndex], egw.Status.NodeList[nodeIndex+1:]...)
					break
				}
			}
			if len(needMoveIPs) > 0 {
				moveEipToReadyNode(&egw, needMoveIPs)
			}
		}
		if needUpdate {
			err := updateGatewayStatusWithUsage(ctx, r.client, &egw)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			// sync all policy status
			err = updateAllPolicyStatus(ctx, r.client, &egw)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

func updateAllPolicyStatus(ctx context.Context, cli client.Client, egw *egress.EgressGateway) error {
	for _, node := range egw.Status.NodeList {
		for _, eip := range node.Eips {
			assignedIP := &AssignedIP{
				Node:      node.Name,
				IPv4:      eip.IPv4,
				IPv6:      eip.IPv6,
				UseNodeIP: false,
			}
			if assignedIP.IPv4 == "" && assignedIP.IPv6 == "" {
				assignedIP.UseNodeIP = true
			}
			for _, p := range eip.Policies {
				if p.Namespace != "" {
					policy := new(egress.EgressPolicy)
					err := cli.Get(ctx, types.NamespacedName{Namespace: p.Namespace, Name: p.Name}, policy)
					if err != nil {
						return err
					}
					err = updateEgressPolicyStatusIfNeed(ctx, cli, policy, assignedIP)
					if err != nil {
						return err
					}
				} else {
					policy := new(egress.EgressClusterPolicy)
					err := cli.Get(ctx, types.NamespacedName{Namespace: p.Namespace, Name: p.Name}, policy)
					if err != nil {
						return err
					}
					err = updateEgressClusterPolicyStatusIfNeed(ctx, cli, policy, assignedIP)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (r *egnReconciler) reconcileGateway(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	egw := &egress.EgressGateway{}
	err := r.cli.Get(ctx, req.NamespacedName, egw)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !egw.GetDeletionTimestamp().IsZero()

	if deleted {
		// case 1
		log.Info("request item is deleted")
		count, err := getPolicyCountByGatewayName(ctx, r.client, req.Name)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		if count == 0 && egw.Name != "" {
			log.Info("remove the egressGatewayFinalizer")
			removeEgressGatewayFinalizer(egw)
			log.V(1).Info("remove the egressGatewayFinalizer", "ObjectMeta", egw.ObjectMeta)
			err = r.client.Update(ctx, egw)
			if err != nil {
				log.Error(err, "remove the egressGatewayFinalizer", "ObjectMeta", egw.ObjectMeta)
				return reconcile.Result{Requeue: true}, err
			}
		}
		return reconcile.Result{Requeue: false}, nil
	}

	// case2: egw match label update
	k8sNodeList := &corev1.NodeList{}
	selector, err := metav1.LabelSelectorAsSelector(egw.Spec.NodeSelector.Selector)
	if err != nil {
		return reconcile.Result{}, err
	}
	opt := &client.ListOptions{LabelSelector: selector}
	err = r.client.List(ctx, k8sNodeList, opt)
	if err != nil {
		return reconcile.Result{}, err
	}

	//
	k8sNodeMap := make(map[string]struct{})
	for _, node := range k8sNodeList.Items {
		k8sNodeMap[node.Name] = struct{}{}
	}

	needUpdate := false

	// need to move eips
	var needMoveIPs []egress.Eips

	for i := 0; i < len(egw.Status.NodeList); {
		node := egw.Status.NodeList[i]
		if _, ok := k8sNodeMap[node.Name]; ok {
			delete(k8sNodeMap, node.Name)
			tunnel := new(egress.EgressTunnel)
			err := r.client.Get(ctx, types.NamespacedName{Name: node.Name}, tunnel)
			if err != nil {
				if !errors.IsNotFound(err) {
					return reconcile.Result{}, err
				}
				// case 1.1: node exists, but tunnel not exists, set statue to pending
				if egress.EgressTunnelPending.IsNotEqual(node.Status) {
					needMoveIPs = append(needMoveIPs, node.Eips...)
					// sync status
					egw.Status.NodeList[i].Status = egress.EgressTunnelPending.String()
					egw.Status.NodeList[i].Eips = []egress.Eips{}
					needUpdate = true
				}
				continue
			}
			// case 1.2: node exists, but node status not equal tunnel statue
			// just sync it
			if tunnel.Status.Phase.IsNotEqual(node.Status) {
				egw.Status.NodeList[i].Status = egress.EgressTunnelPending.String()
				needUpdate = true
			}
			// case 1.3: status has been synchronized, do nothing
			i++
		} else {
			needMoveIPs = append(needMoveIPs, node.Eips...)
			needUpdate = true
			// Remove the element by shifting the subsequent elements forward and reducing the length of the slice.
			copy(egw.Status.NodeList[i:], egw.Status.NodeList[i+1:])
			egw.Status.NodeList = egw.Status.NodeList[:len(egw.Status.NodeList)-1]
		}
	}

	// add k8s nodes to gateway match list
	beforeReadyCount := egw.Status.ReadyCount()
	for node := range k8sNodeMap {
		fmt.Println("")
		needUpdate = true
		status := egress.EgressTunnelPending.String()
		tunnel := new(egress.EgressTunnel)
		err := r.client.Get(ctx, types.NamespacedName{Name: node}, tunnel)
		if err != nil {
			if !errors.IsNotFound(err) {
				return reconcile.Result{}, err
			}
		} else {
			status = tunnel.Status.Phase.String()
		}
		egw.Status.NodeList = append(egw.Status.NodeList, egress.EgressIPStatus{
			Name:   node,
			Eips:   make([]egress.Eips, 0),
			Status: status,
		})
	}

	if len(needMoveIPs) > 0 {
		moveEipToReadyNode(egw, needMoveIPs)
	}

	if beforeReadyCount == 0 {
		res, err := r.checkAndUpdateAllPolicyIfNeedWhenFirstNodeReady(ctx, req, log, egw)
		if err != nil {
			return res, err
		}
	}

	if needUpdate {
		// update
		err := updateGatewayStatusWithUsage(ctx, r.client, egw)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		// sync all policy status
		err = updateAllPolicyStatus(ctx, r.client, egw)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{}, nil
}

func moveEipToReadyNode(gateway *egress.EgressGateway, needMoveIPs []egress.Eips) {
	if gateway.Status.ReadyCount() <= 0 {
		return
	}

	minEipNodeIndex := -1
	minEipCount := -1
	useNodeIPIndex := -1

	for i, node := range gateway.Status.NodeList {
		if node.Status == "Ready" {
			eipCount := len(node.Eips)
			if minEipCount == -1 || eipCount < minEipCount {
				minEipNodeIndex = i
				minEipCount = eipCount
				for tmp, eip := range node.Eips {
					if eip.IPv4 == "" && eip.IPv6 == "" {
						useNodeIPIndex = tmp
					}
				}
			}
		}
	}

	if minEipNodeIndex != -1 {
		for _, eip := range needMoveIPs {
			if eip.IPv4 == "" && eip.IPv6 == "" {
				// case 1: move user node ip case
				if useNodeIPIndex != -1 {
					// case: append policy to target node
					gateway.Status.NodeList[minEipNodeIndex].Eips[useNodeIPIndex].Policies = append(
						gateway.Status.NodeList[minEipNodeIndex].Eips[useNodeIPIndex].Policies,
						eip.Policies...,
					)
				} else {
					// case: need create new
					useNodeIPIndex = len(gateway.Status.NodeList[minEipNodeIndex].Eips)
					gateway.Status.NodeList[minEipNodeIndex].Eips = append(
						gateway.Status.NodeList[minEipNodeIndex].Eips,
						egress.Eips{IPv4: "", IPv6: "", Policies: eip.Policies},
					)
				}
			} else {
				// case 2: move eip case
				gateway.Status.NodeList[minEipNodeIndex].Eips = append(gateway.Status.NodeList[minEipNodeIndex].Eips, eip)
			}
		}
	}

	// case: no healthy nodes to move(migrate) eip, do nothing
}

func (r *egnReconciler) reconcileEgressClusterPolicy(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	policy := new(egress.EgressClusterPolicy)
	err := r.client.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !policy.GetDeletionTimestamp().IsZero()
	if deleted {
		return r.reconcileDeletePolicy(ctx, req, policy.Spec.EgressGatewayName, log)
	}

	if policy != nil && policy.Name != "" && policy.Spec.EgressGatewayName != "" {
		gateway := new(egress.EgressGateway)
		gatewayName := policy.Spec.EgressGatewayName
		err := r.cli.Get(ctx, types.NamespacedName{Name: gatewayName}, gateway)
		if err != nil {
			if !errors.IsNotFound(err) {
				return reconcile.Result{Requeue: true}, err
			}
			return reconcile.Result{Requeue: false}, fmt.Errorf("reconcile EgressPolicy %s, not found egress gateway: %s", req, gatewayName)
		}
		assignedIP := getAssignedIP(gateway, req.Namespace, req.Name)
		if assignedIP == nil {
			assignedIP, err = assignIP(gateway, req, policy.Spec.EgressIP)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			if assignedIP == nil {
				return reconcile.Result{Requeue: true}, fmt.Errorf("not enough ip")
			}
			err = updateGatewayStatusWithUsage(ctx, r.client, gateway)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			err := updateEgressClusterPolicyStatusIfNeed(ctx, r.client, policy, assignedIP)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
		} else {
			err := updateEgressClusterPolicyStatusIfNeed(ctx, r.client, policy, assignedIP)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

func (r *egnReconciler) reconcileDeletePolicy(ctx context.Context, req reconcile.Request, egwName string, log logr.Logger) (reconcile.Result, error) {
	if egwName == "" {
		gatewayList := new(egress.EgressGatewayList)
		err := r.cli.List(ctx, gatewayList)
		if err != nil {
			if !errors.IsNotFound(err) {
				return reconcile.Result{Requeue: true}, err
			}
			return reconcile.Result{Requeue: false}, nil
		}
		if len(gatewayList.Items) == 0 {
			return reconcile.Result{Requeue: false}, nil
		}
		for _, gateway := range gatewayList.Items {
			update, err := deleteEgressPolicy(&gateway, req.Namespace, req.Name)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			if update {
				err := updateGatewayStatusWithUsage(ctx, r.client, &gateway)
				if err != nil {
					return reconcile.Result{Requeue: true}, err
				}
				break
			}
		}
	} else {
		gateway := new(egress.EgressGateway)
		err := r.cli.Get(ctx, types.NamespacedName{Name: egwName}, gateway)
		if err != nil {
			if !errors.IsNotFound(err) {
				return reconcile.Result{Requeue: true}, err
			}
			return reconcile.Result{Requeue: false}, nil
		}
		if gateway.Name != "" {
			update, err := deleteEgressPolicy(gateway, req.Namespace, req.Name)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			if update {
				err := updateGatewayStatusWithUsage(ctx, r.client, gateway)
				if err != nil {
					return reconcile.Result{Requeue: true}, err
				}
			}
		}
	}
	return reconcile.Result{Requeue: false}, nil
}

func (r *egnReconciler) reconcileEgressPolicy(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	policy := new(egress.EgressPolicy)
	err := r.client.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !policy.GetDeletionTimestamp().IsZero()

	if deleted {
		egwName := ""
		if policy != nil && policy.Spec.EgressGatewayName != "" {
			egwName = policy.Spec.EgressGatewayName
		}
		if egwName == "" {
			gatewayList := new(egress.EgressGatewayList)
			err := r.cli.List(ctx, gatewayList)
			if err != nil {
				if !errors.IsNotFound(err) {
					return reconcile.Result{Requeue: true}, err
				}
				return reconcile.Result{Requeue: false}, nil
			}
			if len(gatewayList.Items) == 0 {
				return reconcile.Result{Requeue: false}, nil
			}
			for _, gateway := range gatewayList.Items {
				update, err := deleteEgressPolicy(&gateway, req.Namespace, req.Name)
				if err != nil {
					return reconcile.Result{Requeue: true}, err
				}
				if update {
					err := updateGatewayStatusWithUsage(ctx, r.client, &gateway)
					if err != nil {
						return reconcile.Result{Requeue: true}, err
					}
					break
				}
			}
		} else {
			gateway := new(egress.EgressGateway)
			err := r.cli.Get(ctx, types.NamespacedName{Name: egwName}, gateway)
			if err != nil {
				if !errors.IsNotFound(err) {
					return reconcile.Result{Requeue: true}, err
				}
				return reconcile.Result{Requeue: false}, nil
			}
			if gateway.Name != "" {
				update, err := deleteEgressPolicy(gateway, req.Namespace, req.Name)
				if err != nil {
					return reconcile.Result{Requeue: true}, err
				}
				if update {
					err := updateGatewayStatusWithUsage(ctx, r.client, gateway)
					if err != nil {
						return reconcile.Result{Requeue: true}, err
					}
				}
			}
		}
		return reconcile.Result{Requeue: false}, nil
	}

	if policy != nil && policy.Name != "" && policy.Spec.EgressGatewayName != "" {
		gateway := new(egress.EgressGateway)
		gatewayName := policy.Spec.EgressGatewayName
		err := r.cli.Get(ctx, types.NamespacedName{Name: gatewayName}, gateway)
		if err != nil {
			if !errors.IsNotFound(err) {
				return reconcile.Result{Requeue: true}, err
			}
			return reconcile.Result{Requeue: false}, fmt.Errorf("reconcile EgressPolicy %s, not found egress gateway: %s", req, gatewayName)
		}
		assignedIP := getAssignedIP(gateway, req.Namespace, req.Name)
		if assignedIP == nil {
			assignedIP, err = assignIP(gateway, req, policy.Spec.EgressIP)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			if assignedIP == nil {
				return reconcile.Result{Requeue: true}, fmt.Errorf("not enough ip")
			}
			err = updateGatewayStatusWithUsage(ctx, r.client, gateway)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			//err = updateEgressPolicyIfNeed(ctx, r.client, policy, assignedIP)
			//if err != nil {
			//	return reconcile.Result{Requeue: true}, err
			//}
			err := updateEgressPolicyStatusIfNeed(ctx, r.client, policy, assignedIP)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
		} else {
			err := updateEgressPolicyStatusIfNeed(ctx, r.client, policy, assignedIP)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func assignIP(from *egress.EgressGateway, req reconcile.Request, specEgressIP egress.EgressIP) (*AssignedIP, error) {
	// apply node policy to select node
	nIndex := -1
	eipNum := -1
	for nodeIndex, node := range from.Status.NodeList {
		if node.Status != string(egress.EgressTunnelReady) {
			continue
		}
		if eipNum == -1 {
			eipNum = len(node.Eips)
			nIndex = nodeIndex
		} else if eipNum > len(node.Eips) {
			eipNum = len(node.Eips)
			nIndex = nodeIndex
		}
	}

	// case1
	if specEgressIP.UseNodeIP {
		if nIndex != -1 {
			for eipIndex, eip := range from.Status.NodeList[nIndex].Eips {
				if eip.IPv4 == "" && eip.IPv6 == "" {
					from.Status.NodeList[nIndex].Eips[eipIndex].Policies = append(
						from.Status.NodeList[nIndex].Eips[eipIndex].Policies,
						egress.Policy{Name: req.Name, Namespace: req.Namespace},
					)
					return &AssignedIP{
						Node:      from.Status.NodeList[nIndex].Name,
						UseNodeIP: true,
					}, nil
				}
			}
			// not found
			from.Status.NodeList[nIndex].Eips = append(
				from.Status.NodeList[nIndex].Eips,
				egress.Eips{
					IPv4: "", IPv6: "",
					Policies: []egress.Policy{{Name: req.Name, Namespace: req.Namespace}},
				},
			)
			return &AssignedIP{
				Node:      from.Status.NodeList[nIndex].Name,
				UseNodeIP: true,
			}, nil
		}
		return nil, nil
	}

	// case2 reuse eip
	if specEgressIP.IPv4 != "" || specEgressIP.IPv6 != "" {
		// check
		for nodeIndex, node := range from.Status.NodeList {
			for eipIndex, eip := range node.Eips {
				if eip.IPv4 == specEgressIP.IPv4 || eip.IPv6 == specEgressIP.IPv6 {
					from.Status.NodeList[nodeIndex].Eips[eipIndex].Policies = append(
						from.Status.NodeList[nodeIndex].Eips[eipIndex].Policies, egress.Policy{
							Name:      req.Name,
							Namespace: req.Namespace,
						})
					return &AssignedIP{
						Node:      node.Name,
						IPv4:      eip.IPv4,
						IPv6:      eip.IPv6,
						UseNodeIP: false,
					}, nil
				}
			}
		}
	}

	// case3 assign new IP use eip assign policy
	//
	if specEgressIP.AllocatorPolicy == egress.EipAllocatorRR {
		randObj := rand.New(rand.NewSource(time.Now().UnixNano()))
		assignedIP := &AssignedIP{
			Node:      "",
			IPv4:      "",
			IPv6:      "",
			UseNodeIP: false,
		}
		if len(from.Spec.Ippools.IPv4) > 0 {
			ipv4Ranges, err := ip.MergeIPRanges(constant.IPv4, from.Spec.Ippools.IPv4)
			if err != nil {
				return nil, fmt.Errorf("assignIP MergeIPRanges with error: %s", err)
			}
			// user specify ipv4
			if specEgressIP.IPv4 != "" {
				ok, err := ip.IsIPIncludedRange(constant.IPv4, specEgressIP.IPv4, ipv4Ranges)
				if err != nil {
					return nil, fmt.Errorf("encountered an error while trying to check if the Egress IP of Policy %s/%s exists in the ippool: %v", req.Namespace, req.Name, err)
				}
				if !ok {
					return nil, fmt.Errorf("the specified egress IPv4 %s is not in the gateway's ippool", specEgressIP.IPv4)
				}
				assignedIP.IPv4 = specEgressIP.IPv4
			} else {
				ipv4s, err := ip.ParseIPRanges(constant.IPv4, ipv4Ranges)
				if err != nil {
					return nil, err
				}
				var useIpv4s []net.IP
				for _, node := range from.Status.NodeList {
					for _, eip := range node.Eips {
						if len(eip.IPv4) != 0 {
							useIpv4s = append(useIpv4s, net.ParseIP(eip.IPv4))
						}
					}
				}
				freeIpv4s := ip.IPsDiffSet(ipv4s, useIpv4s, false)
				if len(freeIpv4s) == 0 {
					return nil, fmt.Errorf("EgressGateway %s does not have enough IPs to allocate for Policy %s/%s", from.Name, req.Namespace, req.Name)
				}
				assignedIP.IPv4 = freeIpv4s[randObj.Intn(len(freeIpv4s))].String()
			}
		}

		if len(from.Spec.Ippools.IPv6) > 0 {
			ipv6Ranges, err := ip.MergeIPRanges(constant.IPv6, from.Spec.Ippools.IPv6)
			if err != nil {
				return nil, fmt.Errorf("assignIP MergeIPRanges with error: %s", err)
			}
			// user specify ipv6
			if specEgressIP.IPv6 != "" {
				ok, err := ip.IsIPIncludedRange(constant.IPv6, specEgressIP.IPv6, ipv6Ranges)
				if err != nil {
					return nil, fmt.Errorf("encountered an error while trying to check if the Egress IP of Policy %s/%s exists in the ippool: %v", req.Namespace, req.Name, err)
				}
				if !ok {
					return nil, fmt.Errorf("the specified egress IPv6 %s is not in the gateway's ippool", specEgressIP.IPv6)
				}
				assignedIP.IPv6 = specEgressIP.IPv6
			} else {
				ipv6s, err := ip.ParseIPRanges(constant.IPv6, ipv6Ranges)
				if err != nil {
					return nil, err
				}
				var useIpv6s []net.IP
				for _, node := range from.Status.NodeList {
					for _, eip := range node.Eips {
						if len(eip.IPv6) != 0 {
							useIpv6s = append(useIpv6s, net.ParseIP(eip.IPv6))
						}
					}
				}
				freeIpv6s := ip.IPsDiffSet(ipv6s, useIpv6s, false)
				if len(freeIpv6s) == 0 {
					return nil, fmt.Errorf("EgressGateway %s does not have enough IPs to allocate for Policy %s/%s", from.Name, req.Namespace, req.Name)
				}
				assignedIP.IPv6 = freeIpv6s[randObj.Intn(len(freeIpv6s))].String()
			}
		}

		assignedIP.Node = from.Status.NodeList[nIndex].Name
		// append assignedIP to egw
		from.Status.NodeList[nIndex].Eips = append(
			from.Status.NodeList[nIndex].Eips,
			egress.Eips{
				IPv4:     assignedIP.IPv4,
				IPv6:     assignedIP.IPv6,
				Policies: []egress.Policy{{Name: req.Name, Namespace: req.Namespace}},
			},
		)
		return assignedIP, nil
	} else {
		assignedIP := &AssignedIP{
			Node:      "",
			IPv4:      from.Spec.Ippools.Ipv4DefaultEIP,
			IPv6:      from.Spec.Ippools.Ipv6DefaultEIP,
			UseNodeIP: false,
		}
		defaultEipIndex := -1
		for i, node := range from.Status.NodeList {
			for _, eip := range node.Eips {
				if eip.IPv4 != "" && eip.IPv4 == from.Spec.Ippools.Ipv4DefaultEIP {
					defaultEipIndex = i
					break
				}
				if eip.IPv6 != "" && eip.IPv6 == from.Spec.Ippools.Ipv6DefaultEIP {
					defaultEipIndex = i
					break
				}
			}
			if defaultEipIndex != -1 {
				assignedIP.Node = node.Name
				break
			}
		}
		if defaultEipIndex == -1 {
			for i, node := range from.Status.NodeList {
				if node.Status != string(egress.EgressTunnelReady) {
					continue
				}
				from.Status.NodeList[i].Eips = append(
					from.Status.NodeList[i].Eips,
					egress.Eips{
						IPv4:     from.Spec.Ippools.Ipv4DefaultEIP,
						IPv6:     from.Spec.Ippools.Ipv6DefaultEIP,
						Policies: []egress.Policy{{Name: req.Name, Namespace: req.Namespace}},
					},
				)
				assignedIP.Node = node.Name
				break
			}
		}
		if assignedIP.Node == "" {
			return nil, fmt.Errorf("EgressGateway %s does not have an available Node", from.Name)
		}

		return assignedIP, nil
	}
}

type AssignedIP struct {
	Node      string
	IPv4      string
	IPv6      string
	UseNodeIP bool
}

func updateEgressPolicyIfNeed(ctx context.Context, cli client.Client, policy *egress.EgressPolicy, assignedIP *AssignedIP) error {
	if policy.Spec.EgressIP.IPv4 != assignedIP.IPv4 || policy.Spec.EgressIP.IPv4 != assignedIP.IPv6 {
		policy.Spec.EgressIP.IPv4 = assignedIP.IPv4
		policy.Spec.EgressIP.IPv6 = assignedIP.IPv6
		err := cli.Update(ctx, policy)
		if err != nil {
			return err
		}
	}
	return nil
}

func getAssignedIP(from *egress.EgressGateway, policyNs, policyName string) *AssignedIP {
	for _, node := range from.Status.NodeList {
		for _, eip := range node.Eips {
			for _, policy := range eip.Policies {
				if policy.Name == policyName && policy.Namespace == policyNs {
					return &AssignedIP{Node: node.Name, IPv4: eip.IPv4, IPv6: eip.IPv6}
				}
			}
		}
	}
	return nil
}

func updateEgressPolicyStatusIfNeed(ctx context.Context, cli client.Client, policy *egress.EgressPolicy, assignedIP *AssignedIP) error {
	if policy.Status.Eip.Ipv4 != assignedIP.IPv4 || policy.Status.Eip.Ipv6 != assignedIP.IPv6 || policy.Status.Node != assignedIP.Node {
		policy.Status.Eip.Ipv4 = assignedIP.IPv4
		policy.Status.Eip.Ipv6 = assignedIP.IPv6
		policy.Status.Node = assignedIP.Node

		err := cli.Status().Update(ctx, policy)
		if err != nil {
			if errors.IsConflict(err) {
				newPolicy := new(egress.EgressPolicy)
				err := cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, policy)
				if err != nil {
					if !errors.IsNotFound(err) {
						return nil
					}
					return err
				}
				newPolicy.Status = policy.Status
				err = cli.Status().Update(ctx, policy)
				if err != nil {
					return err
				}
			}
			return err
		}
	}
	return nil
}

func updateEgressClusterPolicyStatusIfNeed(ctx context.Context, cli client.Client, policy *egress.EgressClusterPolicy, assignedIP *AssignedIP) error {
	if policy.Status.Eip.Ipv4 != assignedIP.IPv4 || policy.Status.Eip.Ipv6 != assignedIP.IPv6 || policy.Status.Node != assignedIP.Node {
		policy.Status.Eip.Ipv4 = assignedIP.IPv4
		policy.Status.Eip.Ipv6 = assignedIP.IPv6
		policy.Status.Node = assignedIP.Node
		err := cli.Status().Update(ctx, policy)
		if err != nil {
			if errors.IsConflict(err) {
				newPolicy := new(egress.EgressPolicy)
				err := cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, policy)
				if err != nil {
					if !errors.IsNotFound(err) {
						return nil
					}
					return err
				}
				newPolicy.Status = policy.Status
				err = cli.Status().Update(ctx, policy)
				if err != nil {
					return err
				}
			}
			return err
		}
	}
	return nil
}

func updateGatewayStatusWithUsage(ctx context.Context, cli client.Client, gateway *egress.EgressGateway) error {
	if gateway == nil {
		return fmt.Errorf("gateway is nil")
	}
	ipv4sFree, ipv6sFree, ipv4sTotal, ipv6sTotal, err := countGatewayIP(gateway)
	if err != nil {
		return fmt.Errorf("failed to calculate gateway ip usage")
	}
	gateway.Status.IPUsage.IPv4Free = ipv4sFree
	gateway.Status.IPUsage.IPv6Free = ipv6sFree
	gateway.Status.IPUsage.IPv4Total = ipv4sTotal
	gateway.Status.IPUsage.IPv6Total = ipv6sTotal
	err = cli.Status().Update(ctx, gateway)
	if err != nil {
		return err
	}
	return nil
}

func deleteEgressPolicy(gateway *egress.EgressGateway, policyNs, policyName string) (bool, error) {
	if gateway == nil {
		return false, fmt.Errorf("gateway is nil")
	}

	policyFound := false

	for nodeIndex, node := range gateway.Status.NodeList {
		for eipIndex, eip := range node.Eips {
			for policyIndex, policy := range eip.Policies {
				if policy.Name == policyName && policy.Namespace == policyNs {
					gateway.Status.NodeList[nodeIndex].Eips[eipIndex].Policies = append(
						gateway.Status.NodeList[nodeIndex].Eips[eipIndex].Policies[:policyIndex],
						gateway.Status.NodeList[nodeIndex].Eips[eipIndex].Policies[policyIndex+1:]...,
					)
					// if it is the latest policy, we delete this eip
					if len(gateway.Status.NodeList[nodeIndex].Eips[eipIndex].Policies) == 0 {
						gateway.Status.NodeList[nodeIndex].Eips = append(
							gateway.Status.NodeList[nodeIndex].Eips[:eipIndex],
							gateway.Status.NodeList[nodeIndex].Eips[eipIndex+1:]...,
						)
					}
					policyFound = true
					break
				}
			}
			if policyFound {
				break
			}
		}
		if policyFound {
			break
		}
	}
	return policyFound, nil
}

func NewEgressGatewayController(mgr manager.Manager, log logr.Logger, cfg *config.Config, client client.Client) error {
	if cfg == nil {
		return fmt.Errorf("cfg can not be nil")
	}
	r := &egnReconciler{
		client: mgr.GetClient(),
		log:    log,
		config: cfg,
		cli:    client,
	}

	c, err := controller.New("egressGateway", mgr,
		controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err = c.Watch(source.Kind(mgr.GetCache(), &egress.EgressGateway{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGateway")), egressGatewayPredicate{}); err != nil {
		return fmt.Errorf("failed to watch EgressGateway: %w", err)
	}

	if err = c.Watch(source.Kind(mgr.GetCache(), &corev1.Node{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("Node")), nodePredicate{}); err != nil {
		return fmt.Errorf("failed to watch Node: %w", err)
	}

	if err = c.Watch(source.Kind(mgr.GetCache(), &egress.EgressPolicy{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressPolicy")), egressPolicyPredicate{}); err != nil {
		return fmt.Errorf("failed to watch EgressPolicy: %w", err)
	}

	if err = c.Watch(source.Kind(mgr.GetCache(), &egress.EgressClusterPolicy{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressClusterPolicy")), egressClusterPolicyPredicate{}); err != nil {
		return fmt.Errorf("failed to watch EgressClusterPolicy: %w", err)
	}

	if err = c.Watch(source.Kind(mgr.GetCache(), &egress.EgressTunnel{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressTunnel")), egressTunnelPredicate{}); err != nil {
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

func countGatewayIP(egw *egress.EgressGateway) (ipv4sFree, ipv6sFree, ipv4sTotal, ipv6sTotal int, err error) {
	ipv4s, err := ip.ConvertCidrOrIPrangeToIPs(egw.Spec.Ippools.IPv4, constant.IPv4)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	ipv6s, err := ip.ConvertCidrOrIPrangeToIPs(egw.Spec.Ippools.IPv6, constant.IPv6)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	useIpv4s := make([]net.IP, 0)
	useIpv6s := make([]net.IP, 0)
	for _, node := range egw.Status.NodeList {
		for _, eip := range node.Eips {
			if len(eip.IPv4) != 0 {
				useIpv4s = append(useIpv4s, net.ParseIP(eip.IPv4))
			}
			if len(eip.IPv6) != 0 {
				useIpv6s = append(useIpv6s, net.ParseIP(eip.IPv6))
			}
		}
	}

	ipv4sFree = len(ipv4s) - len(useIpv4s)
	ipv6sFree = len(ipv6s) - len(useIpv6s)

	ipv4sTotal, ipv6sTotal, err = len(ipv4s), len(ipv6s), nil
	return
}

// removeEgressGatewayFinalizer if the egress gateway is being deleted
func removeEgressGatewayFinalizer(egw *egress.EgressGateway) {
	if containsEgressGatewayFinalizer(egw, egressGatewayFinalizers) {
		egw.Finalizers = slice.RemoveElement(egw.Finalizers, egressGatewayFinalizers)
	}
}

func getPolicyCountByGatewayName(ctx context.Context, client client.Client, name string) (int, error) {
	var num int

	list := new(egress.EgressPolicyList)
	err := client.List(ctx, list)
	if err != nil {
		return num, err
	}
	for _, p := range list.Items {
		if p.Spec.EgressGatewayName == name {
			num++
		}
	}

	policyList := new(egress.EgressClusterPolicyList)
	err = client.List(ctx, list)
	if err != nil {
		return num, err
	}
	for _, p := range policyList.Items {
		if p.Spec.EgressGatewayName == name {
			num++
		}
	}

	return num, nil
}

func containsEgressGatewayFinalizer(gateway *egress.EgressGateway, finalizer string) bool {
	for _, f := range gateway.ObjectMeta.Finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

type egressPolicyPredicate struct{}

func (p egressPolicyPredicate) Create(_ event.CreateEvent) bool { return true }
func (p egressPolicyPredicate) Delete(_ event.DeleteEvent) bool { return true }
func (p egressPolicyPredicate) Update(updateEvent event.UpdateEvent) bool {
	oldObj, ok := updateEvent.ObjectOld.(*egress.EgressPolicy)
	if !ok {
		return false
	}
	newObj, ok := updateEvent.ObjectNew.(*egress.EgressPolicy)
	if !ok {
		return false
	}
	if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
		return true
	}
	return false
}
func (p egressPolicyPredicate) Generic(_ event.GenericEvent) bool { return true }

type egressClusterPolicyPredicate struct{}

func (p egressClusterPolicyPredicate) Create(_ event.CreateEvent) bool { return true }
func (p egressClusterPolicyPredicate) Delete(_ event.DeleteEvent) bool { return true }
func (p egressClusterPolicyPredicate) Update(updateEvent event.UpdateEvent) bool {
	oldObj, ok := updateEvent.ObjectOld.(*egress.EgressClusterPolicy)
	if !ok {
		return false
	}
	newObj, ok := updateEvent.ObjectNew.(*egress.EgressClusterPolicy)
	if !ok {
		return false
	}
	if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
		return true
	}
	return false
}
func (p egressClusterPolicyPredicate) Generic(_ event.GenericEvent) bool { return true }

type egressGatewayPredicate struct{}

func (p egressGatewayPredicate) Create(_ event.CreateEvent) bool { return true }
func (p egressGatewayPredicate) Delete(_ event.DeleteEvent) bool { return true }
func (p egressGatewayPredicate) Update(updateEvent event.UpdateEvent) bool {
	oldObj, ok := updateEvent.ObjectOld.(*egress.EgressGateway)
	if !ok {
		return false
	}
	newObj, ok := updateEvent.ObjectNew.(*egress.EgressGateway)
	if !ok {
		return false
	}
	if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
		return true
	}
	return false
}
func (p egressGatewayPredicate) Generic(_ event.GenericEvent) bool { return true }

type nodePredicate struct{}

func (p nodePredicate) Create(_ event.CreateEvent) bool { return true }
func (p nodePredicate) Delete(_ event.DeleteEvent) bool { return true }
func (p nodePredicate) Update(updateEvent event.UpdateEvent) bool {
	oldObj, ok := updateEvent.ObjectOld.(*corev1.Node)
	if !ok {
		return false
	}
	newObj, ok := updateEvent.ObjectNew.(*corev1.Node)
	if !ok {
		return false
	}
	if areMapsEqual(oldObj.Labels, newObj.Labels) {
		return false
	}
	return true
}

func areMapsEqual(mapA, mapB map[string]string) bool {
	if len(mapA) != len(mapB) {
		return false
	}
	for key, valueA := range mapA {
		if valueB, ok := mapB[key]; !ok || valueA != valueB {
			return false
		}
	}
	return true
}

func (p nodePredicate) Generic(_ event.GenericEvent) bool { return true }

type egressTunnelPredicate struct{}

func (p egressTunnelPredicate) Create(_ event.CreateEvent) bool { return true }
func (p egressTunnelPredicate) Delete(_ event.DeleteEvent) bool { return true }
func (p egressTunnelPredicate) Update(updateEvent event.UpdateEvent) bool {
	oldEgressTunnel, ok := updateEvent.ObjectOld.(*egress.EgressTunnel)
	if !ok {
		return false
	}
	newEgressTunnel, ok := updateEvent.ObjectNew.(*egress.EgressTunnel)
	if !ok {
		return false
	}
	if oldEgressTunnel.Status.Phase != newEgressTunnel.Status.Phase {
		return true
	}
	return false
}
func (p egressTunnelPredicate) Generic(_ event.GenericEvent) bool { return true }
