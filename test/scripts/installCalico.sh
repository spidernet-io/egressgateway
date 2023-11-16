#!/bin/bash
## SPDX-License-Identifier: Apache-2.0
## Copyright Authors of Spider

set -o errexit -o pipefail

set -x

OS=$(uname | tr 'A-Z' 'a-z')
SED_COMMAND="sed"
if [ ${OS} == "darwin" ]; then SED_COMMAND=gsed ; fi

CURRENT_DIR_PATH=$(cd $(dirname $0); pwd)
PROJECT_ROOT_PATH=$( cd ${CURRENT_DIR_PATH}/../.. && pwd )

DEST_CALICO_YAML_DIR=${PROJECT_ROOT_PATH}/test/.tmp
rm -rf ${DEST_CALICO_YAML_DIR}
mkdir -p ${DEST_CALICO_YAML_DIR}

CALICO_YAML=${DEST_CALICO_YAML_DIR}/calico.yaml
CALICO_CONFIG=${DEST_CALICO_YAML_DIR}/calico_config.yaml
CALICO_NODE=${DEST_CALICO_YAML_DIR}/calico_node.yaml

# CALICO_VERSION
if [ -z "${CALICO_VERSION}" ]; then
  [ -n "${HTTP_PROXY}" ] && calico_tag=$(curl --retry 3 --retry-delay 5 -x "${HTTP_PROXY}" -s https://api.github.com/repos/projectcalico/calico/releases/latest | jq -r '.tag_name')
  [ -z "${HTTP_PROXY}" ] && calico_tag=$(curl --retry 3 --retry-delay 5 -s https://api.github.com/repos/projectcalico/calico/releases/latest | jq -r '.tag_name')
  [ "${calico_tag}" == "null" ] && { echo "failed get calico version"; exit 1; }
else
  calico_tag=${CALICO_VERSION}
fi
echo "install calico version ${calico_tag}"

[ -n "${HTTP_PROXY}" ] && curl --retry 3 --retry-delay 5 -x "${HTTP_PROXY}" -Lo ${CALICO_YAML}  https://raw.githubusercontent.com/projectcalico/calico/${calico_tag}/manifests/calico.yaml
[ -z "${HTTP_PROXY}" ] && curl --retry 3 --retry-delay 5 -Lo ${CALICO_YAML}  https://raw.githubusercontent.com/projectcalico/calico/${calico_tag}/manifests/calico.yaml

# set registry
if [ -n "${CALICO_REGISTRY}" ]; then
  grep -q -e ".*image:.*docker.io" ${CALICO_YAML} || { echo "failed find image"; exit 1; }
  ${SED_COMMAND} -i -E 's?(.*image:.*)(docker.io)(.*)?\1'"${CALICO_REGISTRY}"'\3?g' ${CALICO_YAML}
fi

# accelerate local cluster , in case that it times out to wait calico ready
IMAGE_LIST=`cat ${CALICO_YAML} | grep "image: " | awk '{print \$2}' | sort  | uniq  | tr '\n' ' ' | tr '\r' ' ' `
echo "image: ${IMAGE_LIST}"
for IMAGE in ${IMAGE_LIST} ; do
    echo "load calico image ${IMAGE} to kind cluster"
    docker pull ${IMAGE}
    kind load docker-image ${IMAGE} --name ${KIND_CLUSTER_NAME}
done

export KUBECONFIG=${E2E_KIND_KUBECONFIG_PATH}

kubectl apply -f  ${CALICO_YAML}
sleep 3

kubectl wait --for=condition=ready -l k8s-app=calico-node --timeout=${INSTALL_TIME_OUT} pod -n kube-system
kubectl get po -n kube-system

echo -e "\033[35m Succeed to install Calico \033[0m"

echo -e "\033[35m Patch Calico \033[0m"

kubectl -n kube-system get cm calico-config -oyaml > ${CALICO_CONFIG}
kubectl -n kube-system get ds calico-node -oyaml > ${CALICO_NODE}

case ${E2E_IP_FAMILY} in
  ipv4)
    # set configmap
    configYaml=$(yq '.data.cni_network_config' ${CALICO_CONFIG} | yq '.plugins[0].ipam = {"type": "calico-ipam", "assign_ipv4": "true", "assign_ipv6": "false"}' --output-format=json)
    configYaml=$configYaml yq e '.data.cni_network_config |= strenv(configYaml)' -i ${CALICO_CONFIG}
    ${SED_COMMAND} -i 's/"mtu": "__CNI_MTU__"/"mtu": __CNI_MTU__/g' ${CALICO_CONFIG}
    kubectl -n kube-system patch cm calico-config --patch "$(cat ${CALICO_CONFIG})" || { echo "failed to patch calico configmap"; exit 1; }
    ;;
  ipv6)
    # set configmap
    configYaml=$(yq '.data.cni_network_config' ${CALICO_CONFIG} | yq '.plugins[0].ipam = {"type": "calico-ipam", "assign_ipv4": "false", "assign_ipv6": "true"}' --output-format=json)
    configYaml=$configYaml yq e '.data.cni_network_config |= strenv(configYaml)' -i ${CALICO_CONFIG}
    ${SED_COMMAND} -i 's/"mtu": "__CNI_MTU__"/"mtu": __CNI_MTU__/g' ${CALICO_CONFIG}
    kubectl -n kube-system patch cm calico-config --patch "$(cat ${CALICO_CONFIG})" || { echo "failed to patch calico configmap"; exit 1; }

    # set calico-node env
    grep -q "FELIX_IPV6SUPPORT" ${CALICO_NODE} || { echo "failed find FELIX_IPV6SUPPORT"; exit 1; }
    ${SED_COMMAND} -i -E '/FELIX_IPV6SUPPORT/{n;s/value: "false"/value: "true"/}' ${CALICO_NODE}

    grep -q "value: autodetect" ${CALICO_NODE} || { echo "failed find autodetect"; exit 1; }
    ${SED_COMMAND} -i '/value: autodetect/a\        - name: IP6\n\          value: autodetect' ${CALICO_NODE}
    kubectl -n kube-system patch ds calico-node --patch "$(cat ${CALICO_NODE})" || { echo "failed to patch calico-node"; exit 1; }
    ;;
  dual)
    # set configmap
    configYaml=$(yq '.data.cni_network_config' ${CALICO_CONFIG} | yq '.plugins[0].ipam = {"type": "calico-ipam", "assign_ipv4": "true", "assign_ipv6": "true"}' --output-format=json)
    configYaml=$configYaml yq e '.data.cni_network_config |= strenv(configYaml)' -i ${CALICO_CONFIG}
    ${SED_COMMAND} -i 's/"mtu": "__CNI_MTU__"/"mtu": __CNI_MTU__/g' ${CALICO_CONFIG}
    kubectl -n kube-system patch cm calico-config --patch "$(cat ${CALICO_CONFIG})" || { echo "failed to patch calico configmap"; exit 1; }

    # set calico-node env
    grep -q "FELIX_IPV6SUPPORT" ${CALICO_NODE} || { echo "failed find FELIX_IPV6SUPPORT"; exit 1; }
    ${SED_COMMAND} -i -E '/FELIX_IPV6SUPPORT/{n;s/value: "false"/value: "true"/}' ${CALICO_NODE}
    grep -q "value: autodetect" ${CALICO_NODE} || { echo "failed find autodetect"; exit 1; }
    ${SED_COMMAND} -i '/value: autodetect/a\        - name: IP6\n\          value: autodetect' ${CALICO_NODE}
    kubectl -n kube-system patch ds calico-node --patch "$(cat ${CALICO_NODE})" || { echo "failed to patch calico-node"; exit 1; }
    ;;
  *)
    echo "the value of E2E_IP_FAMILY: ipv4 or ipv6 or dual"
    exit 1
