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

# ====modify====
COMPONENT_NAMESPACE="kube-system"
COMPONENT_GOROUTINE_MAX=300
CONTROLLER_LABEL="app.kubernetes.io/component=rocktemplate-controller"
AGENT_LABEL="app.kubernetes.io/component=rocktemplate-agent"


CONTROLLER_POD_LIST=$( kubectl get pods --no-headers --kubeconfig ${E2E_KUBECONFIG}  --namespace ${COMPONENT_NAMESPACE} --selector ${CONTROLLER_LABEL} --output jsonpath={.items[*].metadata.name} )
AGENT_POD_LIST=$( kubectl get pods --no-headers --kubeconfig ${E2E_KUBECONFIG}  --namespace ${COMPONENT_NAMESPACE} --selector ${AGENT_LABEL} --output jsonpath={.items[*].metadata.name} )
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
      kubectl exec ${POD} -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG} -- gops stats 1
      kubectl exec ${POD} -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG} -- gops memstats 1
    done

    echo ""
    echo "=============== gops data of agent ============== "
    for POD in $AGENT_POD_LIST ; do
      echo ""
      echo "---------${POD}--------"
      kubectl exec ${POD} -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG} -- gops stats 1
      kubectl exec ${POD} -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG} -- gops memstats 1
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
    echo "------- kubectl get events -n ${COMPONENT_NAMESPACE}"
    kubectl get events -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG}

    echo ""
    echo "=============== rocktemplate-controller describe ============== "
    for POD in $CONTROLLER_POD_LIST ; do
      echo ""
      echo "--------- kubectl describe pod ${POD} -n ${COMPONENT_NAMESPACE}"
      kubectl describe pod ${POD} -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG}
    done

    echo ""
    echo "=============== rocktemplate-agent describe ============== "
    for POD in $AGENT_POD_LIST ; do
      echo ""
      echo "---------kubectl describe pod ${POD} -n ${COMPONENT_NAMESPACE} "
      kubectl describe pod ${POD} -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG}
    done

    echo ""
    echo "=============== rocktemplate-controller logs ============== "
    for POD in $CONTROLLER_POD_LIST ; do
      echo ""
      echo "---------kubectl logs ${POD} -n ${COMPONENT_NAMESPACE}"
      kubectl logs ${POD} -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG}
      echo "--------- kubectl logs ${POD} -n ${COMPONENT_NAMESPACE} --previous"
      kubectl logs ${POD} -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG} --previous
    done

    echo ""
    echo "=============== rocktemplate-agent logs ============== "
    for POD in $AGENT_POD_LIST ; do
      echo ""
      echo "--------- kubectl logs ${POD} -n ${COMPONENT_NAMESPACE} "
      kubectl logs ${POD} -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG}
      echo "--------- kubectl logs ${POD} -n ${COMPONENT_NAMESPACE} --previous"
      kubectl logs ${POD} -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG} --previous
    done

    echo ""
    echo "===============  get crd  ============== "


    echo ""
    echo "=============== node log  ============== "
    KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-"rocktemplate"}
    KIND_NODES=$(  kind get  nodes --name ${KIND_CLUSTER_NAME} )
    [ -z "$KIND_NODES" ] && echo "warning, failed to find nodes of kind cluster $KIND_CLUSTER_NAME " || true
    for NODE in $KIND_NODES ; do
        echo "--------- logs from node ${NODE}"
        docker exec $NODE ls /var/log/
    done


elif [ "$TYPE"x == "error"x ] ; then
    CHECK_ERROR(){
        LOG_MARK="$1"
        POD="$2"
        NAMESPACE="$3"

        echo ""
        echo "---------${POD}--------"
        if kubectl logs ${POD} -n ${NAMESPACE} --kubeconfig ${E2E_KUBECONFIG} | grep -E -i "${LOG_MARK}" &>/dev/null ; then
            echo "error, in ${POD}, found error, ${LOG_MARK} !!!!!!!"
            kubectl logs ${POD} -n ${NAMESPACE} --kubeconfig ${E2E_KUBECONFIG}
            RESUTL_CODE=1
        else
            echo "no error "
        fi
    }

    DATA_RACE_LOG_MARK="WARNING: DATA RACE"
    LOCK_LOG_MARK="Goroutine took lock"
    PANIC_LOG_MARK="panic .* runtime error"

    echo ""
    echo "=============== check kinds of error  ============== "
    for POD in $CONTROLLER_POD_LIST $AGENT_POD_LIST ; do
        echo ""
        echo "----- check data race in ${COMPONENT_NAMESPACE}/${POD} "
        CHECK_ERROR "${DATA_RACE_LOG_MARK}" "${POD}" "${COMPONENT_NAMESPACE}"

        echo ""
        echo "----- check long lock in ${COMPONENT_NAMESPACE}/${POD} "
        CHECK_ERROR "${LOCK_LOG_MARK}" "${POD}" "${COMPONENT_NAMESPACE}"

        echo ""
        echo "----- check panic in ${COMPONENT_NAMESPACE}/${POD} "
        CHECK_ERROR "${PANIC_LOG_MARK}" "${POD}" "${COMPONENT_NAMESPACE}"

        echo ""
        echo "----- check gorouting leak in ${COMPONENT_NAMESPACE}/${POD} "
        GOROUTINE_NUM=`kubectl exec ${POD} -n ${COMPONENT_NAMESPACE} --kubeconfig ${E2E_KUBECONFIG} -- gops stats 1 | grep "goroutines:" | grep -E -o "[0-9]+" `
        if [ -z "$GOROUTINE_NUM" ] ; then
            echo "warning, failed to find GOROUTINE_NUM in ${COMPONENT_NAMESPACE}/${POD} "
        elif (( GOROUTINE_NUM >= COMPONENT_GOROUTINE_MAX )) ; then
             echo "maybe goroutine leak, found ${GOROUTINE_NUM} goroutines in ${COMPONENT_NAMESPACE}/${POD} , which is bigger than default ${COMPONENT_GOROUTINE_MAX}"
             RESUTL_CODE=1
        fi

        echo ""
        echo "----- check pod restart in ${COMPONENT_NAMESPACE}/${POD}"
        RESTARTS=` kubectl get pod ${POD} -n ${COMPONENT_NAMESPACE} -o wide --kubeconfig ${E2E_KUBECONFIG} | sed '1 d'  | awk '{print $4}' `
        if [ -z "$RESTARTS" ] ; then
            echo "warning, failed to find RESTARTS in ${COMPONENT_NAMESPACE}/${POD} "
        elif (( RESTARTS != 0 )) ; then
             echo "found pod restart event"
             RESUTL_CODE=1
        fi

    done

else
    echo "error, unknown type $TYPE "
    RESUTL_CODE=1
fi

exit $RESUTL_CODE
