// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils

import "errors"

var (
	ErrInvalidIPVersion     = errors.New("invalid IP version")
	ErrInvalidIPRangeFormat = errors.New("invalid IP range format")
	ErrInvalidIPFormat      = errors.New("invalid IP format")
	ErrInvalidCIDRFormat    = errors.New("invalid CIDR format")
	ErrInvalidRouteFormat   = errors.New("invalid route format")
	ErrInvalidIP            = errors.New("invalid IP")
)
