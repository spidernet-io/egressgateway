#!/bin/bash

# Copyright 2017-2021 Authors of Cilium
# SPDX-License-Identifier: Apache-2.0

set -o xtrace
set -o errexit
set -o pipefail
set -o nounset

# https://github.com/google/gops
gops_version="v0.3.27"

mkdir -p /go/src/github.com/google
cd /go/src/github.com/google

export GOPROXY="https://goproxy.io|https://goproxy.cn|direct"

git clone https://github.com/google/gops.git
cd gops

git checkout -b "${gops_version}" "${gops_version}"
git --no-pager remote -v
git --no-pager log -1

for arch in amd64 arm64 ; do
  mkdir -p "/out/linux/${arch}/bin"
  GOARCH="${arch}" CGO_ENABLED=0 go build -ldflags "-s -w" -o "/out/linux/${arch}/bin/gops" github.com/google/gops
done

x86_64-linux-gnu-strip /out/linux/amd64/bin/gops
aarch64-linux-gnu-strip /out/linux/arm64/bin/gops
