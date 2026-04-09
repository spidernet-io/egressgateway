#!/bin/bash
## SPDX-License-Identifier: Apache-2.0
## Copyright Authors of Spider

set -o errexit -o pipefail

CALICO_VERSION=${CALICO_VERSION:-v3.31.4}
E2E_KIND_IPV4_POD_CIDR=${E2E_KIND_IPV4_POD_CIDR:-172.40.0.0/16}
E2E_KIND_IPV6_POD_CIDR=${E2E_KIND_IPV6_POD_CIDR:-fd40::/48}

CURRENT_DIR_PATH=$(cd $(dirname $0); pwd)
PROJECT_ROOT_PATH=$(cd ${CURRENT_DIR_PATH}/../.. && pwd)

DEST_DIR=${PROJECT_ROOT_PATH}/test/.tmp
rm -rf ${DEST_DIR}
mkdir -p ${DEST_DIR}

OPERATOR_YAML=${DEST_DIR}/tigera-operator.yaml

echo "install calico version ${CALICO_VERSION} via tigera operator"

# download tigera-operator.yaml
CURL_PROXY_ARGS=""
[ -n "${HTTP_PROXY}" ] && CURL_PROXY_ARGS="-x ${HTTP_PROXY}"
echo "downloading tigera-operator.yaml for calico ${CALICO_VERSION}..."
curl --retry 3 --retry-delay 5 ${CURL_PROXY_ARGS} -Lo ${OPERATOR_YAML} \
    https://raw.githubusercontent.com/projectcalico/calico/${CALICO_VERSION}/manifests/tigera-operator.yaml

export KUBECONFIG=${E2E_KIND_KUBECONFIG_PATH}

echo "creating tigera-operator..."
kubectl create -f ${OPERATOR_YAML}

# wait for tigera-operator deployment to be available
echo "waiting for tigera-operator to be available..."
kubectl wait --for=condition=available --timeout=${INSTALL_TIME_OUT} deploy/tigera-operator -n tigera-operator

# wait for FelixConfiguration CRD to be established before creating resources
echo "waiting for FelixConfiguration CRD to appear..."
for i in $(seq 1 120); do
  kubectl get crd felixconfigurations.crd.projectcalico.org > /dev/null 2>&1 && break
  echo "  ${i}s: CRD not found yet..."
  sleep 1
done

# set chainInsertMode=Append upfront so calico-node picks it up on first start
echo "applying FelixConfiguration chainInsertMode=Append..."
kubectl apply -f - <<EOF
apiVersion: crd.projectcalico.org/v1
kind: FelixConfiguration
metadata:
  name: default
spec:
  chainInsertMode: Append
EOF

# create Installation CR based on IP family
echo "creating Installation CR for ip family: ${E2E_IP_FAMILY}..."
case ${E2E_IP_FAMILY} in
  ipv4)
    cat <<EOF | kubectl create -f -
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  calicoNetwork:
    bgp: Disabled
    nodeAddressAutodetectionV4:
      interface: eth0  
    ipPools:
    - blockSize: 26
      cidr: ${E2E_KIND_IPV4_POD_CIDR}
      encapsulation: VXLAN
      natOutgoing: Enabled
      nodeSelector: all()
EOF
    ;;
  ipv6)
    cat <<EOF | kubectl create -f -
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  calicoNetwork:
    bgp: Disabled
    nodeAddressAutodetectionV4:
      interface: eth0
    nodeAddressAutodetectionV6:
      interface: eth0      
    ipPools:
    - blockSize: 122
      cidr: ${E2E_KIND_IPV6_POD_CIDR}
      encapsulation: VXLAN
      natOutgoing: Enabled
      nodeSelector: all()
EOF
    ;;
  dual)
    cat <<EOF | kubectl create -f -
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  calicoNetwork:
    bgp: Disabled
    nodeAddressAutodetectionV4:
      interface: eth0  
    nodeAddressAutodetectionV6:
      interface: eth0
    ipPools:
    - blockSize: 26
      cidr: ${E2E_KIND_IPV4_POD_CIDR}
      encapsulation: VXLAN
      natOutgoing: Enabled
      nodeSelector: all()
    - blockSize: 122
      cidr: ${E2E_KIND_IPV6_POD_CIDR}
      encapsulation: VXLAN
      natOutgoing: Enabled
      nodeSelector: all()
EOF
    ;;
  *)
    echo "the value of E2E_IP_FAMILY: ipv4 or ipv6 or dual"
    exit 1
    ;;
esac

# wait for calico-node pods (operator installs them in calico-system)
echo "waiting for calico-node pods to be created..."
for i in $(seq 1 120); do
  COUNT=$(kubectl get pod -n calico-system -l k8s-app=calico-node --no-headers 2>/dev/null | wc -l)
  [ "${COUNT}" -gt 0 ] && break
  echo "  ${i}s: no calico-node pods yet..."
  sleep 1
done


echo "waiting for calico-node pods to be ready..."
kubectl wait --for=condition=ready -l k8s-app=calico-node --timeout=${INSTALL_TIME_OUT} pod -n calico-system
kubectl get po -n calico-system

echo -e "Succeed to install Calico"

rm -rf ${DEST_DIR}
