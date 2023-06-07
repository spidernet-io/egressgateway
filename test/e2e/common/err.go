// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import "errors"

var (
	INVALID_INPUT = errors.New("invalid input")
	ERR_IP_FORMAT = errors.New("invalid ip format")
	ERR_CHECK_EIP = errors.New("failed to check eip")
	ERR_TIMEOUT   = errors.New("time out")
)