esac

kubectl patch felixconfigurations.crd.projectcalico.org default --type='merge' -p '{"spec":{"chainInsertMode":"Append"}}' || { echo "failed to patch calico chainInsertMode"; exit 1; }
if [[ ${E2E_IP_FAMILY} != "ipv4" ]]; then
  ok=no
  for i in {1..10}; do
    echo "${i}"s
    sleep 1;
    if kubectl get ippools default-ipv6-ippool > /dev/null 2>&1; then
      ok=yes; break;
    else
      continue;
    fi
  done
  if [[ ${ok} == "no" ]]; then
    echo "time out to wait default-ipv6-ippool"; exit 1;
  fi
  kubectl patch ippools default-ipv6-ippool --type='merge' -p '{"spec":{"natOutgoing":true}}' || { echo "failed to patch calico natOutgoing"; exit 1; }
fi

# restart calico pod
kubectl -n kube-system delete pod -l k8s-app=calico-node --force --grace-period=0
sleep 3
kubectl wait --for=condition=ready -l k8s-app=calico-node --timeout=${INSTALL_TIME_OUT} pod -n kube-system

kubectl -n kube-system delete pod -l k8s-app=calico-kube-controllers --force --grace-period=0
sleep 3
kubectl wait --for=condition=ready -l k8s-app=calico-kube-controllers --timeout=${INSTALL_TIME_OUT} pod -n kube-system

rm -rf ${DEST_CALICO_YAML_DIR}

