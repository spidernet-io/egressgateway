# docker buildx build -t testbase:latest --platform=linux/arm64,linux/amd64 --output type=docker .

# Copyright 2022 Authors of spidernet-io
# SPDX-License-Identifier: Apache-2.0

ARG GOLANG_IMAGE=docker.io/library/golang:1.19
ARG UBUNTU_IMAGE=docker.io/library/ubuntu:22.04

#========= build gops ===============
FROM --platform=${TARGETPLATFORM} ${GOLANG_IMAGE} as gops-cni-builder
COPY /build-gops.sh /build-gops.sh

ARG USE_PROXY_SOURCE
#use alibaba debian source
COPY /sources.list /sources.list
RUN [ "$USE_PROXY_SOURCE" != "true" ] && exit 0 ; \
      rm -f /etc/apt/sources.list ; \
      cp /sources.list /etc/apt/sources.list ;  \
      apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 648ACFD622F3D138

#with debian 11 os
RUN apt-get update && apt-get install -y binutils-aarch64-linux-gnu binutils-x86-64-linux-gnu

# build-gops.sh will build both archs at the same time
WORKDIR /
RUN chmod +x /build-gops.sh && /build-gops.sh

#========== root image ==============
FROM --platform=${TARGETPLATFORM} ${UBUNTU_IMAGE} as rootfs
ARG TARGETARCH
COPY /install-others.sh /install-others.sh
COPY /configure-iptables-wrapper.sh /configure-iptables-wrapper.sh
COPY /iptables-wrapper /usr/sbin/iptables-wrapper

#fix warning: debconf: delaying package configuration, since apt-utils is not installed
ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get upgrade -y

WORKDIR /
RUN chmod +x /configure-iptables-wrapper.sh && /configure-iptables-wrapper.sh && rm -f /configure-iptables-wrapper.sh

ARG TARGETPLATFORM
COPY --from=gops-cni-builder /out/${TARGETPLATFORM}/bin/gops /bin/gops

RUN chmod +x /install-others.sh && /install-others.sh ${TARGETARCH} && rm -f /install-others.sh


#============
FROM scratch
LABEL maintainer="maintainer@spidernet-io"
COPY --from=rootfs / /
