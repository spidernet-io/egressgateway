#!/bin/bash

# Copyright 2022 Authors of spidernet-io
# SPDX-License-Identifier: Apache-2.0

set -o xtrace
set -o errexit
set -o pipefail
set -o nounset

update-alternatives \
  --install /usr/sbin/iptables iptables /usr/sbin/iptables-wrapper 100 \
  --slave /usr/sbin/iptables-restore iptables-restore /usr/sbin/iptables-wrapper \
  --slave /usr/sbin/iptables-save iptables-save /usr/sbin/iptables-wrapper

update-alternatives \
  --install /usr/sbin/ip6tables ip6tables /usr/sbin/iptables-wrapper 100 \
  --slave /usr/sbin/ip6tables-restore ip6tables-restore /usr/sbin/iptables-wrapper \
  --slave /usr/sbin/ip6tables-save ip6tables-save /usr/sbin/iptables-wrapper

chmod +x /usr/sbin/iptables
chmod +x /usr/sbin/iptables-restore
chmod +x /usr/sbin/iptables-save
chmod +x /usr/sbin/ip6tables
chmod +x /usr/sbin/ip6tables-restore
chmod +x /usr/sbin/ip6tables-save
chmod +x /usr/sbin/iptables-wrapper

