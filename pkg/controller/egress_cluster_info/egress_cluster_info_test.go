// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-logr/logr"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
	"github.com/stretchr/testify/assert"
	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var ErrForMock = errors.New("mock err")

func Test_NewEgressClusterInfoController(t *testing.T) {
	kubeConfig := &rest.Config{}
	mgr, _ := ctrl.NewManager(kubeConfig, manager.Options{})
	log := logr.Logger{}

	patch := gomonkey.NewPatches()
	patch.ApplyFuncReturn(controller.New, nil, ErrForMock)
	defer patch.Reset()

	err := NewEgressClusterInfoController(mgr, log)
	assert.Error(t, err)
}

func Test_eciReconciler_Reconcile(t *testing.T) {
	cases := map[string]struct {
		getReqFunc    func() reconcile.Request
		setReconciler func(*eciReconciler)
		patchFunc     func(*eciReconciler) []gomonkey.Patches
		expErr        bool
		expRequeue    bool
	}{
		"reconcile calico, AutoDetect.PodCidrMode not calico": {
			getReqFunc:    mock_request_calico,
			setReconciler: mock_eciReconciler_info_AutoDetect_PodCidrMode_not_calico,
			patchFunc:     mock_eciReconciler_getEgressClusterInfo_not_err,
		},
		"reconcile no matched kind": {
			getReqFunc: mock_request_no_match,
			patchFunc:  mock_eciReconciler_getEgressClusterInfo_not_err,
		},
		"failed status Update IsConflict": {
			getReqFunc:    mock_request_calico,
			setReconciler: mock_eciReconciler_info_AutoDetect_PodCidrMode_calico,
			patchFunc:     mock_Reconciler_Reconcile_failed_Update,
			expErr:        false,
			expRequeue:    true,
		},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())

	r := &eciReconciler{
		eci:           new(egressv1.EgressClusterInfo),
		log:           logr.Logger{},
		k8sPodCidr:    make(map[string]egressv1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
		client:        builder.Build(),
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.setReconciler != nil {
				tc.setReconciler(r)
			}

			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(r)
			}

			req := tc.getReqFunc()
			ctx := context.TODO()

			res, err := r.Reconcile(ctx, req)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tc.expRequeue {
				assert.True(t, res.Requeue)
			}

			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_eciReconciler_reconcileEgressClusterInfo(t *testing.T) {
	cases := map[string]struct {
		setReconciler func(*eciReconciler)
		patchFunc     func(*eciReconciler) []gomonkey.Patches
		expErr        bool
	}{
		"not watch node, failed watch node": {
			setReconciler: mock_eciReconciler_info_isWatchingNode_false,
			patchFunc:     mock_Reconciler_reconcileEgressClusterInfo_watchSource_err,
			expErr:        true,
		},

		"not watnch node, failed listNodeIPs": {
			setReconciler: mock_eciReconciler_info_isWatchingNode_false,
			patchFunc:     mock_Reconciler_reconcileEgressClusterInfo_listNodeIPs_err,
			expErr:        true,
		},

		"not watch node, succeeded listNodeIPs": {
			setReconciler: mock_eciReconciler_info_isWatchingNode_false,
			patchFunc:     mock_Reconciler_reconcileEgressClusterInfo_listNodeIPs_succ,
		},

		"need stopCheckCalico": {
			setReconciler: mock_eciReconciler_info_need_stopCheckCalico,
		},

		"failed checkSomeCniExists": {
			setReconciler: mock_eciReconciler_info_AutoDetect_PodCidrMode_auto,
			patchFunc:     mock_Reconciler_reconcileEgressClusterInfo_checkSomeCniExists_err,
			expErr:        true,
		},

		"autoDetect calico, need watch calico, startCheckCalico": {
			setReconciler: mock_eciReconciler_info_AutoDetect_calico_isWatchingCalico_false,
			patchFunc:     mock_Reconciler_reconcileEgressClusterInfo_checkSomeCniExists_err,
		},

		"autoDetect calico, watching calico, failed listCalicoIPPools": {
			setReconciler: mock_eciReconciler_info_AutoDetect_calico_isWatchingCalico_true,
			patchFunc:     mock_Reconciler_reconcileEgressClusterInfo_listCalicoIPPools_err,
			expErr:        true,
		},

		// "autoDetect ClusterIP, failed getServiceClusterIPRange": {
		// 	setReconciler: mock_eciReconciler_info_AutoDetect_ClusterIP,
		// 	patchFunc:     mock_Reconciler_reconcileEgressClusterInfo_getServiceClusterIPRange_err,
		// 	expErr:        true,
		// },
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	cli := builder.Build()

	// objs = append(objs, egci)
	// 		builder.WithObjects(objs...)
	// 		builder.WithStatusSubresource(objs...)

	// mgrOpts := manager.Options{
	// 	Scheme: schema.GetScheme(),
	// 	NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
	// 		return cli, nil
	// 	},
	// }

	mgr, _ := ctrl.NewManager(&rest.Config{}, manager.Options{})

	r := &eciReconciler{
		mgr:           mgr,
		eci:           new(egressv1.EgressClusterInfo),
		log:           logr.Logger{},
		k8sPodCidr:    make(map[string]egressv1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
		client:        cli,
	}
	c, _ := controller.New("egressClusterInfo", mgr,
		controller.Options{Reconciler: r})
	r.c = c

	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindEGCI + "/", Name: egciName}}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.setReconciler != nil {
				tc.setReconciler(r)
			}

			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(r)
			}

			ctx := context.TODO()

			err := r.reconcileEgressClusterInfo(ctx, req, r.log)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, p := range patches {
				p.Reset()
			}
		})
	}

}

