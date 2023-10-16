#!/usr/bin/env bash

# Copyright 2022 Authors of spidernet-io
# SPDX-License-Identifier: Apache-2.0

CURRENT_FILENAME=`basename $0`
CURRENT_DIR_PATH=$(cd `dirname $0`; pwd)
GINKGO_PKG_PATH=${GINKGO_PKG_PATH:-${CURRENT_DIR_PATH}/../../vendor/github.com/onsi/ginkgo/v2/ginkgo/main.go}

# debug
# git branch
# git show -s --format='format:%H'

export SERVER_A_IPV4=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' nettools-server-a)
export SERVER_A_IPV6=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.GlobalIPv6Address}}{{end}}' nettools-server-a)
export SERVER_B_IPV4=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' nettools-server-b)
export SERVER_B_IPV6=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.GlobalIPv6Address}}{{end}}' nettools-server-b)
echo $SERVER_A_IPV4
echo $SERVER_A_IPV6
echo $SERVER_B_IPV4
echo $SERVER_B_IPV6
echo $IMAGE

if which ginkgo &>/dev/null ; then
  echo "find ginkgo cli"
  echo -e "\e[32m[CMD] ginkgo $@\e[0m"
  ginkgo $@
elif [ -f "$GINKGO_PKG_PATH" ] ; then
  go run $GINKGO_PKG_PATH $@
else
  echo "failed to find ginkgo vendor $GINKGO_PKG_PATH "
  exit 1
fi
