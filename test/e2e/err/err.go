// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package err

import "errors"

var (
	EMPTY_INPUT           = errors.New("empty input")
	TIME_OUT              = errors.New("time out")
	NOT_FOUND             = errors.New("not found")
	IPVERSION_ERR         = errors.New("error, not ipv4 and ipv6")
	ErrInvalidPodCidrMode = errors.New("invalid podCidrMode")
)