func Test_eciReconciler_reconcileCalicoIPPool(t *testing.T) {
	cases := map[string]struct {
		setReconciler func(*eciReconciler)
		patchFunc     func(*eciReconciler) []gomonkey.Patches
		expErr        bool
	}{
		"failed get calicoIPPool": {
			patchFunc: mock_Reconciler_reconcileCalicoIPPool_Get_err,
			expErr:    true,
		},

		"failed getCalicoIPPools": {
			patchFunc: mock_Reconciler_reconcileCalicoIPPool_getCalicoIPPools_err,
			expErr:    true,
		},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	cli := builder.Build()

	// objs = append(objs, egci)
	// 		builder.WithObjects(objs...)
	// 		builder.WithStatusSubresource(objs...)

	// mgrOpts := manager.Options{
	// 	Scheme: schema.GetScheme(),
	// 	NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
	// 		return cli, nil
	// 	},
	// }

	mgr, _ := ctrl.NewManager(&rest.Config{}, manager.Options{})

	r := &eciReconciler{
		mgr:           mgr,
		eci:           new(egressv1.EgressClusterInfo),
		log:           logr.Logger{},
		k8sPodCidr:    make(map[string]egressv1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
		client:        cli,
	}
	c, _ := controller.New("egressClusterInfo", mgr,
		controller.Options{Reconciler: r})
	r.c = c

	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindCalicoIPPool + "/", Name: "xxx"}}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.setReconciler != nil {
				tc.setReconciler(r)
			}

			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(r)
			}

			ctx := context.TODO()

			err := r.reconcileCalicoIPPool(ctx, req, r.log)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_eciReconciler_reconcileNode(t *testing.T) {
	cases := map[string]struct {
		setReconciler func(*eciReconciler)
		patchFunc     func(*eciReconciler) []gomonkey.Patches
		expErr        bool
	}{
		"failed get node": {
			patchFunc: mock_Reconciler_reconcileNode_Get_err,
			expErr:    true,
		},

		"failed getNodeIPs": {
			patchFunc: mock_Reconciler_reconcileNode_getNodeIPs_err,
			expErr:    true,
		},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	cli := builder.Build()

	// objs = append(objs, egci)
	// 		builder.WithObjects(objs...)
	// 		builder.WithStatusSubresource(objs...)

	// mgrOpts := manager.Options{
	// 	Scheme: schema.GetScheme(),
	// 	NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
	// 		return cli, nil
	// 	},
	// }

	mgr, _ := ctrl.NewManager(&rest.Config{}, manager.Options{})

	r := &eciReconciler{
		mgr:           mgr,
		eci:           new(egressv1.EgressClusterInfo),
		log:           logr.Logger{},
		k8sPodCidr:    make(map[string]egressv1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
		client:        cli,
	}
	c, _ := controller.New("egressClusterInfo", mgr,
		controller.Options{Reconciler: r})
	r.c = c

	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindCalicoIPPool + "/", Name: "xxx"}}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.setReconciler != nil {
				tc.setReconciler(r)
			}

			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(r)
			}

			ctx := context.TODO()

			err := r.reconcileNode(ctx, req, r.log)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_eciReconciler_listCalicoIPPools(t *testing.T) {
	cases := map[string]struct {
		setReconciler func(*eciReconciler)
		patchFunc     func(*eciReconciler) []gomonkey.Patches
		expErr        bool
	}{
		"failed List": {
			patchFunc: mock_Reconciler_listCalicoIPPools_List_err,
			expErr:    true,
		},

		"failed IsIPv4Cidr": {
			patchFunc: mock_Reconciler_listCalicoIPPools_IsIPv4Cidr_err,
			expErr:    true,
		},
		"failed IsIPv6Cidr": {
			patchFunc: mock_Reconciler_listCalicoIPPools_IsIPv6Cidr_err,
			expErr:    true,
		},
	}

	calicoIPPoolV4 := &calicov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ippool-v4",
		},
		Spec: calicov1.IPPoolSpec{
			// CIDR: "xxx",
			CIDR: "10.10.0.0/18",
		},
	}
	calicoIPPoolV6 := &calicov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ippool-v6",
		},
		Spec: calicov1.IPPoolSpec{
			CIDR: "fdee:120::/120",
		},
	}

	var objs []client.Object

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	objs = append(objs, calicoIPPoolV4, calicoIPPoolV6)
	builder.WithObjects(objs...)
	builder.WithStatusSubresource(objs...)

	cli := builder.Build()

	mgrOpts := manager.Options{
		Scheme: schema.GetScheme(),
		NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
			return cli, nil
		},
	}

	mgr, _ := ctrl.NewManager(&rest.Config{}, mgrOpts)

	r := &eciReconciler{
		mgr:           mgr,
		eci:           new(egressv1.EgressClusterInfo),
		log:           logr.Logger{},
		k8sPodCidr:    make(map[string]egressv1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
		client:        cli,
	}
	c, _ := controller.New("egressClusterInfo", mgr,
		controller.Options{Reconciler: r})
	r.c = c

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.setReconciler != nil {
				tc.setReconciler(r)
			}

			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(r)
			}

			ctx := context.TODO()

			_, err := r.listCalicoIPPools(ctx)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_eciReconciler_getCalicoIPPools(t *testing.T) {
	cases := map[string]struct {
		setReconciler func(*eciReconciler)
		objs          []client.Object
		patchFunc     func(*eciReconciler) []gomonkey.Patches
		expErr        bool
	}{
		"failed Get": {
			objs:      mock_IPPoolList(),
			patchFunc: mock_Reconciler_reconcileNode_getCalicoIPPools_err,
			expErr:    true,
		},

		"failed IsIPv4Cidr": {
			objs:      mock_calicoIPPoolV4(),
			patchFunc: mock_Reconciler_listCalicoIPPools_IsIPv4Cidr_err,
			expErr:    true,
		},
		"failed IsIPv6Cidr": {
			objs:      mock_calicoIPPoolV6(),
			patchFunc: mock_Reconciler_listCalicoIPPools_IsIPv6Cidr_err,
			expErr:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			builder.WithObjects(tc.objs...)
			builder.WithStatusSubresource(tc.objs...)

			cli := builder.Build()

			mgrOpts := manager.Options{
				Scheme: schema.GetScheme(),
				NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
					return cli, nil
				},
			}

			mgr, _ := ctrl.NewManager(&rest.Config{}, mgrOpts)

			r := &eciReconciler{
				mgr:           mgr,
				eci:           new(egressv1.EgressClusterInfo),
				log:           logr.Logger{},
				k8sPodCidr:    make(map[string]egressv1.IPListPair),
				v4ClusterCidr: make([]string, 0),
				v6ClusterCidr: make([]string, 0),
				client:        cli,
			}
			c, _ := controller.New("egressClusterInfo", mgr,
				controller.Options{Reconciler: r})
			r.c = c

			if tc.setReconciler != nil {
				tc.setReconciler(r)
			}

			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(r)
			}

			ctx := context.TODO()

			_, err := r.getCalicoIPPools(ctx, tc.objs[0].GetName())
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_eciReconciler_listNodeIPs(t *testing.T) {
	cases := map[string]struct {
		setReconciler func(*eciReconciler)
		patchFunc     func(*eciReconciler) []gomonkey.Patches
		expErr        bool
	}{
		"failed List": {
			patchFunc: mock_Reconciler_listNodeIPs_List_err,
			expErr:    true,
		},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())

	cli := builder.Build()

	mgrOpts := manager.Options{
		Scheme: schema.GetScheme(),
		NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
			return cli, nil
		},
	}

	mgr, _ := ctrl.NewManager(&rest.Config{}, mgrOpts)

	r := &eciReconciler{
		mgr:           mgr,
		eci:           new(egressv1.EgressClusterInfo),
		log:           logr.Logger{},
		k8sPodCidr:    make(map[string]egressv1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
		client:        cli,
	}
	c, _ := controller.New("egressClusterInfo", mgr,
		controller.Options{Reconciler: r})
	r.c = c

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.setReconciler != nil {
				tc.setReconciler(r)
			}

			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(r)
			}

			ctx := context.TODO()

			_, err := r.listNodeIPs(ctx)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_eciReconciler_getNodeIPs(t *testing.T) {
	cases := map[string]struct {
		setReconciler func(*eciReconciler)
		patchFunc     func(*eciReconciler) []gomonkey.Patches
		expErr        bool
	}{
		"failed Get": {
			patchFunc: mock_Reconciler_reconcileNode_getNodeIPs_Get_err,
			expErr:    true,
		},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())

	cli := builder.Build()

	mgrOpts := manager.Options{
		Scheme: schema.GetScheme(),
		NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
			return cli, nil
		},
	}

	mgr, _ := ctrl.NewManager(&rest.Config{}, mgrOpts)

	r := &eciReconciler{
		mgr:           mgr,
		eci:           new(egressv1.EgressClusterInfo),
		log:           logr.Logger{},
		k8sPodCidr:    make(map[string]egressv1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
		client:        cli,
	}
	c, _ := controller.New("egressClusterInfo", mgr,
		controller.Options{Reconciler: r})
	r.c = c

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.setReconciler != nil {
				tc.setReconciler(r)
			}

			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(r)
			}

			ctx := context.TODO()

			_, err := r.getNodeIPs(ctx, "fakeName")
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_eciReconciler_checkCalicoExists(t *testing.T) {

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())

	cli := builder.Build()

	mgrOpts := manager.Options{
		Scheme: schema.GetScheme(),
		NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
			return cli, nil
		},
	}

	mgr, _ := ctrl.NewManager(&rest.Config{}, mgrOpts)

	r := &eciReconciler{
		mgr:           mgr,
		eci:           new(egressv1.EgressClusterInfo),
		log:           logr.Logger{},
		k8sPodCidr:    make(map[string]egressv1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
		client:        cli,
	}
	c, _ := controller.New("egressClusterInfo", mgr,
		controller.Options{Reconciler: r})
	r.c = c

	t.Run("isWatchCalico is false", func(t *testing.T) {
		r.isWatchCalico.Store(false)
		r.checkCalicoExists()
	})
	t.Run("failed listCalicoIPPools", func(t *testing.T) {
		r.isWatchCalico.Store(true)

		var patches []gomonkey.Patches
		defer func() {
			for _, p := range patches {
				p.Reset()
			}
		}()

		go func() {
			patch1 := gomonkey.ApplyPrivateMethod(r, "listCalicoIPPools", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
				return nil, ErrForMock
			})

			time.Sleep(time.Second * 3)
			patch1.Reset()

			patch2 := gomonkey.ApplyPrivateMethod(r, "listCalicoIPPools", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
				return nil, nil
			})
			patches = append(patches, *patch2)

			patch3 := gomonkey.ApplyFuncReturn(watchSource, nil)
			patches = append(patches, *patch3)

		}()

		time.Sleep(time.Second)

		r.checkCalicoExists()

	})

	t.Run("failed watchSource", func(t *testing.T) {
		calicoIPPoolV4 := &calicov1.IPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ippool-v4",
			},
			Spec: calicov1.IPPoolSpec{
				// CIDR: "xxx",
				CIDR: "10.10.0.0/18",
			},
		}
		calicoIPPoolV6 := &calicov1.IPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ippool-v6",
			},
			Spec: calicov1.IPPoolSpec{
				CIDR: "fdee:120::/120",
			},
		}

		var objs []client.Object

		builder := fake.NewClientBuilder()
		builder.WithScheme(schema.GetScheme())
		objs = append(objs, calicoIPPoolV4, calicoIPPoolV6)
		builder.WithObjects(objs...)
		builder.WithStatusSubresource(objs...)
		cli := builder.Build()

		mgrOpts := manager.Options{
			Scheme: schema.GetScheme(),
			NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
				return cli, nil
			},
		}

		mgr, _ := ctrl.NewManager(&rest.Config{}, mgrOpts)

		r := &eciReconciler{
			mgr:           mgr,
			eci:           new(egressv1.EgressClusterInfo),
			log:           logr.Logger{},
			k8sPodCidr:    make(map[string]egressv1.IPListPair),
			v4ClusterCidr: make([]string, 0),
			v6ClusterCidr: make([]string, 0),
			client:        cli,
		}
		c, _ := controller.New("egressClusterInfo", mgr,
			controller.Options{Reconciler: r})
		r.c = c

		r.isWatchCalico.Store(true)

		var patches []gomonkey.Patches

		patch2 := gomonkey.ApplyFuncSeq(watchSource, []gomonkey.OutputCell{
			{Values: gomonkey.Params{ErrForMock}, Times: 3},
			{Values: gomonkey.Params{nil}, Times: 3},
		})
		patches = append(patches, *patch2)

		r.checkCalicoExists()

		for _, p := range patches {
			p.Reset()
		}
	})
}

