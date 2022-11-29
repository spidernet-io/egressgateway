#!/bin/bash

# Copyright 2017-2020 Authors of Cilium
# SPDX-License-Identifier: Apache-2.0

# usage ./update-golang-image.sh 1.18

set -o xtrace
set -o errexit
set -o pipefail
set -o nounset

CURRENT_DIR_PATH=$(cd `dirname $0`; pwd)
PROJECT_ROOT_PATH="${CURRENT_DIR_PATH}/../.."

if [ "$#" -gt 1 ] ; then
  echo "$0 supports at most 1 argument"
  exit 1
fi

if ! ( uname -a | grep Linux ) ; then
  echo "error, please run on linux, or else the grep syntax is not supported"
  exit 1
fi

go_version=${GO_VERSION:-"1.18.2"}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

root_dir="$(git rev-parse --show-toplevel)"

cd "${root_dir}"


image="docker.io/library/golang:${go_version}"

#image_digest="$("${script_dir}/get-image-digest.sh" "${image}")"
#if [ -z "${image_digest}" ]; then
#  echo "Image digest not available"
#  exit 1
#fi


# shellcheck disable=SC2207
cd $PROJECT_ROOT_PATH
used_by=($(git grep -l GOLANG_IMAGE= ${PROJECT_ROOT_PATH}/images/*/Dockerfile))

for i in "${used_by[@]}" ; do
    # golang images with image digest
    [ ! -f "${i}" ] && echo "error, failed to find ${i} " && exit 1
    #sed "s|GOLANG_IMAGE=docker\.io/library/golang:[0-9][0-9]*\.[0-9][0-9]*\(\.[0-9][0-9]*\)\?@.*|GOLANG_IMAGE=${image}@${image_digest}|" "${i}" > "${i}.sedtmp" && mv ${i}.sedtmp ${i}
    sed "s|GOLANG_IMAGE=docker\.io/library/golang:[0-9][0-9]*\.[0-9][0-9]*\(\.[0-9][0-9]*\)\?|GOLANG_IMAGE=${image}|" "${i}" > "${i}.sedtmp" && mv ${i}.sedtmp ${i}
done

