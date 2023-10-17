#!/bin/bash

## SPDX-License-Identifier: Apache-2.0
## Copyright Authors of Spider

MACVLAN_MASTER_INTERFACE="eth0"
MACVLAN_MULTUS_NAME="macvlan-$MACVLAN_MASTER_INTERFACE"

IPPOOL_IPV4_RANGE="172.18.2.1-172.18.2.254"
IPV4_SUBNET="172.18.0.1/16"
IPV4_GATEWAY="172.18.0.1"

IPPOOL_IPV6_RANGE="fc00:f853:ccd:e793:a::a0-fc00:f853:ccd:e793:a::fe"
IPV6_SUBNET="fc00:f853:ccd:e793::/64"
IPV6_GATEWAY="fc00:f853:ccd:e793::1"

kubectl create ns kube-spiderpool
kubectl label --overwrite ns kube-spiderpool pod-security.kubernetes.io/enforce=privileged

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

createMultusCon() {
cat <<EOF | kubectl apply -f -
apiVersion: spiderpool.spidernet.io/v2beta1
kind: SpiderMultusConfig
metadata:
  name: ${MACVLAN_MULTUS_NAME}
  namespace: kube-spiderpool
spec:
  cniType: macvlan
  enableCoordinator: true
  macvlan:
    master:
    - ${MACVLAN_MASTER_INTERFACE}
EOF
}

createIPPoolIPV4() {
  cat <<EOF | kubectl apply -f -
apiVersion: spiderpool.spidernet.io/v2beta1
kind: SpiderIPPool
metadata:
  name: default-ipv4
spec:
  default: true
  ips:
  - ${IPPOOL_IPV4_RANGE}
  subnet: ${IPV4_SUBNET}
  gateway: ${IPV4_GATEWAY}
  multusName:
  - kube-spiderpool/${MACVLAN_MULTUS_NAME}
EOF
}

createIPPoolIPV6() {
  cat <<EOF | kubectl apply -f -
apiVersion: spiderpool.spidernet.io/v2beta1
kind: SpiderIPPool
metadata:
  name: default-ipv6
spec:
  default: true
  ips:
  - ${IPPOOL_IPV6_RANGE}
  subnet: ${IPV6_SUBNET}
  gateway: ${IPV6_GATEWAY}
  multusName:
  - kube-spiderpool/${MACVLAN_MULTUS_NAME}
EOF
}

helm repo add spiderpool https://spidernet-io.github.io/spiderpool
helm repo update spiderpool

# [ -z "${SPIDERPOOL_VERSION}" ] && SPIDERPOOL_VERSION=$(helm show chart spiderpool/spiderpool | awk '/version/ {print $2}')

SPIDERPOOL_HELM_OPTIONS+=" --wait --debug --set multus.multusCNI.defaultCniCRName=${MACVLAN_MULTUS_NAME} \
  --set global.imageRegistryOverride=${SPIDERPOOL_REGISTRY} "

if [ -n "${SPIDERPOOL_VERSION}" ]; then
  SPIDERPOOL_HELM_OPTIONS+=" --version=${SPIDERPOOL_VERSION} "
else
  SPIDERPOOL_HELM_OPTIONS+="--set spiderpoolAgent.image.tag=latest \
  --set spiderpoolController.image.tag=latest \
  --set spiderpoolInit.image.tag=latest "
fi

helm install spiderpool spiderpool/spiderpool --namespace kube-spiderpool ${SPIDERPOOL_HELM_OPTIONS}
createMultusCon

if [ "$E2E_IP_FAMILY" == "ipv4" ]; then
  createIPPoolIPV4
  SERVER_A_IPV4=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' nettools-server-a)
  SERVER_B_IPV4=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' nettools-server-b)
  hijackCIDR="{\"spec\": {\"hijackCIDR\": [\"${SERVER_A_IPV4}/32\", \"${SERVER_B_IPV4}/32\"]}}"
  kubectl patch spidercoordinators default --type='merge' -p "${hijackCIDR}"
elif [ "$E2E_IP_FAMILY" == "ipv6" ]; then
  createIPPoolIPV6
  SERVER_A_IPV6=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.GlobalIPv6Address}}{{end}}' nettools-server-a)
  SERVER_B_IPV6=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.GlobalIPv6Address}}{{end}}' nettools-server-b)
  hijackCIDR="{\"spec\": {\"hijackCIDR\": [\"${SERVER_A_IPV6}/32\", \"${SERVER_B_IPV6}/32\"]}}"
  kubectl patch spidercoordinators default --type='merge' -p "${hijackCIDR}"
elif [ "$E2E_IP_FAMILY" == "dual" ]; then
  createIPPoolIPV4
  createIPPoolIPV6
  SERVER_A_IPV4=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' nettools-server-a)
  SERVER_B_IPV4=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' nettools-server-b)
  SERVER_A_IPV6=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.GlobalIPv6Address}}{{end}}' nettools-server-a)
  SERVER_B_IPV6=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.GlobalIPv6Address}}{{end}}' nettools-server-b)
  hijackCIDR="{\"spec\": {\"hijackCIDR\": [\"${SERVER_A_IPV4}/32\", \"${SERVER_A_IPV6}/32\", \"${SERVER_B_IPV4}/32\", \"${SERVER_B_IPV6}/32\"]}}"
  kubectl patch spidercoordinators default --type='merge' -p "${hijackCIDR}"
fi