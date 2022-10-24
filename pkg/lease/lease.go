// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package lease

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coordinationv1client "k8s.io/client-go/kubernetes/typed/coordination/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	leaseDuration      = time.Duration(15) * time.Second
	leaseRenewDeadline = time.Duration(10) * time.Second
	leaseRetryPeriod   = time.Duration(2) * time.Second
)

// NewLeaseElector will return a SpiderLeaseElector object
func NewLeaseElector(ctx context.Context, leaseNamespace, leaseName, leaseID string, logger *zap.Logger) (getLease chan struct{}, lossLease chan struct{}, err error) {

	if len(leaseNamespace) == 0 {
		return nil, nil, fmt.Errorf("failed to new lease elector: Lease Namespace must be specified")
	}
	if len(leaseName) == 0 {
		return nil, nil, fmt.Errorf("failed to new lease elector: Lease Name must be specified")
	}
	if len(leaseID) == 0 {
		return nil, nil, fmt.Errorf("failed to new lease elector: Lease Identity must be specified")
	}
	if logger == nil {
		return nil, nil, fmt.Errorf("miss logger")
	}

	getLease = make(chan struct{})
	lossLease = make(chan struct{})

	coordinationClient, e := coordinationv1client.NewForConfig(ctrl.GetConfigOrDie())
	if e != nil {
		return nil, nil, fmt.Errorf("fail to new coordination client: %w", e)
	}

	leaseLock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: leaseNamespace,
		},
		Client: coordinationClient,
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: leaseID,
		},
	}

	le, e := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          leaseLock,
		LeaseDuration: leaseDuration,
		RenewDeadline: leaseRenewDeadline,
		RetryPeriod:   leaseRetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(_ context.Context) {
				close(getLease)
				logger.Sugar().Infof("get lease: %s/%s ", leaseNamespace, leaseName)
			},
			OnStoppedLeading: func() {
				// we can do cleanup here
				close(lossLease)
				logger.Sugar().Warnf("lost lease : %s/%s ", leaseNamespace, leaseName)
			},
		},
		ReleaseOnCancel: true,
	})
	if nil != e {
		return nil, nil, fmt.Errorf("unable to new leader elector: %w", e)
	}

	go func() {
		// Run will not return before leader election loop is stopped by ctx or it has stopped holding the leader lease
		le.Run(ctx)
		select {
		case <-ctx.Done():
			logger.Sugar().Warnf("context is done, exit lease goroutine, detail=%v", ctx.Err())
			return
		default:
		}
	}()

	return
}
