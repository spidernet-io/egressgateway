#!/bin/bash

# Copyright 2022 Authors of spidernet-io
# SPDX-License-Identifier: Apache-2.0

set -x

set -o xtrace
set -o errexit
set -o pipefail
set -o nounset

packages=(
)

TARGETARCH="$1"
echo "TARGETARCH=$TARGETARCH"

export DEBIAN_FRONTEND=noninteractive
apt-get update
ln -fs /usr/share/zoneinfo/UTC /etc/localtime
apt-get install -y --no-install-recommends "${packages[@]}"
apt-get purge --auto-remove
apt-get clean
rm -rf /var/lib/apt/lists/*



exit 0
