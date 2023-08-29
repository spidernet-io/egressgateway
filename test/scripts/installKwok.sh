#!/bin/bash
## SPDX-License-Identifier: Apache-2.0
## Copyright Authors of Spider

set -o errexit -o nounset -o pipefail
set -x

# KWOK repository
KWOK_REPO=kubernetes-sigs/kwok

# Deployment kwok and set up CRDs
curl -x "${https_proxy}" -L https://github.com/${KWOK_REPO}/releases/download/${KWOK_VERSION}/kwok.yaml | kubectl --kubeconfig ${E2E_KIND_KUBECONFIG_PATH} apply -f - || { echo failed install kwok; exit 1; }

# Set up default CRs of Stages
curl -x "${https_proxy}" -L https://github.com/${KWOK_REPO}/releases/download/${KWOK_VERSION}/stage-fast.yaml | kubectl --kubeconfig ${E2E_KIND_KUBECONFIG_PATH} apply -f - || { echo failed install kwok; exit 1; }

# Wait pod running
kubectl --kubeconfig ${E2E_KIND_KUBECONFIG_PATH} wait --for=condition=ready -l app=kwok-controller --timeout=${INSTALL_TIME_OUT} pod -n kube-system
kubectl --kubeconfig ${E2E_KIND_KUBECONFIG_PATH} get po -l app=kwok-controller -n kube-system

echo -e "\033[35m Succeed to install KWOK \033[0m"