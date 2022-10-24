#!/bin/bash

# SPDX-License-Identifier: Apache-2.0
# Copyright Authors of Spider

CURRENT_FILENAME=$( basename $0 )
CURRENT_DIR_PATH=$(cd $(dirname $0); pwd)
PROJECT_ROOT_PATH=$( cd ${CURRENT_DIR_PATH}/../.. && pwd )

E2E_KUBECONFIG="$1"
# gops or detail
TYPE="$2"
E2E_LOG_FILE_NAME="$3"

[ -z "$E2E_KUBECONFIG" ] && echo "error, miss E2E_KUBECONFIG " && exit 1
[ ! -f "$E2E_KUBECONFIG" ] && echo "error, could not find file $E2E_KUBECONFIG " && exit 1
echo "$CURRENT_FILENAME : E2E_KUBECONFIG $E2E_KUBECONFIG "

CONTROLLER_POD_LIST=$( kubectl get pods --no-headers --kubeconfig ${E2E_KUBECONFIG}  --namespace kube-system --selector app.kubernetes.io/component=egressgateway-controller --output jsonpath={.items[*].metadata.name} )
AGENT_POD_LIST=$( kubectl get pods --no-headers --kubeconfig ${E2E_KUBECONFIG}  --namespace kube-system --selector app.kubernetes.io/component=egressgateway-agent --output jsonpath={.items[*].metadata.name} )
[ -z "$CONTROLLER_POD_LIST" ] && echo "error, failed to find any spider controller pod" && exit 1
[ -z "$AGENT_POD_LIST" ] && echo "error, failed to find any spider agent pod" && exit 1


if [ -n "$E2E_LOG_FILE_NAME" ] ; then
    echo "output debug information to $E2E_LOG_FILE_NAME"
    exec 6>&1
    exec >>${E2E_LOG_FILE_NAME} 2>&1
fi


RESUTL_CODE=0
if [ "$TYPE"x == "gops"x ] ; then
    echo ""
    echo "=============== gops data of controller ============== "
    for POD in $CONTROLLER_POD_LIST ; do
      echo ""
      echo "---------${POD}--------"
      kubectl exec ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG}  gops stats 1
      kubectl exec ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG}  gops memstats 1
    done

    echo ""
    echo "=============== gops data of agent ============== "
    for POD in $AGENT_POD_LIST ; do
      echo ""
      echo "---------${POD}--------"
      kubectl exec ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG}  gops stats 1
      kubectl exec ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG}  gops memstats 1
    done

elif [ "$TYPE"x == "detail"x ] ; then

    # ====modify==== add more log here

    echo "=============== nodes status ============== "
    echo "-------- kubectl get node -o wide"
    kubectl get node -o wide --kubeconfig ${E2E_KUBECONFIG} --show-labels

    echo "=============== pods status ============== "
    echo "-------- kubectl get pod -A -o wide"
    kubectl get pod -A -o wide --kubeconfig ${E2E_KUBECONFIG} --show-labels

    echo ""
    echo "=============== event ============== "
    echo "------- kubectl get events -n kube-system"
    kubectl get events -n kube-system --kubeconfig ${E2E_KUBECONFIG}

    echo ""
    echo "=============== egressgateway-controller describe ============== "
    for POD in $CONTROLLER_POD_LIST ; do
      echo ""
      echo "--------- kubectl describe pod ${POD} -n kube-system"
      kubectl describe pod ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG}
    done

    echo ""
    echo "=============== egressgateway-agent describe ============== "
    for POD in $AGENT_POD_LIST ; do
      echo ""
      echo "---------kubectl describe pod ${POD} -n kube-system "
      kubectl describe pod ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG}
    done

    echo ""
    echo "=============== egressgateway-controller logs ============== "
    for POD in $CONTROLLER_POD_LIST ; do
      echo ""
      echo "---------kubectl logs ${POD} -n kube-system"
      kubectl logs ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG}
      echo "--------- kubectl logs ${POD} -n kube-system --previous"
      kubectl logs ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG} --previous
    done

    echo ""
    echo "=============== egressgateway-agent logs ============== "
    for POD in $AGENT_POD_LIST ; do
      echo ""
      echo "--------- kubectl logs ${POD} -n kube-system "
      kubectl logs ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG}
      echo "--------- kubectl logs ${POD} -n kube-system --previous"
      kubectl logs ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG} --previous
    done

    echo ""
    echo "===============  get crd  ============== "


    echo ""
    echo "=============== node log  ============== "
    KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-"egressgateway"}
    KIND_NODES=$(  kind get  nodes --name ${KIND_CLUSTER_NAME} )
    [ -z "$KIND_NODES" ] && echo "warning, failed to find nodes of kind cluster $KIND_CLUSTER_NAME " || true
    for NODE in $KIND_NODES ; do
        echo "--------- logs from node ${NODE}"
        docker exec $NODE ls /var/log/
    done


elif [ "$TYPE"x == "datarace"x ] ; then
    LOG_MARK="WARNING: DATA RACE"

    CHECK_DATA_RACE(){
      echo ""
      echo "---------${POD}--------"
      if kubectl logs ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG} | grep "${LOG_MARK}" &>/dev/null ; then
          echo "error, data race in ${POD} !!!!!!!"
          kubectl logs ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG}
          RESUTL_CODE=1
      else
          echo "no data race "
      fi
    }

    echo ""
    echo "=============== egressgateway-controller data race ============== "
    for POD in $CONTROLLER_POD_LIST ; do
      CHECK_DATA_RACE
    done

    echo ""
    echo "=============== egressgateway-agent data race ============== "
    for POD in $AGENT_POD_LIST ; do
      CHECK_DATA_RACE
    done

elif [ "$TYPE"x == "longlock"x ] ; then
    LOG_MARK="Goroutine took lock"

    CHECK_LONG_LOCK(){
      echo ""
      echo "---------${POD}--------"
      if kubectl logs ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG} | grep "${LOG_MARK}" &>/dev/null ; then
          echo "error, long lock in ${POD} !!!!!!!"
          kubectl logs ${POD} -n kube-system --kubeconfig ${E2E_KUBECONFIG}
          RESUTL_CODE=1
      else
          echo "no long lock "
      fi
    }
    echo ""
    echo "=============== egressgateway-controller long lock ============== "
    for POD in $CONTROLLER_POD_LIST ; do
      CHECK_LONG_LOCK
    done

    echo ""
    echo "=============== egressgateway-agent long lock ============== "
    for POD in $AGENT_POD_LIST ; do
      CHECK_LONG_LOCK
    done


else
    echo "error, unknown type $TYPE "
    RESUTL_CODE=1
fi

exit $RESUTL_CODE
