#！/bin/bash
## SPDX-License-Identifier: Apache-2.0
## Copyright Authors of Spider

OS=$(uname | tr 'A-Z' 'a-z')

DOWNLOAD_OPT=""
if [ -n "$http_proxy" ]; then
  DOWNLOAD_OPT=" -x $http_proxy "
fi

if ! kubectl help &>/dev/null  ; then
    echo "error, miss 'kubectl', try to install it "
    LATEST_VERSION=$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)
    echo "downland kubectl ${LATEST_VERSION}"
    curl ${DOWNLOAD_OPT} -Lo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${LATEST_VERSION}/bin/$OS/amd64/kubectl
    chmod +x /usr/local/bin/kubectl
    ! kubectl -h  &>/dev/null && echo "error, failed to install kubectl" && exit 1
else
    echo "pass   'kubectl' installed:  $(kubectl version --client=true | grep -E -o "Client.*GitVersion:\"[^[:space:]]+\"" | awk -F, '{print $NF}') "
fi


# Install Kind Bin
if ! kind &> /dev/null ; then
    echo "error, miss 'kind', try to install it "
    LATEST_VERSION=` curl -s https://api.github.com/repos/kubernetes-sigs/kind/releases/latest |  grep -Po '"tag_name": "\K.*?(?=")' `
    if [ -z "$LATEST_VERSION" ] ; then
        echo "error, failed to get latest version for kind"
        exit 1
    fi
    echo "downland kind $LATEST_VERSION"
    curl ${DOWNLOAD_OPT} -Lo /usr/local/bin/kind https://github.com/kubernetes-sigs/kind/releases/download/${LATEST_VERSION}/kind-$OS-amd64
    chmod +x /usr/local/bin/kind
    ! kind -h  &>/dev/null && echo "error, failed to install kind" && exit 1
else
    echo "pass   'kind' installed:  $(kind --version) "
fi


# Install Helm
if ! helm > /dev/null 2>&1 ; then
    echo "error, miss 'helm', try to install it "
    LATEST_VERSION=` curl -s https://api.github.com/repos/helm/helm/releases/latest |  grep -Po '"tag_name": "\K.*?(?=")' `
    if [ -z "$LATEST_VERSION" ] ; then
        echo "error, failed to get latest version for helm"
        exit 1
    fi
    curl ${DOWNLOAD_OPT} -Lo /tmp/helm.tar.gz "https://get.helm.sh/helm-${LATEST_VERSION}-$OS-amd64.tar.gz"
    tar -xzvf /tmp/helm.tar.gz && mv $OS-amd64/helm  /usr/local/bin
    chmod +x /usr/local/bin/helm
    rm /tmp/helm.tar.gz
    rm $OS-amd64/LICENSE
    rm $OS-amd64/README.md
    ! helm version &>/dev/null && echo "error, failed to install helm" && exit 1
else
    echo "pass   'helm' installed:  $( helm version | grep -E -o "Version:\"v[^[:space:]]+\"" ) "
fi

# docker
if ! docker &> /dev/null ; then
    echo "error, miss 'docker'"
    exit 1
else
    echo "pass   'docker' installed:  $(docker -v) "
fi

# ====modify==== add more e2e binray


exit 0
