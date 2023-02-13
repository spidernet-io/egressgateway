// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/agent/route"
	"github.com/spidernet-io/egressgateway/pkg/agent/vxlan"
	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

type vxlanReconciler struct {
	client client.Client
	log    *zap.Logger
	cfg    *config.Config
	vtep   *VTEP

	peerMap *utils.SyncMap[string, vxlan.Peer]

	vxlan     *vxlan.Device
	getParent func(version int) (*vxlan.Parent, error)

	ruleRoute      *route.RuleRoute
	ruleRouteCache RuleRouteCache
}

type RuleRouteCache struct {
	ipv4List []net.IP
	ipv6List []net.IP
}

type VTEP struct {
	IPv4 *net.IPNet
	IPv6 *net.IPNet
	MAC  net.HardwareAddr
}

func (r *vxlanReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
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
		return r.reconcileGateway(ctx, newReq, log)
	case "EgressNode":
		return r.reconcileEgressNode(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileGateway
func (r *vxlanReconciler) reconcileGateway(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
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
		ipv4List := make([]net.IP, 0)
		ipv6List := make([]net.IP, 0)
		r.ruleRouteCache.ipv4List = ipv4List
		r.ruleRouteCache.ipv6List = ipv6List
		err = r.ruleRoute.Ensure(r.cfg.FileConfig.VXLAN.Name, ipv4List, ipv6List)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	log.Info("calculate l3 route")
	nodeList := make([]string, 0)
	for _, node := range gateway.Status.NodeList {
		if node.Ready && node.Active {
			if node.Name == r.cfg.NodeName {
				log.Info("current node is gateway node, skip")

				r.ruleRouteCache = RuleRouteCache{
					ipv4List: make([]net.IP, 0),
					ipv6List: make([]net.IP, 0),
				}

				return reconcile.Result{}, nil
			}

			log.Sugar().Infof("active node: %s", node.Name)

			nodeList = append(nodeList, node.Name)
		}
	}

	ipv4List := make([]net.IP, 0)
	ipv6List := make([]net.IP, 0)

	for _, node := range nodeList {
		egressNode := new(egressv1.EgressNode)
		err := r.client.Get(ctx, types.NamespacedName{Name: node}, egressNode)
		if err != nil {
			if !errors.IsNotFound(err) {
				continue
			}
			return reconcile.Result{}, err
		}

		log.Sugar().Debugf("get egress node: %s", node)

		ipv4 := egressNode.Status.VxlanIPv4IP
		ipv6 := egressNode.Status.VxlanIPv6IP

		if ipv4 != "" {
			log.Sugar().Debugf("parse ip: %s", ipv4)
			ip := net.ParseIP(ipv4)
			if ip.To4() != nil {
				log.Sugar().Debugf("append ip: %s", ipv4)
				ipv4List = append(ipv4List, ip)
			}
		}

		if ipv6 != "" {
			log.Sugar().Debugf("parse ip: %s", ipv6)
			ip := net.ParseIP(ipv6)
			if ip.To16() != nil {
				log.Sugar().Debugf("append ip: %s", ipv6)
				ipv6List = append(ipv6List, ip)
			}
		}
	}

	r.ruleRouteCache.ipv4List = ipv4List
	r.ruleRouteCache.ipv6List = ipv6List

	log.Sugar().Infof("ensure route: %v, %v", ipv4List, ipv6List)
	err = r.ruleRoute.Ensure(r.cfg.FileConfig.VXLAN.Name, ipv4List, ipv6List)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// reconcileEgressNode
func (r *vxlanReconciler) reconcileEgressNode(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	node := new(egressv1.EgressNode)
	deleted := false
	err := r.client.Get(ctx, req.NamespacedName, node)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !node.GetDeletionTimestamp().IsZero()

	isPeer := true
	if req.Name == r.cfg.NodeName {
		isPeer = false
	}
	if deleted {
		if isPeer {
			r.peerMap.Delete(req.Name)
			err := r.ensureRoute()
			if err != nil {
				log.Info("delete egress node, ensure route with error", zap.Error(err))
			}
		}
		return reconcile.Result{}, nil
	}

	// early check for early return
	if isPeer {
		var ip string
		if r.version() == 4 {
			ip = node.Status.PhysicalInterfaceIPv4
		} else {
			ip = node.Status.PhysicalInterfaceIPv6
		}
		if ip == "" {
			log.Sugar().Info("peer %v, parent ip not ready, skip", node.Name)
			return reconcile.Result{}, nil
		}

		parentIP := net.ParseIP(ip)
		mac, err := net.ParseMAC(node.Status.TunnelMac)
		if err != nil {
			log.Info("mac addr not ready, skip", zap.String("mac", node.Status.TunnelMac))
			return reconcile.Result{}, nil
		}

		ipv4 := net.ParseIP(node.Status.VxlanIPv4IP).To4()
		ipv6 := net.ParseIP(node.Status.VxlanIPv6IP).To16()

		peer := vxlan.Peer{Parent: parentIP, MAC: mac}
		if ipv4 != nil {
			peer.IPv4 = &ipv4
		}
		if ipv6 != nil {
			peer.IPv6 = &ipv6
		}

		r.peerMap.Store(node.Name, peer)
		err = r.ensureRoute()
		if err != nil {
			log.Info("add egress node, ensure route with error", zap.Error(err))
		}
		return reconcile.Result{}, nil
	}

	err = r.ensureEgressNodeStatus(node)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *vxlanReconciler) ensureEgressNodeStatus(node *egressv1.EgressNode) error {
	needUpdate := false

	if r.version() == 4 && node.Status.PhysicalInterfaceIPv4 == "" {
		needUpdate = true
	}

	if r.version() == 6 && node.Status.PhysicalInterfaceIPv6 == "" {
		needUpdate = true
	}

	if needUpdate {
		err := r.updateEgressNodeStatus(node, r.version())
		if err != nil {
			return err
		}
	}

	vtep := r.parseVTEP(node.Status)
	if vtep != nil {
		r.vtep = vtep
	}
	return nil
}

func (r *vxlanReconciler) updateEgressNodeStatus(node *egressv1.EgressNode, version int) error {
	parent, err := r.getParent(version)
	if err != nil {
		return err
	}

	if node == nil {
		node = new(egressv1.EgressNode)
		ctx := context.Background()
		err = r.client.Get(ctx, types.NamespacedName{Name: r.cfg.NodeName}, node)
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil
			}
			return err
		}
	}

	needUpdate := false
	if node.Status.PhysicalInterface != parent.Name {
		needUpdate = true
		node.Status.PhysicalInterface = parent.Name
	}

	if version == 4 {
		if node.Status.PhysicalInterfaceIPv4 != parent.IP.String() {
			needUpdate = true
			node.Status.PhysicalInterfaceIPv4 = parent.IP.String()
		}
		if node.Status.PhysicalInterfaceIPv6 != "" {
			needUpdate = true
			node.Status.PhysicalInterfaceIPv6 = ""
		}
	} else {
		if node.Status.PhysicalInterfaceIPv6 != parent.IP.String() {
			needUpdate = true
			node.Status.PhysicalInterfaceIPv6 = parent.IP.String()
		}
		if node.Status.PhysicalInterfaceIPv4 != "" {
			needUpdate = true
			node.Status.PhysicalInterfaceIPv4 = ""
		}
	}

	// calculate whether the state has changed, update if the status changes.
	vtep := r.parseVTEP(node.Status)
	if vtep != nil {
		phase := "Successfully"
		if node.Status.Phase != phase {
			needUpdate = true
			node.Status.Phase = phase
		}
	}

	if needUpdate {
		ctx := context.Background()
		err = r.client.Status().Update(ctx, node)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *vxlanReconciler) parseVTEP(status egressv1.EgressNodeStatus) *VTEP {
	var ipv4 *net.IPNet
	var ipv6 *net.IPNet
	ready := true

	if status.VxlanIPv4IP == "" && r.cfg.FileConfig.EnableIPv4 {
		if status.VxlanIPv4IP == "" {
			ready = false
		} else {
			ip := net.ParseIP(status.VxlanIPv4IP)
			if ip.To4() == nil {
				ready = false
			}
			ipv4 = &net.IPNet{
				IP:   ip.To4(),
				Mask: r.cfg.FileConfig.TunnelIPv4Net.Mask,
			}
		}
	}
	if r.cfg.FileConfig.EnableIPv6 {
		if status.VxlanIPv6IP == "" {
			ready = false
		} else {
			ip := net.ParseIP(status.VxlanIPv6IP)
			if ip.To16() == nil {
				ready = false
			}
			ipv6 = &net.IPNet{
				IP:   ip.To16(),
				Mask: r.cfg.FileConfig.TunnelIPv6Net.Mask,
			}
		}
	}

	mac, err := net.ParseMAC(status.TunnelMac)
	if err != nil {
		ready = false
	}

	if !ready {
		return nil
	}
	return &VTEP{
		IPv4: ipv4,
		IPv6: ipv6,
		MAC:  mac,
	}
}

func (r *vxlanReconciler) version() int {
	version := 4
	if !r.cfg.FileConfig.EnableIPv4 && r.cfg.FileConfig.EnableIPv6 {
		version = 6
	}
	return version
}

func (r *vxlanReconciler) keepVXLAN() {
	reduce := false
	for {
		if r.vtep == nil {
			r.log.Sugar().Debugf("vtep not ready")
			time.Sleep(time.Second)
			continue
		}

		name := r.cfg.FileConfig.VXLAN.Name
		vni := r.cfg.FileConfig.VXLAN.ID
		port := r.cfg.FileConfig.VXLAN.Port
		ipv4 := r.vtep.IPv4
		ipv6 := r.vtep.IPv6
		mac := r.vtep.MAC
		disableChecksumOffload := r.cfg.FileConfig.VXLAN.DisableChecksumOffload

		err := r.updateEgressNodeStatus(nil, r.version())
		if err != nil {
			r.log.Sugar().Errorf("update EgressNode status with error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		err = r.vxlan.EnsureLink(name, vni, port, mac, 0, ipv4, ipv6, disableChecksumOffload)
		if err != nil {
			r.log.Sugar().Errorf("ensure vxlan link with error: %v", err)
			reduce = false
			time.Sleep(time.Second)
			continue
		}

		r.log.Sugar().Debugf("link ensure has completed")

		err = r.ensureRoute()
		if err != nil {
			r.log.Sugar().Errorf("ensure route with error: %v", err)
			reduce = false
			time.Sleep(time.Second)
			continue
		}

		r.log.Sugar().Debugf("route ensure has completed")

		err = r.ruleRoute.Ensure(r.cfg.FileConfig.VXLAN.Name, r.ruleRouteCache.ipv4List, r.ruleRouteCache.ipv6List)
		if err != nil {
			r.log.Sugar().Errorf("ensure vxlan link with error: %v", err)
			reduce = false
			time.Sleep(time.Second)
			continue
		}

		r.log.Sugar().Debugf("route rule ensure has completed")

		if !reduce {
			r.log.Sugar().Info("vxlan and route has completed")
			reduce = true
		}

		time.Sleep(time.Second * 10)
	}
}

func (r *vxlanReconciler) ensureRoute() error {
	neighList, err := r.vxlan.ListNeigh()
	if err != nil {
		return err
	}

	peerMap := make(map[string]vxlan.Peer, 0)
	r.peerMap.Range(func(key string, peer vxlan.Peer) bool {
		peerMap[key] = peer
		return false
	})

	expected := make(map[string]struct{}, 0)
	for _, peer := range peerMap {
		expected[peer.MAC.String()] = struct{}{}
	}

	for _, existing := range neighList {
		if _, ok := expected[existing.HardwareAddr.String()]; !ok {
			err := r.vxlan.Del(existing)
			if err != nil {
				r.log.Sugar().Warn("del existing neigh with error: %v, %v", existing, err)
			}
		}
	}

	for _, peer := range peerMap {
		err := r.vxlan.Add(peer)
		if err != nil {
			r.log.Sugar().Errorf("add peer route with error: %v, %v", peer, err)
		}
	}

	return nil
}

func newEgressNodeController(mgr manager.Manager, cfg *config.Config, log *zap.Logger) error {
	multiPath := false
	if cfg.FileConfig.ForwardMethod == "active-active" {
		multiPath = true
	}
	ruleRoute := route.NewRuleRoute(cfg.FileConfig.StartRouteTable, 0x11000000, 0xffffffff, multiPath, log)

	r := &vxlanReconciler{
		client:    mgr.GetClient(),
		log:       log,
		cfg:       cfg,
		peerMap:   utils.NewSyncMap[string, vxlan.Peer](),
		vxlan:     vxlan.New(),
		getParent: vxlan.GetParent,
		ruleRoute: ruleRoute,
		ruleRouteCache: RuleRouteCache{
			ipv4List: make([]net.IP, 0),
			ipv6List: make([]net.IP, 0),
		},
	}

	c, err := controller.New("vxlan", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressGateway{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGateway"))); err != nil {
		return fmt.Errorf("failed to watch EgressGateway: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressNode{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressNode"))); err != nil {
		return fmt.Errorf("failed to watch EgressNode: %w", err)
	}

	go r.keepVXLAN()

	return nil
}
