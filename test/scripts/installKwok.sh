#!/bin/bash
## SPDX-License-Identifier: Apache-2.0
## Copyright Authors of Spider

set -o errexit -o nounset -o pipefail

# Temporary directory
KWOK_WORK_DIR=$(mktemp -d)
# KWOK repository
KWOK_REPO=kubernetes-sigs/kwok
# Get latest
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')

cat <<EOF > "${KWOK_WORK_DIR}/kustomization.yaml"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
- name: registry.k8s.io/kwok/kwok
  newTag: "${KWOK_LATEST_RELEASE}"
resources:
- "https://github.com/${KWOK_REPO}/kustomize/kwok?ref=${KWOK_LATEST_RELEASE}"
EOF

kubectl --kubeconfig ${E2E_KIND_KUBECONFIG_PATH} kustomize "${KWOK_WORK_DIR}" > "${KWOK_WORK_DIR}/kwok.yaml"

kubectl --kubeconfig ${E2E_KIND_KUBECONFIG_PATH} apply -f "${KWOK_WORK_DIR}/kwok.yaml"

kubectl --kubeconfig ${E2E_KIND_KUBECONFIG_PATH} wait --for=condition=ready -l app=kwok-controller --timeout=${INSTALL_TIME_OUT} pod -n kube-system
kubectl --kubeconfig ${E2E_KIND_KUBECONFIG_PATH} get po -l app=kwok-controller -n kube-system

echo -e "\033[35m Succeed to install KWOK \033[0m"
rm -rf ${KWOK_WORK_DIR}/kustomization.yaml || exit 0
