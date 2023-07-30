// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package ip

import "errors"

var (
	ErrInvalidIPVersion     = errors.New("invalid IP version")
	ErrInvalidIPRangeFormat = errors.New("invalid IP range format")
	ErrInvalidIP            = errors.New("invalid IP")
)