func Test_eciReconciler_getServiceClusterIPRange(t *testing.T) {

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())

	cli := builder.Build()

	mgrOpts := manager.Options{
		Scheme: schema.GetScheme(),
		NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
			return cli, nil
		},
	}

	mgr, _ := ctrl.NewManager(&rest.Config{}, mgrOpts)

	r := &eciReconciler{
		mgr:           mgr,
		eci:           new(egressv1.EgressClusterInfo),
		log:           logr.Logger{},
		k8sPodCidr:    make(map[string]egressv1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
		client:        cli,
	}
	c, _ := controller.New("egressClusterInfo", mgr,
		controller.Options{Reconciler: r})
	r.c = c

	t.Run("failed GetPodByLabel", func(t *testing.T) {
		patch := gomonkey.ApplyFuncReturn(GetPodByLabel, nil, ErrForMock)
		defer patch.Reset()
		_, _, err := r.getServiceClusterIPRange()
		assert.Error(t, err)
	})
}

func Test_eciReconciler_checkSomeCniExists(t *testing.T) {
	cases := map[string]struct {
		setReconciler func(*eciReconciler)
		patchFunc     func(*eciReconciler) []gomonkey.Patches
		expErr        bool
	}{
		"failed listCalicoIPPools": {
			patchFunc: mock_Reconciler_checkSomeCniExists_listCalicoIPPools_err,
			expErr:    true,
		},
		"failed watchSource": {
			patchFunc: mock_Reconciler_checkSomeCniExists_watchSource_err,
			expErr:    true,
		},
		"succeeded watchSource": {
			patchFunc: mock_Reconciler_checkSomeCniExists_watchSource_succ,
		},

		"failed getK8sPodCidr": {
			patchFunc: mock_Reconciler_checkSomeCniExists_getK8sPodCidr_err,
			expErr:    true,
		},

		"succeeded getK8sPodCidr": {
			patchFunc: mock_Reconciler_checkSomeCniExists_getK8sPodCidr_succ,
		},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())

	cli := builder.Build()

	mgrOpts := manager.Options{
		Scheme: schema.GetScheme(),
		NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
			return cli, nil
		},
	}

	mgr, _ := ctrl.NewManager(&rest.Config{}, mgrOpts)

	r := &eciReconciler{
		mgr:           mgr,
		eci:           new(egressv1.EgressClusterInfo),
		log:           logr.Logger{},
		k8sPodCidr:    make(map[string]egressv1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
		client:        cli,
	}
	c, _ := controller.New("egressClusterInfo", mgr,
		controller.Options{Reconciler: r})
	r.c = c

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.setReconciler != nil {
				tc.setReconciler(r)
			}

			patches := make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patches = tc.patchFunc(r)
			}

			err := r.checkSomeCniExists()
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}
func Test_watchSource(t *testing.T) {

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())

	cli := builder.Build()

	mgr, _ := ctrl.NewManager(&rest.Config{}, manager.Options{})

	r := &eciReconciler{
		mgr:           mgr,
		eci:           new(egressv1.EgressClusterInfo),
		log:           logr.Logger{},
		k8sPodCidr:    make(map[string]egressv1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
		client:        cli,
	}
	c, _ := controller.New("egressClusterInfo", mgr,
		controller.Options{Reconciler: r})
	r.c = c

	t.Run("failed Watch", func(t *testing.T) {
		patch := gomonkey.ApplyMethodReturn(c, "Watch", ErrForMock)
		defer patch.Reset()
		err := watchSource(c, source.Kind(mgr.GetCache(), &egressv1.EgressClusterInfo{}), kindEGCI)
		assert.Error(t, err)
	})
}

