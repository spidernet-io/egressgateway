// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/ipset"
	"github.com/spidernet-io/egressgateway/pkg/iptables"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

const (
	EgressClusterCIDRIPv4 = "egress-cluster-cidr-ipv4"
	EgressClusterCIDRIPv6 = "egress-cluster-cidr-ipv6"
)

type policeReconciler struct {
	client   client.Client
	log      logr.Logger
	cfg      *config.Config
	ipsetMap *utils.SyncMap[string, *ipset.IPSet]
	ipset    ipset.Interface
	doOnce   sync.Once

	ruleV4Map     *utils.SyncMap[string, iptables.Rule]
	ruleV6Map     *utils.SyncMap[string, iptables.Rule]
	mangleTables  []*iptables.Table
	filterTables  []*iptables.Table
	natTables     []*iptables.Table
	policyMapNode *utils.SyncMap[egressv1.Policy, string]
}

func (r *policeReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	r.doOnce.Do(func() {
		r.log.Info("starting first reconciliation of policy controller")
	redo:
		err := r.initApplyPolicy()
		if err != nil {
			r.log.Error(err, "init policy")
			goto redo
		}
	})
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		return reconcile.Result{}, err
	}
	log := r.log.WithValues("kind", kind)
	var res reconcile.Result
	switch kind {
	case "EgressGateway":
		res, err = r.reconcileGateway(ctx, newReq, log)
	case "EgressClusterPolicy":
		res, err = r.reconcileClusterPolicy(ctx, newReq, log)
	case "EgressPolicy":
		res, err = r.reconcilePolicy(ctx, newReq, log)
	case "EgressClusterInfo":
		res, err = r.reconcileClusterInfo(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
	return res, err
}

type PolicyCommon struct {
	NodeName   string
	DestSubnet []string
	IP         IP
}

type IP struct {
	V4 string
	V6 string
}

// initApplyPolicy init applies the given policy
// list egress gateway
// list policy/cluster-policy
// range policy, list egress endpoint slices/egress cluster policies
// build ipset
// build route table rule
// build iptables
func (r *policeReconciler) initApplyPolicy() error {
	r.log.Info("apply policy")
	ctx := context.Background()

	gateways := new(egressv1.EgressGatewayList)
	err := r.client.List(ctx, gateways)
	if err != nil {
		return fmt.Errorf("failed to list gateway: %v", err)
	}

	if len(gateways.Items) == 0 {
		return nil
	}

	err = r.ensureClusterInfoIPSet()
	if err != nil {
		return fmt.Errorf("ensure cluster info ipset with error: %v", err)
	}

	unSnatPolicies := make(map[egressv1.Policy]*PolicyCommon)
	snatPolicies := make(map[egressv1.Policy]*PolicyCommon)
	for _, item := range gateways.Items {
		for _, list := range item.Status.NodeList {
			if list.Name == r.cfg.NodeName {
				for _, eip := range list.Eips {
					for _, policy := range eip.Policies {
						snatPolicies[policy] = &PolicyCommon{
							NodeName: list.Name,
							IP:       IP{V4: eip.IPv4, V6: eip.IPv6},
						}
					}
				}
			} else {
				for _, eip := range list.Eips {
					for _, policy := range eip.Policies {
						unSnatPolicies[policy] = &PolicyCommon{NodeName: list.Name}
					}
				}
			}
		}
	}

	for policy, val := range unSnatPolicies {
		val.DestSubnet, err = r.getPolicySubnet(policy.Namespace, policy.Name)
		if err != nil {
			return err
		}
		err := r.updatePolicyIPSet(policy.Namespace, policy.Name, false, val.DestSubnet)
		if err != nil {
			return err
		}
	}

	for policy, val := range snatPolicies {
		val.DestSubnet, err = r.getPolicySubnet(policy.Namespace, policy.Name)
		if err != nil {
			return err
		}
		err := r.updatePolicyIPSet(policy.Namespace, policy.Name, true, val.DestSubnet)
		if err != nil {
			return err
		}
	}

	baseMark, err := parseMark(r.cfg.FileConfig.Mark)
	if err != nil {
		return err
	}

	for _, table := range r.filterTables {
		chainMapRules := buildFilterStaticRule(baseMark)
		for chain, rules := range chainMapRules {
			table.InsertOrAppendRules(chain, rules)
		}
	}

	for _, table := range r.mangleTables {
		table.UpdateChain(&iptables.Chain{Name: "EGRESSGATEWAY-MARK-REQUEST"})
		chainMapRules := buildMangleStaticRule(baseMark)
		for chain, rules := range chainMapRules {
			table.InsertOrAppendRules(chain, rules)
		}
	}

	for _, table := range r.mangleTables {
		rules := make([]iptables.Rule, 0)
		for policy, val := range unSnatPolicies {
			node := new(egressv1.EgressTunnel)
			err := r.client.Get(context.Background(), types.NamespacedName{Name: val.NodeName}, node)
			if err != nil {
				r.log.Error(err, "failed to get egress node, skip building rule of policy")
				continue
			}
			policyName := policy.Name
			if policy.Namespace != "" {
				policyName = fmt.Sprintf("%s-%s", policy.Namespace, policy.Name)
			}

			mark, err := parseMark(node.Status.Mark)
			if err != nil {
				return err
			}

			isIgnoreInternalCIDR := false
			if len(val.DestSubnet) <= 0 {
				isIgnoreInternalCIDR = true
			}

			rule := r.buildPolicyRule(policyName, mark, table.IPVersion, isIgnoreInternalCIDR)
			rules = append(rules, *rule)
		}
		table.UpdateChain(&iptables.Chain{
			Name:  "EGRESSGATEWAY-MARK-REQUEST",
			Rules: rules,
		})
	}

	for _, table := range r.natTables {
		rules := make([]iptables.Rule, 0)
		for policy, val := range snatPolicies {
			policyName := policy.Name
			if policy.Namespace != "" {
				policyName = fmt.Sprintf("%s-%s", policy.Namespace, policy.Name)
			}

			isIgnoreInternalCIDR := false
			if len(val.DestSubnet) <= 0 {
				isIgnoreInternalCIDR = true
			}

			rule := buildEipRule(policyName, val.IP, table.IPVersion, isIgnoreInternalCIDR)
			if rule != nil {
				rules = append(rules, *rule)
			}
		}

		table.UpdateChain(&iptables.Chain{Name: "EGRESSGATEWAY-SNAT-EIP", Rules: rules})
		chainMapRules := buildNatStaticRule(baseMark)
		for chain, rules := range chainMapRules {
			table.InsertOrAppendRules(chain, rules)
		}
	}

	allTables := append(r.natTables, r.filterTables...)
	allTables = append(allTables, r.mangleTables...)
	for _, table := range allTables {
		_, err := table.Apply()
		if err != nil {
			return fmt.Errorf("failed to apply rule %v: %v", table.Name, err)
		}
	}

	setList, err := r.ipset.ListSets()
	if err != nil {
		r.log.Error(err, "list ipset")
		return err
	}

	for _, name := range setList {
		if !strings.HasPrefix(name, "egress-") {
			continue
		}

		if name == EgressClusterCIDRIPv6 || name == EgressClusterCIDRIPv4 {
			continue
		}

		if _, ok := r.ipsetMap.Load(name); !ok {
			err = r.ipset.DestroySet(name)
			if err != nil {
				r.log.Error(err, "clean ipset", "ipset", name)
			}
		}
	}

	return nil
}

func (r *policeReconciler) getPolicySubnet(ns, name string) ([]string, error) {
	var obj client.Object
	key := types.NamespacedName{Namespace: ns, Name: name}
	getSubnet := func(obj client.Object) []string {
		switch obj := obj.(type) {
		case *egressv1.EgressPolicy:
			return obj.Spec.DestSubnet
		case *egressv1.EgressClusterPolicy:
			return obj.Spec.DestSubnet
		default:
			return nil
		}
	}
	if ns != "" {
		obj = new(egressv1.EgressPolicy)
	} else {
		obj = new(egressv1.EgressClusterPolicy)
	}
	err := r.client.Get(context.Background(), key, obj)
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
	}
	return getSubnet(obj), nil
}

