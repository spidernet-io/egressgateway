#!/bin/bash
## SPDX-License-Identifier: Apache-2.0
## Copyright Authors of Spider

set -x
set -o errexit
set -o pipefail
set -o nounset

if ! command -v cilium &> /dev/null; then
    CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
    CLI_ARCH=amd64
    if [ "$(uname -m)" = "aarch64" ]; then CLI_ARCH=arm64; fi
    curl -L --fail --remote-name-all https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}
    sha256sum --check cilium-linux-${CLI_ARCH}.tar.gz.sha256sum
    sudo tar xzvfC cilium-linux-${CLI_ARCH}.tar.gz /usr/local/bin
    rm cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}
fi

echo $E2E_KIND_IPV4_POD_CIDR
echo $E2E_KIND_IPV6_POD_CIDR

case "${E2E_IP_FAMILY}" in
  ipv4)
    cilium install --wait --set enable-ipv4=true --set ipv4NativeRoutingCIDR=$E2E_KIND_IPV4_POD_CIDR --set autoDirectNodeRoutes=true --set routingMode="native" --set bpf.masquerade=false
    ;;
  ipv6)
    cilium install --wait --set enable-ipv6=true --set ipv6NativeRoutingCIDR=$E2E_KIND_IPV6_POD_CIDR --set autoDirectNodeRoutes=true --set routingMode="native" --set bpf.masquerade=false
    ;;
  dual)
    cilium install --wait --set enable-ipv4=true --set enable-ipv6=true --set ipv4NativeRoutingCIDR=$E2E_KIND_IPV4_POD_CIDR --set ipv6NativeRoutingCIDR=$E2E_KIND_IPV6_POD_CIDR --set autoDirectNodeRoutes=true --set routingMode="native" --set bpf.masquerade=false
    ;;
  *)
    echo "Invalid value for E2E_IP_FAMILY: ${E2E_IP_FAMILY}. Expected 'ipv4', 'ipv6', or 'dual'."
    exit 1
    ;;
esac