func mock_eciReconciler_info_AutoDetect_PodCidrMode_not_calico(r *eciReconciler) {
	r.eci.Spec.AutoDetect.PodCidrMode = egressv1.CniTypeK8s
}

func mock_eciReconciler_info_AutoDetect_PodCidrMode_calico(r *eciReconciler) {
	r.eci.Spec.AutoDetect.PodCidrMode = egressv1.CniTypeCalico
}

func mock_request_calico() reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindCalicoIPPool + "/", Name: "xxx"}}
}

func mock_request_no_match() reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "notMatch" + "/", Name: "xxx"}}
}

func mock_eciReconciler_getEgressClusterInfo_not_err(r *eciReconciler) []gomonkey.Patches {
	patch := gomonkey.ApplyPrivateMethod(r, "getEgressClusterInfo", func(_ *eciReconciler) error {
		return nil
	})
	return []gomonkey.Patches{*patch}
}

func mock_Reconciler_Reconcile_failed_Update(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(r, "getEgressClusterInfo", func(_ *eciReconciler) error {
		return nil
	})
	patch2 := gomonkey.ApplyFuncReturn(reflect.DeepEqual, false)
	patch3 := gomonkey.ApplyPrivateMethod(r, "reconcileCalicoIPPool", func(_ *eciReconciler) error {
		return nil
	})
	patch4 := gomonkey.ApplyFuncReturn(apierrors.IsConflict, true)
	return []gomonkey.Patches{*patch1, *patch2, *patch3, *patch4}
}

