// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"net"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/layer2"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

type eip struct {
	client client.Client
	log    logr.Logger
	cfg    *config.Config

	announce *layer2.Announce
}

func (r *eip) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		return reconcile.Result{}, err
	}
	log := r.log.WithValues("kind", kind)
	var res reconcile.Result
	switch kind {
	case "EgressClusterPolicy":
		res, err = r.reconcileClusterPolicy(ctx, newReq, log)
	case "EgressPolicy":
		res, err = r.reconcilePolicy(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
	return res, err
}

func (r *eip) reconcilePolicy(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	log = log.WithValues("name", req.Name, "namespace", req.Namespace)
	log.V(1).Info("reconcile")

	deleted := false
	policy := new(egressv1.EgressPolicy)
	err := r.client.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !policy.GetDeletionTimestamp().IsZero()

	if deleted {
		r.announce.DeleteBalancer(req.NamespacedName.String())
		return reconcile.Result{}, nil
	}

	if policy.Status.Node != r.cfg.NodeName {
		r.announce.DeleteBalancer(req.NamespacedName.String())
		return reconcile.Result{}, nil
	}

	ip := net.ParseIP(policy.Status.Eip.Ipv4)
	if ip.To4() != nil {
		adv := layer2.NewIPAdvertisement(ip, true, sets.Set[string]{})
		r.announce.SetBalancer(req.NamespacedName.String(), adv)
	}

	ip = net.ParseIP(policy.Status.Eip.Ipv6)
	if ip.To16() != nil {
		adv := layer2.NewIPAdvertisement(ip, true, sets.Set[string]{})
		r.announce.SetBalancer(req.NamespacedName.String(), adv)
	}

	return reconcile.Result{}, nil
}

func (r *eip) reconcileClusterPolicy(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	log = log.WithValues("name", req.Name)
	log.V(1).Info("reconcile")

	deleted := false
	policy := new(egressv1.EgressClusterPolicy)
	err := r.client.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !policy.GetDeletionTimestamp().IsZero()

	if deleted {
		r.announce.DeleteBalancer(req.NamespacedName.String())
		return reconcile.Result{}, nil
	}

	if policy.Status.Node != r.cfg.NodeName {
		r.announce.DeleteBalancer(req.NamespacedName.String())
		return reconcile.Result{}, nil
	}

	ip := net.ParseIP(policy.Status.Eip.Ipv4)
	if ip.To4() != nil {
		adv := layer2.NewIPAdvertisement(ip, true, sets.Set[string]{})
		r.announce.SetBalancer(req.NamespacedName.String(), adv)
	}

	ip = net.ParseIP(policy.Status.Eip.Ipv6)
	if ip.To16() != nil {
		adv := layer2.NewIPAdvertisement(ip, true, sets.Set[string]{})
		r.announce.SetBalancer(req.NamespacedName.String(), adv)
	}

	return reconcile.Result{}, nil
}

// newEipCtrl return a new egress ip controller
func newEipCtrl(mgr manager.Manager, log logr.Logger, cfg *config.Config) error {
	an, err := layer2.New(log, cfg.FileConfig.AnnounceExcludeRegexp)
	if err != nil {
		return err
	}

	eip := &eip{
		cfg:      cfg,
		log:      log,
		client:   mgr.GetClient(),
		announce: an,
	}

	c, err := controller.New("eip", mgr, controller.Options{Reconciler: eip})
	if err != nil {
		return err
	}

	sourceEgressPolicy := utils.SourceKind(
		mgr.GetCache(),
		&egressv1.EgressPolicy{},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressPolicy")),
	)
	if err := c.Watch(sourceEgressPolicy); err != nil {
		return fmt.Errorf("failed to watch EgressPolicy: %w", err)
	}

	sourceEgressPolicy = utils.SourceKind(
		mgr.GetCache(),
		&egressv1.EgressClusterPolicy{},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressClusterPolicy")),
	)
	if err := c.Watch(sourceEgressPolicy); err != nil {
		return fmt.Errorf("failed to watch EgressClusterPolicy: %w", err)

	}

	return nil
}
