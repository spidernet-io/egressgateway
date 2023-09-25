#!/bin/bash

## SPDX-License-Identifier: Apache-2.0
## Copyright Authors of Spider

kubectl create ns kube-flannel
kubectl label --overwrite ns kube-flannel pod-security.kubernetes.io/enforce=privileged

host_arch=$(uname -m)
case "$host_arch" in
  "x86_64")
    cni_arch="amd64"
    ;;
  "aarch64")
    cni_arch="arm64"
    ;;
  *)
    echo "Unsupported host architecture: $host_arch"
    exit 1
    ;;
esac

cni_plugin_version="v1.2.0"
cni_url="https://github.com/containernetworking/plugins/releases/download/v1.2.0/cni-plugins-linux-$cni_arch-$cni_plugin_version.tgz"
cni_filename="cni-plugins-linux-$cni_arch-$cni_plugin_version.tgz"

container_list=$(docker ps | grep egressgateway | awk '{print $1}')
for container_id in $container_list; do
  docker exec $container_id mkdir -p /opt/cni/bin
  docker exec $container_id curl -O -L $cni_url
  docker exec $container_id tar -C /opt/cni/bin -xzf $cni_filename
done

helm repo add flannel https://flannel-io.github.io/flannel/
helm repo update

if [ "$E2E_IP_FAMILY" == "ipv4" ]; then
  helm install flannel --set podCidr="172.40.0.0/16" --namespace kube-flannel flannel/flannel --wait --debug
elif [ "$E2E_IP_FAMILY" == "ipv6" ]; then
  helm install flannel --set podCidrv6="fd40::/48" --namespace kube-flannel flannel/flannel --wait --debug
elif [ "$E2E_IP_FAMILY" == "dual" ]; then
  helm install flannel --set podCidr="172.40.0.0/16" --set podCidrv6="fd40::/48" --namespace kube-flannel flannel/flannel --wait --debug
fi
