#!/bin/bash
## SPDX-License-Identifier: Apache-2.0
## Copyright Authors of Spider

checkNettoolsServer() {
    containerName=$1

    echo "E2E_IP_FAMILY=$E2E_IP_FAMILY"

    if [[ ${E2E_IP_FAMILY} == "ipv4" ]] || [[ ${E2E_IP_FAMILY} == "dual" ]]; then
        serverIPv4=$(docker inspect "$containerName" -f '{{.NetworkSettings.Networks.kind.IPAddress}}')
        if [[ -z "$serverIPv4" ]]; then
            echo "Error: IPv4 address not found for container: $containerName"
            exit 1
        else
            echo "find IPv4 address: $serverIPv4"
            eip=$(ip r get "$serverIPv4" | awk '/src/ { print $5 }')
            "${CLIENT}" -addr "${serverIPv4}" -protocol all -tcpPort "${TCP_PORT}" -udpPort "${UDP_PORT}" -webPort "${WEB_PORT}" -eip $eip -batch true
            if [ $? -ne 0 ]; then
                print_error "failed to check net tool server: $containerName"
                exit 1
            fi
            print_green "success to check net tools server: $containerName"
        fi
    fi

    if [[ ${E2E_IP_FAMILY} == "ipv6" ]] || [[ ${E2E_IP_FAMILY} == "dual" ]]; then
        serverIPv6=$(docker inspect "$containerName" -f '{{.NetworkSettings.Networks.kind.GlobalIPv6Address}}')
        if [[ -z "$serverIPv6" ]]; then
            echo "Error: IPv6 address not found for container: $containerName"
            exit 1
        else
            echo "find IPv6 address: $serverIPv6"
            eip=$(ip r get "$serverIPv6" | awk '{for(i=1;i<=NF;i++) if ($i=="src") print $(i+1)}')

            "${CLIENT}" -addr "${serverIPv6}" -protocol all -tcpPort "${TCP_PORT}" -udpPort "${UDP_PORT}" -webPort "${WEB_PORT}" -eip $eip -batch true
            if [ $? -ne 0 ]; then
                print_error "failed to check net tool server: $containerName"
                exit 1
            fi
            print_green "success to check net tools server: $containerName"
        fi
    fi
}

print_green() {
  echo -e "\033[0;32m$1\033[0m"
}

print_error() {
  echo -e "\033[0;31m$1\033[0m"
}

checkNettoolsServer ${NETTOOLS_SERVER_A}
checkNettoolsServer ${NETTOOLS_SERVER_B}
