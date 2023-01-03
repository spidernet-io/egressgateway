// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"crypto/sha1"
	"fmt"
	"k8s.io/apimachinery/pkg/labels"
	"net"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/ipset"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

type policeReconciler struct {
	client   client.Client
	log      *zap.Logger
	cfg      *config.Config
	ipsetMap *utils.SyncMap[string, *ipset.IPSet]
	ipset    ipset.Interface
}

func (r policeReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
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
	case "EgressGatewayNode":
		return r.reconcileEGN(ctx, newReq, log)
	case "EgressGatewayPolicy":
		return r.reconcileEGP(ctx, newReq, log)
	case "Pod":
		return r.reconcilePod(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileEGN reconcile egress gateway node
// goal:
// - add/update/delete egress gateway node
//   - iptables
func (r policeReconciler) reconcileEGN(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

// reconcileEGP reconcile egress gateway policy
// add/update/delete policy
//   - ipset
//   - iptables
func (r policeReconciler) reconcileEGP(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	policy := new(egressv1.EgressGatewayPolicy)
	deleted := false
	err := r.client.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !policy.GetDeletionTimestamp().IsZero()

	setNames := GetIPSetNamesByPolicy(policy.Name)

	// reconcile delete event
	if deleted {
		log.Info("request item is deleted")
		// TODO remove iptables

		_ = setNames.Map(func(set SetName) error {
			r.removeIPSet(log, set.Name)
			return nil
		})

		return reconcile.Result{}, nil
	}

	// reconcile add or update event
	err = setNames.Map(func(set SetName) error {
		return r.createIPSet(log, set)
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	toAddList := make(map[string][]string, 0)
	toDelList := make(map[string][]string, 0)

	// calculate src ip list
	podIPv4List, podIPv6List, err := r.getPodIPsByLabelSelector(ctx, policy.Spec.AppliedTo.PodSelector)
	if err != nil {
		return reconcile.Result{}, err
	}

	// calculate dst ip list
	dstIPv4List, dstIPv6List, err := r.getDstCIDR(policy.Spec.DestSubnet)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = setNames.Map(func(set SetName) error {
		oldIPList, err := r.ipset.ListEntries(set.Name)
		if err != nil {
			return err
		}
		switch set.Kind {
		case IPSrc:
			if set.Stack == IPv4 && r.cfg.FileConfig.EnableIPv4 {
				toAddList[set.Name], toDelList[set.Name] = findDiff(oldIPList, podIPv4List)
			} else if r.cfg.FileConfig.EnableIPv6 {
				toAddList[set.Name], toDelList[set.Name] = findDiff(oldIPList, podIPv6List)
			}
		case IPDst:
			if set.Stack == IPv4 && r.cfg.FileConfig.EnableIPv4 {
				toAddList[set.Name], toDelList[set.Name] = findDiff(oldIPList, dstIPv4List)
			} else if r.cfg.FileConfig.EnableIPv6 {
				toAddList[set.Name], toDelList[set.Name] = findDiff(oldIPList, dstIPv6List)
			}
		}
		return nil
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	for set, ips := range toAddList {
		log.Sugar().Debugf("add IPSet entries: %v", ips)
		ipSet, ok := r.ipsetMap.Load(set)
		if !ok {
			continue
		}
		for _, ip := range ips {
			err := r.ipset.AddEntry(ip, ipSet, false)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// TODO update iptables

	for name, ips := range toDelList {
		for _, ip := range ips {
			err := r.ipset.DelEntry(ip, name)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

func findDiff(oldList, newList []string) (toAdd, toDel []string) {
	oldCopy := make([]string, len(oldList))
	copy(oldCopy, oldList)
	newCopy := make([]string, len(newList))
	copy(newCopy, newList)

	oldMap := make(map[string]bool)
	for _, s := range oldCopy {
		oldMap[s] = true
	}
	newMap := make(map[string]bool)
	for _, s := range newCopy {
		newMap[s] = true
	}

	toAdd = make([]string, 0)
	toDel = make([]string, 0)
	for _, s := range newCopy {
		if !oldMap[s] {
			toAdd = append(toAdd, s)
		}
	}
	for _, s := range oldCopy {
		if !newMap[s] {
			toDel = append(toDel, s)
		}
	}
	return toAdd, toDel
}

func findElements(include bool, parent []string, sub []string) []string {
	parentCopy := make([]string, len(parent))
	parentMap := make(map[string]struct{})
	for _, s := range parentCopy {
		parentMap[s] = struct{}{}
	}
	res := make([]string, 0)
	for _, s := range sub {
		if _, ok := parentMap[s]; ok == include {
			res = append(res, s)
		}
	}
	return res
}

func (r policeReconciler) getDstCIDR(list []string) ([]string, []string, error) {
	ipv4List := make([]string, 0)
	ipv6List := make([]string, 0)

	for _, item := range list {
		ip, ipNet, err := net.ParseCIDR(item)
		if err != nil {
			return nil, nil, err
		}
		if ip == nil {
			continue
		}
		if ip4 := ip.To4(); ip4 != nil {
			//nolint
			ipv4List = append(ipv4List, ipNet.String())
		} else {
			//nolint
			ipv6List = append(ipv6List, ipNet.String())
		}
	}
	return nil, nil, nil
}

func (r policeReconciler) getPodIPsByLabelSelector(ctx context.Context, ls *metav1.LabelSelector) ([]string, []string, error) {
	podList := &corev1.PodList{}
	selPods, err := metav1.LabelSelectorAsSelector(ls)
	if err != nil {
		return nil, nil, err
	}
	err = r.client.List(ctx, podList, &client.ListOptions{
		LabelSelector: selPods,
	})
	if err != nil {
		return nil, nil, err
	}

	ipv4List, ipv6List := getPodIPsByPodList(podList)
	return ipv4List, ipv6List, nil
}

func getPodIPsByPodList(podList *corev1.PodList) ([]string, []string) {
	ipv4List := make([]string, 0)
	ipv6List := make([]string, 0)

	for _, pod := range podList.Items {
		ipv4ListTmp, ipv6ListTmp := getPodIPsBy(pod)
		ipv4List = append(ipv4List, ipv4ListTmp...)
		ipv6List = append(ipv6List, ipv6ListTmp...)
	}
	return ipv4List, ipv6List
}

func getPodIPsBy(pod corev1.Pod) ([]string, []string) {
	ipv4List := make([]string, 0)
	ipv6List := make([]string, 0)
	for _, item := range pod.Status.PodIPs {
		ip := net.ParseIP(item.IP)
		if ip == nil {
			continue
		}
		if ip4 := ip.To4(); ip4 != nil {
			ipv4List = append(ipv4List, item.IP)
		} else {
			ipv6List = append(ipv6List, item.IP)
		}
	}
	return ipv4List, ipv6List
}

func (r policeReconciler) removeIPSet(log *zap.Logger, name string) {
	_, ok := r.ipsetMap.Load(name)
	if ok {
		err := r.ipset.DestroySet(name)
		if err != nil {
			log.Sugar().Warnf("delete IPSet with error: %v", err)
		}
		r.ipsetMap.Delete(name)
	}
}

func (r policeReconciler) createIPSet(log *zap.Logger, set SetName) error {
	_, exits := r.ipsetMap.Load(set.Name)
	if !exits {
		if set.Stack == IPv4 && !r.cfg.FileConfig.EnableIPv4 {
			return nil
		}
		if set.Stack == IPv6 && !r.cfg.FileConfig.EnableIPv6 {
			return nil
		}

		log.Sugar().Debug("add src ipset")
		ipSet := &ipset.IPSet{
			Name:       set.Name,
			SetType:    ipset.HashNet,
			HashFamily: set.Stack.HashFamily(),
			Comment:    "",
		}
		err := r.ipset.CreateSet(ipSet, false)
		if err != nil {
			log.Sugar().Errorf("add src ipset with error: %v", err)
			return err
		}
		r.ipsetMap.Store(set.Name, ipSet)
	}
	return nil
}

// reconcilePod reconcile pod
// add/update/remove pod
// - update ipset entry
func (r policeReconciler) reconcilePod(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	pod := new(corev1.Pod)
	deleted := false
	err := r.client.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	if deleted {
		log.Sugar().Debug("watch pod deleted event, skip")
		return reconcile.Result{}, nil
	}
	deleted = deleted || !pod.GetDeletionTimestamp().IsZero()

	policyList := new(egressv1.EgressGatewayPolicyList)
	err = r.client.List(ctx, policyList)
	if err != nil {
		return reconcile.Result{}, err
	}

	ipv4List, ipv6List := getPodIPsBy(*pod)

	for _, item := range policyList.Items {
		podLabelSelector, err := metav1.LabelSelectorAsSelector(item.Spec.AppliedTo.PodSelector)
		if err != nil {
			log.Sugar().Error(err)
			continue
		}
		if !podLabelSelector.Matches(labels.Set(pod.Labels)) {
			log.Sugar().Debug("pod not matching egn(%s)", item.Name)
			continue
		}

		toAddList := make(map[string][]string, 0)
		toDelList := make(map[string][]string, 0)

		setNames := GetIPSetNamesByPolicy(item.Name)
		err = setNames.Map(func(set SetName) error {
			oldIPList, err := r.ipset.ListEntries(set.Name)
			if err != nil {
				return err
			}

			if set.Kind == IPSrc {
				if set.Stack == IPv4 {
					if deleted {
						toDelList[set.Name] = findElements(true, oldIPList, ipv4List)
					} else {
						toAddList[set.Name] = findElements(false, oldIPList, ipv4List)
					}
				} else {
					if deleted {
						toDelList[set.Name] = findElements(true, oldIPList, ipv6List)
					} else {
						toAddList[set.Name] = findElements(false, oldIPList, ipv6List)
					}
				}
			}
			return nil
		})
		if err != nil {
			return reconcile.Result{}, err
		}

		for set, ips := range toAddList {
			for _, ip := range ips {
				ipSet, ok := r.ipsetMap.Load(set)
				if ok {
					err := r.ipset.AddEntry(ip, ipSet, false)
					if err != nil {
						return reconcile.Result{}, err
					}
				}
			}
		}

		for name, ips := range toDelList {
			for _, ip := range ips {
				err := r.ipset.DelEntry(ip, name)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	return reconcile.Result{}, nil
}

func newPolicyController(mgr manager.Manager, log *zap.Logger, cfg *config.Config) error {
	e := exec.New()
	r := &policeReconciler{
		client:   mgr.GetClient(),
		ipsetMap: utils.NewSyncMap[string, *ipset.IPSet](),
		log:      log,
		ipset:    ipset.New(e),
		cfg:      cfg,
	}

	c, err := controller.New("policy", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressGatewayNode{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGatewayNode"))); err != nil {
		return fmt.Errorf("failed to watch EgressGatewayNode: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressGatewayPolicy{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGatewayPolicy"))); err != nil {
		return fmt.Errorf("failed to watch EgressGatewayPolicy: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &corev1.Pod{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("Pod"))); err != nil {
		return fmt.Errorf("failed to watch Pod: %w", err)
	}

	return nil
}

func GetIPSetNamesByPolicy(name string) SetNames {
	return []SetName{
		{Name: formatIPSetName("egress-src-v4-", name), Stack: IPv4, Kind: IPSrc},
		{Name: formatIPSetName("egress-src-v6-", name), Stack: IPv6, Kind: IPSrc},
		{Name: formatIPSetName("egress-dst-v4-", name), Stack: IPv4, Kind: IPDst},
		{Name: formatIPSetName("egress-dst-v6-", name), Stack: IPv6, Kind: IPDst},
	}
}

type SetNames []SetName

type SetName struct {
	Name  string
	Stack IPStack
	Kind  IPKind
}

func (m SetNames) Map(f func(name SetName) error) error {
	for _, item := range m {
		err := f(item)
		if err != nil {
			return err
		}
	}
	return nil
}

func formatIPSetName(prefix, name string) string {
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(name)))
	i := 31 - len(prefix)
	return prefix + hash[:i]
}

type IPKind int

const (
	IPSrc IPKind = iota
	IPDst
)

type IPStack int

const (
	IPv4 IPStack = iota
	IPv6
)

func (stack IPStack) HashFamily() string {
	switch stack {
	case 0:
		return "IPv4"
	case 1:
		return "IPv6"
	default:
		return ""
	}
}
