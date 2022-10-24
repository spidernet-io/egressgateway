// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressGatewayManager

import (
	"context"
	"fmt"
	crdclientset "github.com/spidernet-io/egressgateway/pkg/k8s/client/clientset/versioned"
	"github.com/spidernet-io/egressgateway/pkg/k8s/client/informers/externalversions"
	"github.com/spidernet-io/egressgateway/pkg/lease"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"time"
)

type informerHandler struct {
	logger         *zap.Logger
	leaseName      string
	leaseNameSpace string
	leaseId        string
}

func (s *informerHandler) informerAddHandler(obj interface{}) {
	s.logger.Sugar().Infof("start crd add: %+v", obj)

	time.Sleep(30 * time.Second)
	s.logger.Sugar().Infof("done crd add: %+v", obj)
}

func (s *informerHandler) informerUpdateHandler(oldObj interface{}, newObj interface{}) {
	s.logger.Sugar().Infof("crd update old: %+v", oldObj)
	s.logger.Sugar().Infof("crd update new: %+v", newObj)
}

func (s *informerHandler) informerDeleteHandler(obj interface{}) {
	s.logger.Sugar().Infof("crd delete: %+v", obj)
}

func (s *informerHandler) executeInformer() {

	config, err := rest.InClusterConfig()
	if err != nil {
		s.logger.Sugar().Fatalf("failed to InClusterConfig, reason=%v", err)
	}
	clientset, err := crdclientset.NewForConfig(config) // 初始化 client
	if err != nil {
		s.logger.Sugar().Fatalf("failed to NewForConfig, reason=%v", err)
		return
	}

	stopInfomer := make(chan struct{})

	if len(s.leaseName) > 0 && len(s.leaseNameSpace) > 0 && len(s.leaseId) > 0 {
		s.logger.Sugar().Infof("%v try to get lease %s/%s to run informer", s.leaseId, s.leaseNameSpace, s.leaseName)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		rlogger := s.logger.Named(fmt.Sprintf("lease %s/%s", s.leaseNameSpace, s.leaseName))
		// id := globalConfig.PodName
		getLease, lossLease, err := lease.NewLeaseElector(ctx, s.leaseNameSpace, s.leaseName, s.leaseId, rlogger)
		if err != nil {
			s.logger.Sugar().Fatalf("failed to generate lease, reason=%v ", err)
		}
		<-getLease
		s.logger.Sugar().Infof("succeed to get lease %s/%s to run informer", s.leaseNameSpace, s.leaseName)

		go func(lossLease chan struct{}) {
			<-lossLease
			close(stopInfomer)
			s.logger.Sugar().Warnf("lease %s/%s is loss, informer is broken", s.leaseNameSpace, s.leaseName)
		}(lossLease)
	}

	// setup informer
	s.logger.Info("begin to setup informer")
	factory := externalversions.NewSharedInformerFactory(clientset, 0)
	// 注意，一个 factory 下  对同一种 CRD 不能 创建 多个Informer，不然会 数据竞争 问题。 而 一个 factory 下， 可对不同 CRD 产生 各种的 Informer
	inform := factory.Egressgateway().V1().EgressGateways().Informer()

	// 在一个 Handler 逻辑中，是顺序消费所有的 crd 事件的
	// 简单说：有2个 crd add 事件，那么，先会调用 informerAddHandler 完成 事件1 后，才会 调用 informerAddHandler 处理 事件2
	inform.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.informerAddHandler,
		UpdateFunc: s.informerUpdateHandler,
		DeleteFunc: s.informerDeleteHandler,
	})

	// 一个 inform 下  如果注册 第二套 AddEventHandler，那么，对于同一个 事件，两套 handler 是 使用 独立协程 并发调用的 .
	// 这样，就能实现对同一个事件 并发调用不同的回调，好处是，他们底层是基于同一个 NewSharedInformer ， 共用一个cache，能降低api server 之间的数据同步
	inform.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.informerAddHandler,
		UpdateFunc: s.informerUpdateHandler,
		DeleteFunc: s.informerDeleteHandler,
	})

	inform.Run(stopInfomer)

}

func (s *mybookManager) RunInformer(leaseName, leaseNameSpace string, leaseId string) {
	t := &informerHandler{
		logger:         s.logger,
		leaseName:      leaseName,
		leaseNameSpace: leaseNameSpace,
		leaseId:        leaseId,
	}
	s.informer = t

	go func() {
		for {
			t.executeInformer()
			time.Sleep(time.Duration(5) * time.Second)
		}
	}()
}