func mock_Reconciler_reconcileEgressClusterInfo_watchSource_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(watchSource, ErrForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_eciReconciler_info_isWatchingNode_false(r *eciReconciler) {
	r.eci.Spec.AutoDetect.NodeIP = true
	r.isWatchingNode.Store(false)
}

func mock_Reconciler_reconcileEgressClusterInfo_listNodeIPs_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(watchSource, nil)
	patch2 := gomonkey.ApplyPrivateMethod(r, "listNodeIPs", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, ErrForMock
	})
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_Reconciler_reconcileEgressClusterInfo_listNodeIPs_succ(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(watchSource, nil)
	patch2 := gomonkey.ApplyPrivateMethod(r, "listNodeIPs", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, nil
	})
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_eciReconciler_info_need_stopCheckCalico(r *eciReconciler) {
	r.eci.Spec.AutoDetect.PodCidrMode = egressv1.CniTypeEmpty
	r.isWatchCalico.Store(true)
}

func mock_eciReconciler_info_AutoDetect_PodCidrMode_auto(r *eciReconciler) {
	r.eci.Spec.AutoDetect.PodCidrMode = egressv1.CniTypeAuto
}

func mock_Reconciler_reconcileEgressClusterInfo_checkSomeCniExists_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(r, "checkSomeCniExists", func(_ *eciReconciler) error {
		return ErrForMock
	})
	return []gomonkey.Patches{*patch1}
}

