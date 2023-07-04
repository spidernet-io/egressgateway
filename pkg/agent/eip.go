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
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/layer2"
)

type eip struct {
	client client.Client
	log    logr.Logger
	cfg    *config.Config

	announce *layer2.Announce
}

func (r *eip) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("name", req.NamespacedName.Name, "kind", "EgressGateway")
	log.Info("reconcile")

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
		if ip.To4() == nil {
			continue
		}
		adv := layer2.NewIPAdvertisement(ip, true, sets.Set[string]{})
		r.announce.SetBalancer(gateway.Name, adv)

		ip = net.ParseIP(status.IPv6)
		if ip.To16() == nil {
			continue
		}
		adv = layer2.NewIPAdvertisement(ip, true, sets.Set[string]{})
		r.announce.SetBalancer(gateway.Name, adv)
	}

	return reconcile.Result{}, nil
}

// newEipCtrl return a new egress ip controller
func newEipCtrl(mgr manager.Manager, log logr.Logger, cfg *config.Config) error {
	lw := logWrapper{log: log}
	an, err := layer2.New(lw, nil)
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

type logWrapper struct {
	log logr.Logger
}

func (lw logWrapper) Log(keyVals ...interface{}) error {
	fields := make([]interface{}, 0, len(keyVals)/2)
	var msgValue interface{}
	var kind string
	for i := 0; i < len(keyVals); i += 2 {
		key, ok := keyVals[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", keyVals[i])
		}
		if key == "msg" {
			msgValue = keyVals[i+1]
		} else if key == "level" {
			kind = fmt.Sprintf("%v", keyVals[i+1])
		} else {
			fields = append(fields, key, keyVals[i+1])
		}
	}

	var msg string
	if msgValue != nil {
		msg = fmt.Sprintf("%v", msgValue)
	}

	switch kind {
	case "debug":
		lw.log.V(1).Info(msg, fields...)
	case "warn":
		lw.log.Info(msg, fields...)
	case "error":
		lw.log.Error(fmt.Errorf(msg), "", fields...)
	default:
		lw.log.Info(msg, fields...)
	}

	return nil
}
