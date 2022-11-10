// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package vxlan

import "context"

type Interface interface {
	Update() error
	Start(ctx context.Context)
}
