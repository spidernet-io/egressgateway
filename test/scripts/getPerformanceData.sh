#!/bin/bash

# Copyright 2022 Authors of spidernet-io
# SPDX-License-Identifier: Apache-2.0

CURRENT_DIR_PATH=$( dirname $0 )
CURRENT_DIR_PATH=$(cd ${CURRENT_DIR_PATH} ; pwd)
PROJECT_ROOT_PATH=$(cd ${CURRENT_DIR_PATH}/../.. ; pwd)

E2E_REPORT_PATH="$1"
if [ ! -f "$E2E_REPORT_PATH" ] ; then
    echo "error, no file $E2E_REPORT_PATH " >&2
    exit 1
fi
