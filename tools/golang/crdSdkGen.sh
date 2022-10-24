#!/bin/bash

# Copyright 2022 Authors of spidernet-io
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail


APIS_PKG="pkg/k8s/apis"
OUTPUT_PKG="pkg/k8s/client"
GROUPS_WITH_VERSIONS="egressgateway.spidernet.io:v1"

#===================

PROJECT_ROOT=$(git rev-parse --show-toplevel)
CODEGEN_PKG=${CODEGEN_PKG_PATH:-$(cd ${PROJECT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}
MODULE_NAME=$(cat ${PROJECT_ROOT}/go.mod | grep -e "module[[:space:]][^[:space:]]*" | awk '{print $2}')

SPDX_COPYRIGHT_HEADER="${PROJECT_ROOT}/tools/copyright-header.txt"

TMP_DIR="${PROJECT_ROOT}/output/codeGen"
LICENSE_FILE="${TMP_DIR}/boilerplate.go.txt"
GO_PATH_DIR="${TMP_DIR}/go"

rm -rf ${TMP_DIR}
mkdir -p ${TMP_DIR}

touch ${LICENSE_FILE}
while read -r line || [[ -n ${line} ]]
do
    echo "// ${line}" >>${LICENSE_FILE}
done < ${SPDX_COPYRIGHT_HEADER}

cd "${PROJECT_ROOT}"
GO_PKG_DIR=$(dirname "${GO_PATH_DIR}/src/${MODULE_NAME}")
mkdir -p "${GO_PKG_DIR}"

if [[ ! -e "${GO_PKG_DIR}" || "$(readlink "${GO_PKG_DIR}")" != "${PROJECT_ROOT}" ]]; then
  ln -snf "${PROJECT_ROOT}" "${GO_PKG_DIR}"
fi
rm -rf ${OUTPUT_PKG} || true

export GOPATH="${GO_PATH_DIR}"
bash ${PROJECT_ROOT}/${CODEGEN_PKG}/generate-groups.sh "client,informer,lister" \
  ${MODULE_NAME}/${OUTPUT_PKG} \
  ${MODULE_NAME}/${APIS_PKG} \
  ${GROUPS_WITH_VERSIONS} \
  --go-header-file ${LICENSE_FILE} -v 10
(($?!=0)) && echo "error, failed to generate crd sdk" && exit 1

rm -rf ${TMP_DIR}
exit 0