func (r *policeReconciler) updatePolicyIPSet(policyNs string, policyName string, isEipNodeSet bool, destSubnet []string) error {
	// calculate src ip list
	srcIPv4List, srcIPv6List, err := r.getPolicySrcIPs(policyNs, policyName, func(e egressv1.EgressEndpoint) bool {
		if e.Node == r.cfg.EnvConfig.NodeName {
			return true
		}
		if isEipNodeSet {
			return true
		}
		return false
	})

	if err != nil {
		return err
	}

	// calculate dst ip list
	dstIPv4List, dstIPv6List, err := r.getDstCIDR(destSubnet)
	if err != nil {
		return err
	}

	toAddList := make(map[string][]string, 0)
	toDelList := make(map[string][]string, 0)
	setNames := buildIPSetNamesByPolicy(policyNs, policyName, r.cfg.FileConfig.EnableIPv4, r.cfg.FileConfig.EnableIPv6)

	err = setNames.Map(func(set SetName) error {
		r.log.V(1).Info("check ipset", "ipset", set.Name)
		return r.createIPSet(r.log, set)
	})
	if err != nil {
		return err
	}

	err = setNames.Map(func(set SetName) error {
		oldIPList, err := r.ipset.ListEntries(set.Name)
		if err != nil {
			if strings.Contains(err.Error(), "The set with the given name does not exist") {
			} else {
				return err
			}
		}
		switch set.Kind {
		case IPSrc:
			if set.Stack == IPv4 && r.cfg.FileConfig.EnableIPv4 {
				toAddList[set.Name], toDelList[set.Name] = findDiff(oldIPList, srcIPv4List)
			} else if r.cfg.FileConfig.EnableIPv6 {
				toAddList[set.Name], toDelList[set.Name] = findDiff(oldIPList, srcIPv6List)
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
		r.log.V(1).Info("add ipset entries", "entries", ips)
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

func (r *policeReconciler) getPolicySrcIPs(policyNs, policyName string, filter func(slice egressv1.EgressEndpoint) bool) ([]string, []string, error) {
	ctx := context.Background()
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{egressv1.LabelPolicyName: policyName},
	})
	if err != nil {
		return nil, nil, err
	}
	opt := &client.ListOptions{LabelSelector: selector}

	ipv4List := make([]string, 0)
	ipv6List := make([]string, 0)

	if policyNs == "" {
		eps := new(egressv1.EgressClusterEndpointSliceList)
		err = r.client.List(ctx, eps, opt)
		if err != nil {
			return nil, nil, err
		}
		for _, ep := range eps.Items {
			if ep.DeletionTimestamp.IsZero() {
				for _, e := range ep.Endpoints {
					if filter(e) {
						ipv4List = append(ipv4List, e.IPv4...)
						ipv6List = append(ipv6List, e.IPv6...)
					}
				}
			}
		}
	} else {
		eps := new(egressv1.EgressEndpointSliceList)
		err = r.client.List(ctx, eps, opt)
		if err != nil {
			return nil, nil, err
		}
		for _, ep := range eps.Items {
			if ep.DeletionTimestamp.IsZero() {
				for _, e := range ep.Endpoints {
					if filter(e) {
						ipv4List = append(ipv4List, e.IPv4...)
						ipv6List = append(ipv6List, e.IPv6...)
					}
				}
			}
		}
	}

	return ipv4List, ipv6List, nil
}

func buildEipRule(policyName string, eip IP, version uint8, isIgnoreInternalCIDR bool) *iptables.Rule {
	if eip.V4 == "" && eip.V6 == "" {
		return nil
	}

	tmp := "v4-"
	ip := eip.V4
	ignoreName := EgressClusterCIDRIPv4
	if version == 6 {
		tmp = "v6-"
		ip = eip.V6
		ignoreName = EgressClusterCIDRIPv6
	}
	srcName := formatIPSetName("egress-src-"+tmp, policyName)
	dstName := formatIPSetName("egress-dst-"+tmp, policyName)

	matchCriteria := iptables.MatchCriteria{}.SourceIPSet(srcName).DestIPSet(dstName).
		CTDirectionOriginal(iptables.DirectionOriginal)

	if isIgnoreInternalCIDR {
		matchCriteria = iptables.MatchCriteria{}.SourceIPSet(srcName).NotDestIPSet(ignoreName).
			CTDirectionOriginal(iptables.DirectionOriginal)
	}

	action := iptables.SNATAction{ToAddr: ip}
	rule := &iptables.Rule{Match: matchCriteria, Action: action, Comment: []string{}}
	return rule
}

func parseMark(mark string) (uint32, error) {
	tmp := strings.ReplaceAll(mark, "0x", "")
	i64, err := strconv.ParseInt(tmp, 16, 32)
	if err != nil {
		return 0, err
	}
	i32 := uint32(i64)
	return i32, nil
}

func (r *policeReconciler) buildPolicyRule(policyName string, mark uint32, version uint8, isIgnoreInternalCIDR bool) *iptables.Rule {
	tmp := "v4-"
	ignoreInternalCIDRName := EgressClusterCIDRIPv4
	if version == 6 {
		tmp = "v6-"
		ignoreInternalCIDRName = EgressClusterCIDRIPv6
	}
	srcName := formatIPSetName("egress-src-"+tmp, policyName)
	dstName := formatIPSetName("egress-dst-"+tmp, policyName)

	matchCriteria := iptables.MatchCriteria{}.SourceIPSet(srcName).DestIPSet(dstName).
		CTDirectionOriginal(iptables.DirectionOriginal)

	if isIgnoreInternalCIDR {
		matchCriteria = iptables.MatchCriteria{}.SourceIPSet(srcName).NotDestIPSet(ignoreInternalCIDRName).
			CTDirectionOriginal(iptables.DirectionOriginal)
	}

	action := iptables.SetMaskedMarkAction{Mark: mark, Mask: 0xffffffff}
	rule := &iptables.Rule{Match: matchCriteria, Action: action, Comment: []string{}}
	return rule
}

func buildNatStaticRule(base uint32) map[string][]iptables.Rule {
	res := map[string][]iptables.Rule{"POSTROUTING": {
		{
			Match:  iptables.MatchCriteria{}.MarkMatchesWithMask(base, 0xffffffff),
			Action: iptables.AcceptAction{}},
		{
			Match: iptables.MatchCriteria{}, Action: iptables.JumpAction{Target: "EGRESSGATEWAY-SNAT-EIP"},
		},
	}}
	return res
}

func (r *policeReconciler) reconcileClusterInfo(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	log = log.WithValues("name", req.Name)
	log.Info("reconciling")

	info := new(egressv1.EgressClusterInfo)
	deleted := false
	err := r.client.Get(ctx, req.NamespacedName, info)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !info.GetDeletionTimestamp().IsZero()

	if deleted {
		return reconcile.Result{}, nil
	}

	ipv4 := make([]string, 0)
	ipv6 := make([]string, 0)

	addIP := func(items ...string) {
		for _, ip := range items {
			ip := net.ParseIP(ip)
			if ip.To4() != nil {
				ipv4 = append(ipv4, ip.String())
			} else if ip.To16() != nil {
				ipv6 = append(ipv6, ip.String())
			}
		}
	}

	nodesIPv4 := make([]string, 0)
	for _, pair := range info.Status.NodeIP {
		nodesIPv4 = append(nodesIPv4, pair.IPv4...)
	}
	nodesIPv6 := make([]string, 0)
	for _, pair := range info.Status.NodeIP {
		nodesIPv6 = append(nodesIPv6, pair.IPv6...)
	}
	addIP(nodesIPv4...)
	addIP(nodesIPv6...)

	addCIDR := func(items ...string) {
		for _, item := range items {
			ip, cidr, err := net.ParseCIDR(item)
			if err != nil {
				continue
			}
			if ip.To4() != nil {
				ipv4 = append(ipv4, cidr.String())
			} else if ip.To16() != nil {
				ipv6 = append(ipv6, cidr.String())
			}
		}
	}

	v4PodCidrs := make([]string, 0)
	for _, pair := range info.Status.PodCIDR {
		v4PodCidrs = append(v4PodCidrs, pair.IPv4...)
	}
	v6PodCidrs := make([]string, 0)
	for _, pair := range info.Status.PodCIDR {
		v6PodCidrs = append(v6PodCidrs, pair.IPv6...)
	}

	addCIDR(v4PodCidrs...)
	addCIDR(v6PodCidrs...)

	if info.Status.ClusterIP != nil {
		addCIDR(info.Status.ClusterIP.IPv4...)
		addCIDR(info.Status.ClusterIP.IPv6...)
	}

	addCIDR(info.Status.ExtraCidr...)

	process := func(gotList []string, expList []string, toAdd, toDel func(item string) error) error {
		got := sets.NewString(gotList...)
		exp := sets.NewString(expList...)

		for _, key := range got.List() {
			if exp.Has(key) {
				exp.Delete(key)
			}
		}
		for _, key := range exp.List() {
			if got.Has(key) {
				exp.Delete(key)
			}
		}
		for _, key := range exp.List() {
			if err := toAdd(key); err != nil {
				return err
			}
		}
		for _, key := range got.List() {
			if err := toDel(key); err != nil {
				return err
			}
		}
		return nil
	}

	err = r.ensureClusterInfoIPSet()
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("ensure cluster info ipset with error: %v", err)
	}

	gotIPv4, err := r.ipset.ListEntries(EgressClusterCIDRIPv4)
	if err != nil {
		return reconcile.Result{}, err
	}
	gotIPv6, err := r.ipset.ListEntries(EgressClusterCIDRIPv6)
	if err != nil {
		return reconcile.Result{}, err
	}

	ipSet4 := &ipset.IPSet{Name: EgressClusterCIDRIPv4, SetType: ipset.HashNet, HashFamily: "inet"}
	ipSet6 := &ipset.IPSet{Name: EgressClusterCIDRIPv6, SetType: ipset.HashNet, HashFamily: "inet6"}

	err = process(gotIPv4, ipv4, func(item string) error {
		return r.ipset.AddEntry(item, ipSet4, true)
	}, func(item string) error {
		return r.ipset.DelEntry(item, ipSet4.Name)
	})
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	err = process(gotIPv6, ipv6, func(item string) error {
		return r.ipset.AddEntry(item, ipSet6, true)
	}, func(item string) error {
		return r.ipset.DelEntry(item, ipSet6.Name)
	})
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

func (r *policeReconciler) ensureClusterInfoIPSet() error {
	if err := r.ipset.CreateSet(&ipset.IPSet{
		Name:       EgressClusterCIDRIPv4,
		SetType:    ipset.HashNet,
		HashFamily: "inet",
	}, true); err != nil {
		return err
	}
	if err := r.ipset.CreateSet(&ipset.IPSet{
		Name:       EgressClusterCIDRIPv6,
		SetType:    ipset.HashNet,
		HashFamily: "inet6",
	}, true); err != nil {
		return err
	}
	return nil
}

// reconcileGateway reconcile egress gateway
// - add/update/delete egress gateway
//   - iptables/ipset
func (r *policeReconciler) reconcileGateway(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	log.Info("reconciling")
	err := r.initApplyPolicy()
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

func buildFilterStaticRule(base uint32) map[string][]iptables.Rule {
	res := map[string][]iptables.Rule{
		"FORWARD": {{
			Match:  iptables.MatchCriteria{}.MarkMatchesWithMask(base, 0xffffffff),
			Action: iptables.AcceptAction{},
		}},
		"OUTPUT": {{
			Match:  iptables.MatchCriteria{}.MarkMatchesWithMask(base, 0xffffffff),
			Action: iptables.AcceptAction{},
		}},
	}
	return res
}

func buildMangleStaticRule(base uint32) map[string][]iptables.Rule {
	res := map[string][]iptables.Rule{
		"FORWARD": {{
			Match:  iptables.MatchCriteria{}.MarkMatchesWithMask(base, 0xff000000),
			Action: iptables.SetMaskedMarkAction{Mark: base, Mask: 0xffffffff},
		}},
		"POSTROUTING": {{
			Match:  iptables.MatchCriteria{}.MarkMatchesWithMask(base, 0xffffffff),
			Action: iptables.AcceptAction{},
		}},
		"PREROUTING": {{Match: iptables.MatchCriteria{}, Action: iptables.JumpAction{Target: "EGRESSGATEWAY-MARK-REQUEST"}}},
	}
	return res
}

// reconcilePolicy reconcile egress policy
// watch update/delete events
// - ipset
func (r *policeReconciler) reconcilePolicy(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	log = log.WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info("reconciling")

	policy := new(egressv1.EgressPolicy)
	deleted := false
	err := r.client.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !policy.GetDeletionTimestamp().IsZero()

	// delete event
	if deleted {
		setNames := buildIPSetNamesByPolicy(req.Namespace, req.Name, true, true)
		log.Info("request item deleted, delete related policies")
		_ = setNames.Map(func(set SetName) error {
			r.removeIPSet(log, set.Name)
			return nil
		})
		return reconcile.Result{}, nil
	}

	gateway := new(egressv1.EgressGateway)
	err = r.client.Get(ctx, types.NamespacedName{Name: policy.Spec.EgressGatewayName}, gateway)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		return reconcile.Result{}, nil
	}

	nodeName := ""
	for _, node := range gateway.Status.NodeList {
		for _, eip := range node.Eips {
			for _, p := range eip.Policies {
				if p.Name == policy.Name && p.Namespace == policy.Namespace {
					nodeName = node.Name
				}
			}
		}
	}

	flag := false
	if nodeName == r.cfg.EnvConfig.NodeName {
		flag = true
	}

	// update event
	err = r.updatePolicyIPSet(policy.Namespace, policy.Name, flag, policy.Spec.DestSubnet)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

// reconcileClusterPolicy reconcile egress cluster policy
// watch update/delete events
// - ipset
func (r *policeReconciler) reconcileClusterPolicy(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	log = log.WithValues("name", req.Name)
	log.Info("reconciling")

	policy := new(egressv1.EgressClusterPolicy)
	deleted := false
	err := r.client.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !policy.GetDeletionTimestamp().IsZero()

	// delete event
	if deleted {
		setNames := buildIPSetNamesByPolicy(req.Namespace, req.Name, true, true)
		log.Info("request item deleted, delete related policies")
		_ = setNames.Map(func(set SetName) error {
			r.removeIPSet(log, set.Name)
			return nil
		})
		return reconcile.Result{}, nil
	}

	gateway := new(egressv1.EgressGateway)
	err = r.client.Get(ctx, types.NamespacedName{Name: policy.Spec.EgressGatewayName}, gateway)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		return reconcile.Result{Requeue: false}, nil
	}

	nodeName := ""
	for _, node := range gateway.Status.NodeList {
		for _, eip := range node.Eips {
			for _, p := range eip.Policies {
				if p.Name == policy.Name && p.Namespace == policy.Namespace {
					nodeName = node.Name
				}
			}
		}
	}

	flag := false
	if nodeName == r.cfg.EnvConfig.NodeName {
		flag = true
	}

	// update event
	err = r.updatePolicyIPSet(policy.Namespace, policy.Name, flag, policy.Spec.DestSubnet)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

func findDiff(oldList, newList []string) (toAdd, toDel []string) {
	oldCopy := make([]string, len(oldList))
	copy(oldCopy, oldList)
	newCopy := make([]string, len(newList))
	copy(newCopy, newList)

	for i, s := range newCopy {
		// for single ip address
		if strings.HasSuffix(s, "/32") {
			newCopy[i] = strings.TrimSuffix(s, "/32")
			continue
		} else if strings.HasSuffix(s, "/128") {
			ip := net.ParseIP(strings.TrimSuffix(s, "/128"))
			if ip.To16() != nil {
				newCopy[i] = ip.To16().String()
			}
			continue
		}
		// for ip cidr
		_, cidr, _ := net.ParseCIDR(s)
		if cidr != nil {
			newCopy[i] = cidr.String()
		}
	}

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

func (r *policeReconciler) removeIPSet(log logr.Logger, name string) {
	_, ok := r.ipsetMap.Load(name)
	if ok {
		err := r.ipset.DestroySet(name)
		if err != nil {
			log.Info("failed to delete ipset", "ipset", name, "warn", err)
		}
		r.ipsetMap.Delete(name)
	}
}

func (r *policeReconciler) createIPSet(log logr.Logger, set SetName) error {
	_, exits := r.ipsetMap.Load(set.Name)
	if !exits {
		if set.Stack == IPv4 && !r.cfg.FileConfig.EnableIPv4 {
			return nil
		}
		if set.Stack == IPv6 && !r.cfg.FileConfig.EnableIPv6 {
			return nil
		}

		log.V(1).Info("add src ipset")
		ipSet := &ipset.IPSet{
			Name:       set.Name,
			SetType:    ipset.HashNet,
			HashFamily: set.Stack.HashFamily(),
			Comment:    "",
		}
		err := r.ipset.CreateSet(ipSet, true)
		if err != nil {
			log.Error(err, "add src ipset with error", err)
			return err
		}
		r.ipsetMap.Store(set.Name, ipSet)
	}
	return nil
}

func newPolicyController(mgr manager.Manager, log logr.Logger, cfg *config.Config) error {
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

	if err := c.Watch(source.Kind(mgr.GetCache(), &egressv1.EgressGateway{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGateway"))); err != nil {
		return fmt.Errorf("failed to watch EgressGateway: %w", err)
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &egressv1.EgressPolicy{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressPolicy")), policyPredicate{}); err != nil {
		return fmt.Errorf("failed to watch EgressPolicy: %w", err)
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &egressv1.EgressClusterPolicy{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressClusterPolicy")), policyPredicate{}); err != nil {
		return fmt.Errorf("failed to watch EgressClusterPolicy: %w", err)
	}

	if err := c.Watch(
		source.Kind(mgr.GetCache(), &egressv1.EgressEndpointSlice{}),
		handler.EnqueueRequestsFromMapFunc(enqueueEndpointSlice()),
		epSlicePredicate{},
	); err != nil {
		return fmt.Errorf("failed to watch EgressEndpointSlice: %w", err)
	}

	if err := c.Watch(
		source.Kind(mgr.GetCache(), &egressv1.EgressClusterEndpointSlice{}),
		handler.EnqueueRequestsFromMapFunc(enqueueEndpointSlice()),
		epSlicePredicate{},
	); err != nil {
		return fmt.Errorf("failed to watch EgressClusterEndpointSlice: %w", err)
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &egressv1.EgressClusterInfo{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressClusterInfo"))); err != nil {
		return fmt.Errorf("failed to watch EgressClusterInfo: %w", err)
	}

	return nil
}

func buildIPSetNamesByPolicy(ns, name string, enableIPv4, enableIPv6 bool) SetNames {
	if ns != "" {
		name = ns + "-" + name
	}

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

type policyPredicate struct{}

func (p policyPredicate) Create(_ event.CreateEvent) bool   { return false }
func (p policyPredicate) Delete(_ event.DeleteEvent) bool   { return true }
func (p policyPredicate) Update(_ event.UpdateEvent) bool   { return true }
func (p policyPredicate) Generic(_ event.GenericEvent) bool { return false }

type epSlicePredicate struct{}

func (p epSlicePredicate) Create(_ event.CreateEvent) bool   { return false }
func (p epSlicePredicate) Delete(_ event.DeleteEvent) bool   { return false }
func (p epSlicePredicate) Update(_ event.UpdateEvent) bool   { return true }
func (p epSlicePredicate) Generic(_ event.GenericEvent) bool { return false }

func enqueueEndpointSlice() handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		namespace := obj.GetNamespace()
		policyName, ok := obj.GetLabels()[egressv1.LabelPolicyName]
		if !ok {
			return nil
		}

		res := make([]reconcile.Request, 0)

		if namespace == "" {
			req := types.NamespacedName{
				Namespace: "EgressClusterPolicy/",
				Name:      policyName,
			}
			res = append(res, reconcile.Request{NamespacedName: req})
		} else {
			req := types.NamespacedName{
				Namespace: path.Join("EgressPolicy", namespace),
				Name:      policyName,
			}
			res = append(res, reconcile.Request{NamespacedName: req})
		}

		return res
	}
}