func mock_eciReconciler_info_AutoDetect_calico_isWatchingCalico_false(r *eciReconciler) {
	r.isWatchingCalico.Store(false)
	r.eci.Spec.AutoDetect.PodCidrMode = egressv1.CniTypeCalico
}

func mock_eciReconciler_info_AutoDetect_calico_isWatchingCalico_true(r *eciReconciler) {
	r.isWatchingCalico.Store(true)
	r.eci.Spec.AutoDetect.PodCidrMode = egressv1.CniTypeCalico
}

func mock_Reconciler_reconcileEgressClusterInfo_listCalicoIPPools_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(r, "listCalicoIPPools", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, ErrForMock
	})
	return []gomonkey.Patches{*patch1}
}

func mock_eciReconciler_info_AutoDetect_ClusterIP(r *eciReconciler) {
	r.eci.Spec.AutoDetect.ClusterIP = true
}

func mock_Reconciler_reconcileEgressClusterInfo_getServiceClusterIPRange_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(r, "getServiceClusterIPRange", func(_ *eciReconciler) (ipv4Range, ipv6Range []string, err error) {
		return nil, nil, ErrForMock
	})
	return []gomonkey.Patches{*patch1}
}

func mock_Reconciler_reconcileCalicoIPPool_Get_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", ErrForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_Reconciler_reconcileCalicoIPPool_getCalicoIPPools_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", nil)
	patch2 := gomonkey.ApplyPrivateMethod(r, "getCalicoIPPools", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, ErrForMock
	})
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_Reconciler_reconcileNode_Get_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", ErrForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_Reconciler_reconcileNode_getNodeIPs_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", nil)
	patch2 := gomonkey.ApplyPrivateMethod(r, "getNodeIPs", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, ErrForMock
	})
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_Reconciler_listCalicoIPPools_List_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "List", ErrForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_Reconciler_listCalicoIPPools_IsIPv4Cidr_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(ip.IsIPv4Cidr, false, ErrForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_Reconciler_listCalicoIPPools_IsIPv6Cidr_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(ip.IsIPv6Cidr, false, ErrForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_Reconciler_reconcileNode_getCalicoIPPools_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", ErrForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_calicoIPPoolV4() []client.Object {
	pool := &calicov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ippool-v4",
		},
		Spec: calicov1.IPPoolSpec{
			// CIDR: "xxx",
			CIDR: "10.10.0.0/18",
		},
	}
	return []client.Object{pool}
}

