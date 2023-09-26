#!/bin/bash

## SPDX-License-Identifier: Apache-2.0
## Copyright Authors of Spider

cni_plugin_version="v1.2.0"
cni_url="https://github.com/containernetworking/plugins/releases/download/v1.2.0/cni-plugins-linux-$cni_arch-$cni_plugin_version.tgz"
cni_filename="cni-plugins-linux-$cni_arch-$cni_plugin_version.tgz"

container_list=$(docker ps | grep egressgateway | awk '{print $1}')
for container_id in $container_list; do
  docker exec $container_id mkdir -p /opt/cni/bin
  docker exec $container_id curl -O -L $cni_url
  docker exec $container_id tar -C /opt/cni/bin -xzf $cni_filename
done

if [ -f "weave-daemonset-k8s.yaml" ]; then
    rm -rf "weave-daemonset-k8s.yaml"
fi

if [ -z "${WEAVE_VERSION}" ]; then
  [ -n "${HTTP_PROXY}" ] && weave_tag=$(curl -x "${HTTP_PROXY}" -s https://api.github.com/repos/weaveworks/weave/releases/latest | jq -r '.tag_name')
  [ -z "${HTTP_PROXY}" ] && weave_tag=$(curl -s https://api.github.com/repos/weaveworks/weave/releases/latest | jq -r '.tag_name')
  [ -z "${weave_tag}" ] && { echo "failed get weave version"; exit 1; }
else
  weave_tag=${WEAVE_VERSION}
fi
echo "install weave version ${weave_tag}"

[ -n "${HTTP_PROXY}" ] && wget wget --proxy=${HTTP_PROXY} https://github.com/weaveworks/weave/releases/download/${weave_tag}/weave-daemonset-k8s.yaml
[ -z "${HTTP_PROXY}" ] && wget https://github.com/weaveworks/weave/releases/download/${weave_tag}/weave-daemonset-k8s.yaml

if [ "$E2E_IP_FAMILY" == "ipv4" ]; then
  if grep -q "IPALLOC_RANGE" < weave-daemonset-k8s.yaml; then
    yq -i '.items[5].spec.template.spec.containers[0].env.[] |= (select(.name == "IPALLOC_RANGE") | .value = strenv(E2E_KIND_IPV4_POD_CIDR))' weave-daemonset-k8s.yaml
  else
    yq -i '.items[5].spec.template.spec.containers[0].env += [{"name": "IPALLOC_RANGE", "value": strenv(E2E_KIND_IPV4_POD_CIDR)}]' weave-daemonset-k8s.yaml
  fi
  kubectl apply -f weave-daemonset-k8s.yaml
  kubectl wait --for=condition=Ready pod -l name=weave-net -A
elif [ "$E2E_IP_FAMILY" == "ipv6" ]; then
  echo "weave not support ipv6"
  exit 1
elif [ "$E2E_IP_FAMILY" == "dual" ]; then
  echo "weave not support dual"
  exit 1
fi
