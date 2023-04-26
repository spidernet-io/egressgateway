// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"os"
	"testing"

	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	c "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	mock "github.com/spidernet-io/egressgateway/pkg/mock/controller-runtime"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

var (
	mockCtrl         *gomock.Controller
	err              error
	cfg              *config.Config
	log              *zap.Logger
	egressIgnoreCIDR config.EgressIgnoreCIDR
	fakeClient       c.Client
	mockManager      *mock.MockManager
)

var (
	ctx    context.Context
	cancel context.CancelFunc
)

var ERR_FAILED_NEW_CONTROLLER = errors.New("failed to New controller")

func TestController(t *testing.T) {
	mockCtrl = gomock.NewController(t)
	defer mockCtrl.Finish()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeEach(func() {
	egressIgnoreCIDR = config.EgressIgnoreCIDR{
		AutoDetect: config.AutoDetect{
			PodCIDR:   "calico",
			ClusterIP: true,
			NodeIP:    true,
		},
	}

	cfg = &config.Config{
		FileConfig: config.FileConfig{
			EgressIgnoreCIDR: egressIgnoreCIDR,
		},
		KubeConfig: &rest.Config{},
	}

	log = logger.NewStdoutLogger(os.Getenv("LOG_LEVEL"))

	fakeClient = fake.NewClientBuilder().WithScheme(schema.GetScheme()).Build()

	mockManager = mock.NewMockManager(mockCtrl)
	mockManager.EXPECT().GetClient().Return(fakeClient)
})
