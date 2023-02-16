// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/ipset"
	"github.com/spidernet-io/egressgateway/pkg/iptables"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

type policeReconciler struct {
	client   client.Client
	log      *zap.Logger
	cfg      *config.Config
	ipsetMap *utils.SyncMap[string, *ipset.IPSet]
	ipset    ipset.Interface
	doOnce   sync.Once

	ruleV4Map    *utils.SyncMap[string, iptables.Rule]
	ruleV6Map    *utils.SyncMap[string, iptables.Rule]
	mangleTables []*iptables.Table
	filterTables []*iptables.Table
	natTables    []*iptables.Table
}

func (r *policeReconciler) initApplyPolicy() error {
	policyList := new(egressv1.EgressGatewayPolicyList)
	err := r.client.List(context.Background(), policyList)
	if err != nil {
		return err
	}

	gatewayList := new(egressv1.EgressGatewayList)
	err = r.client.List(context.Background(), gatewayList)
	if err != nil {
		return err
	}
	if len(gatewayList.Items) == 0 {
		return nil
	}
	isGatewayNode := false
	hasGateway := false
	for _, node := range gatewayList.Items[0].Status.NodeList {
		if node.Name == r.cfg.NodeName {
			isGatewayNode = true
		}
		if node.Active {
			hasGateway = true
		}
	}

	for _, table := range r.natTables {
		chainMapRules := buildNatStaticRule()
		for chain, rules := range chainMapRules {
			table.InsertOrAppendRules(chain, rules)
		}
	}

	for _, table := range r.filterTables {
		chainMapRules := buildFilterStaticRule()
		for chain, rules := range chainMapRules {
			table.InsertOrAppendRules(chain, rules)
		}
	}

	for _, table := range r.mangleTables {
		table.UpdateChain(&iptables.Chain{
			Name: "EGRESSGATEWAY-MARK-REQUEST",
		})
		chainMapRules := buildMangleStaticRule(isGatewayNode, hasGateway)
		for chain, rules := range chainMapRules {
			table.InsertOrAppendRules(chain, rules)
		}
	}

	for _, policy := range policyList.Items {
		log := r.log.With(
			zap.String("namespacedName", policy.Name),
			zap.String("kind", "policy"),
		)
		if policy.DeletionTimestamp.IsZero() {
			err := r.addOrUpdatePolicy(context.Background(), true, &policy, log)
			if err != nil {
				return err
			}
		}
	}

	allTables := append(r.natTables, r.filterTables...)
	allTables = append(allTables, r.mangleTables...)
	for _, table := range allTables {
		_, err = table.Apply()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *policeReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	r.doOnce.Do(func() {
		r.log.Sugar().Info("first reconcile of policy controller, init apply policy")
	redo:
		err := r.initApplyPolicy()
		if err != nil {
			r.log.Sugar().Error("first reconcile of policy controller, init apply policy, with error:", err)
			goto redo
		}
	})
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
	case "EgressGatewayPolicy":
		return r.reconcileEGP(ctx, newReq, log)
	case "Pod":
		return r.reconcilePod(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileEG reconcile egress gateway
// - add/update/delete egress gateway
//   - iptables
func (r *policeReconciler) reconcileEG(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	gateway := new(egressv1.EgressGateway)
	deleted := false
	err := r.client.Get(ctx, req.NamespacedName, gateway)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !gateway.GetDeletionTimestamp().IsZero()

	if deleted {
		log.Sugar().Info("request item deleted, rebuild the iptables rules to overwrite the system rules.")
		for _, table := range r.mangleTables {
			chainMapRules := buildMangleStaticRule(false, false)

			for chain, rules := range chainMapRules {
				table.InsertOrAppendRules(chain, rules)
				_, err := table.Apply()
				if err != nil {
					return reconcile.Result{Requeue: true}, err
				}
			}
		}
	}

	isGatewayNode := false
	hasGateway := false
	for _, node := range gateway.Status.NodeList {
		if node.Name == r.cfg.EnvConfig.NodeName {
			isGatewayNode = true
		}
		if node.Active {
			hasGateway = true
		}
	}

	for _, table := range r.mangleTables {
		log.Sugar().Debug("building a static rule for the mangle table",
			zap.Bool("isGatewayNode", isGatewayNode),
			zap.Bool("hasGateway", hasGateway))

		chainMapRules := buildMangleStaticRule(isGatewayNode, hasGateway)

		for chain, rules := range chainMapRules {
			log.Debug("insert or append rules", zap.String("chain", chain))
			table.InsertOrAppendRules(chain, rules)
			_, err := table.Apply()
			if err != nil {
				log.Error("failed to apply iptables", zap.Error(err), zap.String("chain", chain))
				return reconcile.Result{Requeue: true}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func buildNatStaticRule() map[string][]iptables.Rule {
	res := map[string][]iptables.Rule{}
	return res
}

func buildFilterStaticRule() map[string][]iptables.Rule {
	res := map[string][]iptables.Rule{
		"FORWARD": {
			{
				Match:  iptables.MatchCriteria{}.MarkMatchesWithMask(0x11000000, 0xffffffff),
				Action: iptables.AcceptAction{},
			},
		},
		"OUTPUT": {
			{
				Match:  iptables.MatchCriteria{}.MarkMatchesWithMask(0x11000000, 0xffffffff),
				Action: iptables.AcceptAction{},
			},
		},
	}
	return res
}

func buildMangleStaticRule(isGatewayNode bool, hasGateway bool) map[string][]iptables.Rule {
	res := map[string][]iptables.Rule{
		"FORWARD": {
			{
				Match:  iptables.MatchCriteria{}.MarkMatchesWithMask(0x11000000, 0xffffffff),
				Action: iptables.SetMarkAction{Mark: 0x12000000},
			},
		},
		"POSTROUTING": {
			{
				Match:  iptables.MatchCriteria{}.MarkMatchesWithMask(0x11000000, 0xffffffff),
				Action: iptables.AcceptAction{},
			},
		},
	}
	if !isGatewayNode && hasGateway {
		res["PREROUTING"] = []iptables.Rule{
			{
				Match: iptables.MatchCriteria{},
				Action: iptables.JumpAction{
					Target: "EGRESSGATEWAY-MARK-REQUEST",
				},
			},
		}
	}
	return res
}

// reconcileEGP reconcile egress gateway policy
// add/update/delete policy
//   - ipset
//   - iptables
func (r *policeReconciler) reconcileEGP(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
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

	// reconcile delete event
	if deleted {
		setNames := GetIPSetNamesByPolicy(req.Name, true, true)
		log.Info("request item deleted, delete related policies")

		for _, table := range r.mangleTables {
			rules, changed := r.removePolicyRule(req.Name, table.IPVersion)
			if changed {
				table.UpdateChain(&iptables.Chain{
					Name:  "EGRESSGATEWAY-MARK-REQUEST",
					Rules: rules,
				})
				_, err := table.Apply()
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}

		_ = setNames.Map(func(set SetName) error {
			r.removeIPSet(log, set.Name)
			return nil
		})

		return reconcile.Result{}, nil
	}

	err = r.addOrUpdatePolicy(ctx, false, policy, log)
	if err != nil {
		return reconcile.Result{
			Requeue: true,
		}, err
	}
	return reconcile.Result{}, nil
}

// addOrUpdatePolicy reconcile add or update egress policy
func (r *policeReconciler) addOrUpdatePolicy(ctx context.Context, firstInit bool, policy *egressv1.EgressGatewayPolicy, log *zap.Logger) error {
	setNames := GetIPSetNamesByPolicy(policy.Name, r.cfg.FileConfig.EnableIPv4, r.cfg.FileConfig.EnableIPv6)
	err := setNames.Map(func(set SetName) error {
		log.Debug("check ipset", zap.String("ipset", set.Name))
		return r.createIPSet(log, set)
	})
	if err != nil {
		return err
	}

	toAddList := make(map[string][]string, 0)
	toDelList := make(map[string][]string, 0)

	// calculate src ip list
	podIPv4List, podIPv6List, err := r.getPodIPsByLabelSelector(ctx, policy.Spec.AppliedTo.PodSelector)
	if err != nil {
		return err
	}

	// calculate dst ip list
	dstIPv4List, dstIPv6List, err := r.getDstCIDR(policy.Spec.DestSubnet)
	if err != nil {
		return err
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
		return err
	}

	for set, ips := range toAddList {
		log.Sugar().Debugf("add IPSet entries: %v", ips)
		ipSet, ok := r.ipsetMap.Load(set)
		if !ok {
			continue
		}
		for _, ip := range ips {
			err := r.ipset.AddEntry(ip, ipSet, true)
			if err != nil && err != ipset.ErrAlreadyAddedEntry {
				return err
			}
		}
	}

	for _, table := range r.mangleTables {
		rules, changed := r.updatePolicyRule(policy.Name, table.IPVersion)
		if changed {
			table.UpdateChain(&iptables.Chain{
				Name:  "EGRESSGATEWAY-MARK-REQUEST",
				Rules: rules,
			})
			if !firstInit {
				_, err := table.Apply()
				if err != nil {
					return err
				}
			}
		}
	}

	for name, ips := range toDelList {
		for _, ip := range ips {
			err := r.ipset.DelEntry(ip, name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *policeReconciler) removePolicyRule(policyName string, version uint8) ([]iptables.Rule, bool) {
	changed := false
	var ruleMap *utils.SyncMap[string, iptables.Rule]
	switch version {
	case 4:
		ruleMap = r.ruleV4Map
	case 6:
		ruleMap = r.ruleV6Map
	default:
		panic("not supported ip version")
	}
	if _, ok := ruleMap.Load(policyName); ok {
		ruleMap.Delete(policyName)
		changed = true
	}
	if !changed {
		return make([]iptables.Rule, 0), changed
	}
	return buildRuleList(ruleMap), changed
}

// updatePolicyRule make policy to rule
func (r *policeReconciler) updatePolicyRule(policyName string, version uint8) ([]iptables.Rule, bool) {
	changed := false
	tmp := ""
	var ruleMap *utils.SyncMap[string, iptables.Rule]
	switch version {
	case 4:
		ruleMap = r.ruleV4Map
		if _, ok := ruleMap.Load(policyName); ok {
			break
		}
		tmp = "v4-"
		changed = true
	case 6:
		ruleMap = r.ruleV6Map
		if _, ok := ruleMap.Load(policyName); ok {
			break
		}
		tmp = "v6-"
		changed = true
	default:
		panic("not supported ip version")
	}
	if !changed {
		return make([]iptables.Rule, 0), changed
	}
	rule := iptables.Rule{
		Match: iptables.MatchCriteria{}.
			SourceIPSet(formatIPSetName("egress-src-"+tmp, policyName)).
			DestIPSet(formatIPSetName("egress-dst-"+tmp, policyName)),
		Action:  iptables.SetMaskedMarkAction{Mark: 0x11000000, Mask: 0xffffffff},
		Comment: []string{},
	}
	ruleMap.Store(policyName, rule)
	return buildRuleList(ruleMap), changed
}

func buildRuleList(ruleMap *utils.SyncMap[string, iptables.Rule]) []iptables.Rule {
	list := make([]iptables.Rule, 0)
	ruleMap.Range(func(key string, val iptables.Rule) bool {
		list = append(list, val)
		return false
	})
	return list
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

func (r *policeReconciler) getDstCIDR(list []string) ([]string, []string, error) {
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
			ipv4List = append(ipv4List, ipNet.String())
		} else {
			ipv6List = append(ipv6List, ipNet.String())
		}
	}
	return ipv4List, ipv6List, nil
}

func (r *policeReconciler) getPodIPsByLabelSelector(ctx context.Context, ls *metav1.LabelSelector) ([]string, []string, error) {
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
		if pod.DeletionTimestamp.IsZero() {
			ipv4ListTmp, ipv6ListTmp := getPodIPsBy(pod)
			ipv4List = append(ipv4List, ipv4ListTmp...)
			ipv6List = append(ipv6List, ipv6ListTmp...)
		}
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

func (r *policeReconciler) removeIPSet(log *zap.Logger, name string) {
	_, ok := r.ipsetMap.Load(name)
	if ok {
		err := r.ipset.DestroySet(name)
		if err != nil {
			log.Sugar().Warnf("delete IPSet with error: %v", err)
		}
		r.ipsetMap.Delete(name)
	}
}

func (r *policeReconciler) createIPSet(log *zap.Logger, set SetName) error {
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
		err := r.ipset.CreateSet(ipSet, true)
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
func (r *policeReconciler) reconcilePod(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
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
			log.Sugar().Debugf("pod not matching egn(%s)", item.Name)
			continue
		}

		toAddList := make(map[string][]string, 0)
		toDelList := make(map[string][]string, 0)

		setNames := GetIPSetNamesByPolicy(item.Name, r.cfg.FileConfig.EnableIPv4, r.cfg.FileConfig.EnableIPv6)
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
					err := r.ipset.AddEntry(ip, ipSet, true)
					if err != nil && err != ipset.ErrAlreadyAddedEntry {
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
	iptablesCfg := cfg.FileConfig.IPTables
	opt := iptables.Options{
		HistoricChainPrefixes:    []string{"egw"},
		BackendMode:              cfg.FileConfig.IPTables.BackendMode,
		InsertMode:               "insert",
		RefreshInterval:          time.Second * time.Duration(iptablesCfg.RefreshIntervalSecond),
		LockTimeout:              time.Second * time.Duration(iptablesCfg.LockTimeoutSecond),
		LockProbeInterval:        time.Millisecond * time.Duration(iptablesCfg.LockProbeIntervalMillis),
		InitialPostWriteInterval: time.Second * time.Duration(iptablesCfg.InitialPostWriteIntervalSecond),
		RestoreSupportsLock:      iptablesCfg.RestoreSupportsLock,
	}
	var lock sync.Locker
	if cfg.FileConfig.IPTables.RestoreSupportsLock {
		log.Info("iptables-restore has built-in lock implementation")
		lock = iptables.DummyLock{}
	} else {
		log.Info("iptables-restore use shared lock")
		lock = iptables.NewSharedLock(iptablesCfg.LockFilePath, opt.LockTimeout, opt.LockProbeInterval)
	}
	opt.XTablesLock = lock

	mangleTables := make([]*iptables.Table, 0)
	filterTables := make([]*iptables.Table, 0)
	natTables := make([]*iptables.Table, 0)
	if cfg.FileConfig.EnableIPv4 {
		mangleTable, err := iptables.NewTable("mangle", 4, "egw:", opt, log)
		if err != nil {
			return err
		}
		mangleTables = append(mangleTables, mangleTable)

		natTable, err := iptables.NewTable("nat", 4, "egw:", opt, log)
		if err != nil {
			return err
		}
		natTables = append(natTables, natTable)

		filterTable, err := iptables.NewTable("filter", 4, "egw:", opt, log)
		if err != nil {
			return err
		}
		filterTables = append(filterTables, filterTable)
	}
	if cfg.FileConfig.EnableIPv6 {
		mangle, err := iptables.NewTable("mangle", 6, "egw:-", opt, log)
		if err != nil {
			return err
		}
		mangleTables = append(mangleTables, mangle)
		nat, err := iptables.NewTable("nat", 6, "egw:", opt, log)
		if err != nil {
			return err
		}
		natTables = append(natTables, nat)
		filter, err := iptables.NewTable("filter", 6, "egw:", opt, log)
		if err != nil {
			return err
		}
		filterTables = append(filterTables, filter)
	}

	e := exec.New()
	r := &policeReconciler{
		client:       mgr.GetClient(),
		ipsetMap:     utils.NewSyncMap[string, *ipset.IPSet](),
		log:          log,
		ipset:        ipset.New(e),
		cfg:          cfg,
		mangleTables: mangleTables,
		filterTables: filterTables,
		natTables:    natTables,
		ruleV4Map:    utils.NewSyncMap[string, iptables.Rule](),
		ruleV6Map:    utils.NewSyncMap[string, iptables.Rule](),
	}

	c, err := controller.New("policy", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressGateway{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGateway"))); err != nil {
		return fmt.Errorf("failed to watch EgressGateway: %w", err)
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

func GetIPSetNamesByPolicy(name string, enableIPv4, enableIPv6 bool) SetNames {
	res := make([]SetName, 0)
	if enableIPv4 {
		res = append(res, []SetName{
			{Name: formatIPSetName("egress-src-v4-", name), Stack: IPv4, Kind: IPSrc},
			{Name: formatIPSetName("egress-dst-v4-", name), Stack: IPv4, Kind: IPDst},
		}...)
	}
	if enableIPv6 {
		res = append(res, []SetName{
			{Name: formatIPSetName("egress-src-v6-", name), Stack: IPv6, Kind: IPSrc},
			{Name: formatIPSetName("egress-dst-v6-", name), Stack: IPv6, Kind: IPDst},
		}...)
	}
	return res
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
		return "inet"
	case 1:
		return "inet6"
	default:
		return ""
	}
}