func mock_calicoIPPoolV6() []client.Object {
	pool := &calicov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ippool-v6",
		},
		Spec: calicov1.IPPoolSpec{
			CIDR: "fdee:120::/120",
		},
	}
	return []client.Object{pool}

}

func mock_IPPoolList() []client.Object {
	calicoIPPoolV4 := &calicov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ippool-v4",
		},
		Spec: calicov1.IPPoolSpec{
			// CIDR: "xxx",
			CIDR: "10.10.0.0/18",
		},
	}
	calicoIPPoolV6 := &calicov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ippool-v6",
		},
		Spec: calicov1.IPPoolSpec{
			CIDR: "fdee:120::/120",
		},
	}
	return []client.Object{calicoIPPoolV4, calicoIPPoolV6}
}

func mock_Reconciler_listNodeIPs_List_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "List", ErrForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_Reconciler_reconcileNode_getNodeIPs_Get_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", ErrForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_Reconciler_checkSomeCniExists_listCalicoIPPools_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(r, "listCalicoIPPools", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, ErrForMock
	})
	return []gomonkey.Patches{*patch1}
}

func mock_Reconciler_checkSomeCniExists_watchSource_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(r, "listCalicoIPPools", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, nil
	})
	patch2 := gomonkey.ApplyFuncReturn(watchSource, ErrForMock)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_Reconciler_checkSomeCniExists_watchSource_succ(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(r, "listCalicoIPPools", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, nil
	})
	patch2 := gomonkey.ApplyFuncReturn(watchSource, nil)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_Reconciler_checkSomeCniExists_getK8sPodCidr_err(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(r, "listCalicoIPPools", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, ErrForMock
	})
	patch2 := gomonkey.ApplyPrivateMethod(r, "getK8sPodCidr", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, ErrForMock
	})
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_Reconciler_checkSomeCniExists_getK8sPodCidr_succ(r *eciReconciler) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(r, "listCalicoIPPools", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, ErrForMock
	})
	patch2 := gomonkey.ApplyPrivateMethod(r, "getK8sPodCidr", func(_ *eciReconciler) (map[string]egressv1.IPListPair, error) {
		return nil, nil
	})
	return []gomonkey.Patches{*patch1, *patch2}
}
