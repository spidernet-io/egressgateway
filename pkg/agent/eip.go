// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"github.com/spidernet-io/egressgateway/pkg/layer2"
	"k8s.io/apimachinery/pkg/util/sets"
	"net"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
)

type eip struct {
	client client.Client
	log    *zap.Logger
	cfg    *config.Config

	announce *layer2.Announce
}

func (r *eip) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.With(
		zap.String("namespace", req.NamespacedName.Namespace),
		zap.String("name", req.NamespacedName.Name),
		zap.String("kind", "EgressGateway"),
	)

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
func newEipCtrl(mgr manager.Manager, log *zap.Logger, cfg *config.Config) error {
	lw := logWrapper{log: log.Named("layer2")}
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
	log *zap.Logger
}

func (lw logWrapper) Log(keyVals ...interface{}) error {
	fields := make([]zap.Field, 0, len(keyVals)/2)
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
			fields = append(fields, zap.Any(key, keyVals[i+1]))
		}
	}

	var msg string
	if msgValue != nil {
		msg = fmt.Sprintf("%v", msgValue)
	}

	switch kind {
	case "debug":
		lw.log.Debug(msg, fields...)
	case "warn":
		lw.log.Warn(msg, fields...)
	case "error":
		lw.log.Error(msg, fields...)
	default:
		lw.log.Info(msg, fields...)
	}

	return nil
}
