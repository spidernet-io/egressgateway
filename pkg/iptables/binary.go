// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0
//
// Copyright (c) 2017-2022 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package iptables

import (
	"errors"
	"os/exec"
)

// FindBestBinary tries to find an iptables binary for the specific variant (legacy/nftables mode)
// and returns the name of the binary. Falls back on iptables-restore/iptables-save if the specific
// variant isn't available.
func FindBestBinary(lookPath func(file string) (string, error), ipVersion uint8, backendMode, saveOrRestore string) (string, error) {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	verInfix := ""
	if ipVersion == 6 {
		verInfix = "6"
	}
	candidates := []string{
		"ip" + verInfix + "tables-" + backendMode + "-" + saveOrRestore,
		"ip" + verInfix + "tables-" + saveOrRestore,
	}

	for _, candidate := range candidates {
		_, err := lookPath(candidate)
		if err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("failed to find iptables command")
}
