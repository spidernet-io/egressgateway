// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package types

type EgressGatewayManager interface {
	RunWebhookServer(webhookPort int, tlsDir string)
	RunInformer(leaseName, leaseNameSpace string, leaseId string)
}
