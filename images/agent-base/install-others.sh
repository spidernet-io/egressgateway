#!/bin/bash

# Copyright 2022 Authors of spidernet-io
# SPDX-License-Identifier: Apache-2.0

set -x

set -o xtrace
set -o errexit
set -o pipefail
set -o nounset

packages=(
  # Additional iproute2 runtime dependencies
  libelf1
  libmnl0
  #bash-completion
  iptables
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



#========= verify

# maybe fail to call on building machine
#iptables-legacy --version
#iptables-nft --version
#ip6tables-legacy --version
#ip6tables-nft --version
which iptables-legacy
which iptables-nft
which ip6tables-legacy
which ip6tables-nft


exit 0
