// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressGatewayManager

import (
	"github.com/spidernet-io/egressgateway/pkg/egressGatewayManager/types"
	"go.uber.org/zap"
)

type mybookManager struct {
	logger   *zap.Logger
	webhook  *webhookhander
	informer *informerHandler
}

var _ types.EgressGatewayManager = (*mybookManager)(nil)

func New(logger *zap.Logger) types.EgressGatewayManager {
	return &mybookManager{
		logger: logger.Named("EgressGatewayManager"),
	}
}
