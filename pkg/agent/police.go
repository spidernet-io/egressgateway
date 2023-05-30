// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
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
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
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
	// list egress gateway
	// list policy/cluster-policy
	// list egress endpoint slices/egress cluster policies
	// build ipset
	// build route table rule
	// build iptables
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
	case "EgressPolicy":
		return r.reconcileEGP(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileEG reconcile egress gateway
// - add/update/delete egress gateway
//   - iptables
func (r *policeReconciler) reconcileEG(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func buildNatStaticRule() map[string][]iptables.Rule {
	res := map[string][]iptables.Rule{
		"POSTROUTING": {},
	}
	return res
}

func buildFilterStaticRule() map[string][]iptables.Rule {
	res := map[string][]iptables.Rule{}
	return res
}

func buildMangleStaticRule(isGatewayNode bool, hasGateway bool) map[string][]iptables.Rule {
	res := map[string][]iptables.Rule{
		"FORWARD":     {},
		"POSTROUTING": {},
		"PREROUTING":  {},
	}
	return res
}

// reconcileEGP reconcile egress policy
// add/update/delete policy
//   - ipset
//   - iptables
func (r *policeReconciler) reconcileEGP(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

// addOrUpdatePolicy reconcile add or update egress policy
func (r *policeReconciler) addOrUpdatePolicy(ctx context.Context, firstInit bool, policy *egressv1.EgressPolicy, log *zap.Logger) error {
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
	matchCriteria := iptables.MatchCriteria{}.
		SourceIPSet(formatIPSetName("egress-src-"+tmp, policyName)).
		DestIPSet(formatIPSetName("egress-dst-"+tmp, policyName)).
		CTDirectionOriginal(iptables.DirectionOriginal)
	action := iptables.SetMaskedMarkAction{Mark: 0x11000000, Mask: 0xffffffff}
	rule := iptables.Rule{Match: matchCriteria, Action: action, Comment: []string{}}
	ruleMap.Store(policyName, rule)
	return buildRuleList(ruleMap), changed
}

func buildRuleList(ruleMap *utils.SyncMap[string, iptables.Rule]) []iptables.Rule {
	list := make([]iptables.Rule, 0)
	ruleMap.Range(func(key string, val iptables.Rule) bool {
		list = append(list, val)
		return true
	})
	return list
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

func (r *policeReconciler) removeIPSet(log *zap.Logger, name string) {
	_, ok := r.ipsetMap.Load(name)
	if ok {
		err := r.ipset.DestroySet(name)
		if err != nil {
			log.Warn("failed to delete ipset", zap.String("ipset", name), zap.Error(err))
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

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressPolicy{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressPolicy"))); err != nil {
		return fmt.Errorf("failed to watch EgressPolicy: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressClusterPolicy{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressClusterPolicy"))); err != nil {
		return fmt.Errorf("failed to watch EgressClusterPolicy: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressEndpointSlice{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressEndpointSlice"))); err != nil {
		return fmt.Errorf("failed to watch EgressEndpointSlice: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressClusterEndpointSlice{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressClusterEndpointSlice"))); err != nil {
		return fmt.Errorf("failed to watch EgressClusterEndpointSlice: %w", err)
	}

	return nil
}

func buildIPSetNamesByPolicy(name string, enableIPv4, enableIPv6 bool) SetNames {
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
