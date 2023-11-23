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
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/layer2"
)

type eip struct {
	client client.Client
	log    logr.Logger
	cfg    *config.Config

	announce *layer2.Announce
}

func (r *eip) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("name", req.Name, "kind", "EgressGateway")
	log.V(1).Info("reconcile")

	deleted := false
	gateway := new(egressv1.EgressGateway)
	err := r.client.Get(ctx, req.NamespacedName, gateway)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !gateway.GetDeletionTimestamp().IsZero()

	if deleted {
		r.announce.DeleteBalancer(req.NamespacedName.Name)
		return reconcile.Result{}, nil
	}

	ips := gateway.Status.GetNodeIPs(r.cfg.NodeName)
	for _, status := range ips {
		ip := net.ParseIP(status.IPv4)
		if ip.To4() != nil {
			adv := layer2.NewIPAdvertisement(ip, true, sets.Set[string]{})
			r.announce.SetBalancer(gateway.Name, adv)
		}
		ip = net.ParseIP(status.IPv6)
		if ip.To16() != nil {
			adv := layer2.NewIPAdvertisement(ip, true, sets.Set[string]{})
			r.announce.SetBalancer(gateway.Name, adv)
		}
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

	if err = c.Watch(source.Kind(mgr.GetCache(), &egressv1.EgressGateway{}),
		&handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch EgressGateway: %v", err)
	}

	return nil
}
