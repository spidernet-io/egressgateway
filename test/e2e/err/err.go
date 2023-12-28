// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package err

import (
	"errors"
)

var (
	ErrTimeout               = errors.New("error timeout")
	ErrWaitNodeOnTimeout     = errors.New("timeout waiting node to be ready timeout")
	ErrWaitPodRunningTimeout = errors.New("timeout waiting for pod running")
)
