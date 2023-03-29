#!/bin/bash
## SPDX-License-Identifier: Apache-2.0
## Copyright Authors of Spider

set -x

TCP_HELLO="TCP Server Say hello"
UDP_HELLO="UDP Server Say hello"
WEB_HELLO="WebSocket Server Say hello"

if [[ ${E2E_IP_FAMILY} == "ipv4" ]]; then SERVER_IP=$(docker inspect ${NETTOOLS_SERVER} | grep -w IPAddress | grep -E -o '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | tr -d '\n'); fi
if [ "${E2E_IP_FAMILY}" == "ipv6" ] || [ "${E2E_IP_FAMILY}" == "dual" ]; then
  v6=$(docker inspect "${NETTOOLS_SERVER}" | grep -w GlobalIPv6Address  | sed 1d | awk '{print $2}' | tr -d '",' | tr -d '\n')
  SERVER_IP=[${v6}]
fi
RESULT=$(mktemp)

"${CLIENT}" -addr "${SERVER_IP}" -protocol all -tcpPort "${TCP_PORT}"  -udpPort "${UDP_PORT}" -webPort "${WEB_PORT}" > "${RESULT}" 2>&1 &

server="bad"

for i in {0..10}; do
  if grep -e "${TCP_HELLO}" -e "${UDP_HELLO}" -e "${WEB_HELLO}"  "${RESULT}"; then
      echo "server is ok"
      server="ok"
      break
  else
    echo "some connect not ready, wait..."
    sleep 2s
  fi
done

if [[ ${server} == "bad" ]]; then echo time out to wait server ready; exit 1; fi
if [[ ${server} == "ok" ]]; then
  echo server is ok, delete the test-client process
  ps=$(pgrep -f "${CLIENT}" | tr '\n' ' ')
  if [[ -n $ps ]]; then
    for p in $ps; do
      set +m kill "$p" 2>&1 >/dev/null
    done
  fi
fi
