include ../Makefile.defs


.PHONY: init_kind_env
init_kind_env:
	make init_one_kind -e KIND_CONFIG_PATH=./kindconfig/global-kind.yaml -e KIND_CLUSTER_NAME=$(E2E_KIND_CLUSTER_NAME) -e KIND_KUBECONFIG=$(E2E_KIND_KUBECONFIG_PATH)
	make install_proscope
	make install_calico
	make run_nettool_server
	make load_nettool_image


.PHONY: init_one_kind
init_one_kind: KIND_CONFIG_PATH ?=
init_one_kind: KIND_CLUSTER_NAME ?=
init_one_kind: KIND_KUBECONFIG ?=
init_one_kind: checkBin clean
	@echo "================== init kind cluster $(KIND_CLUSTER_NAME) KIND_CONFIG_PATH=$(KIND_CONFIG_PATH) KIND_KUBECONFIG=$(KIND_KUBECONFIG) E2E_IP_FAMILY=$(E2E_IP_FAMILY)"
	[ -n $(KIND_CLUSTER_NAME) ] || { echo "error, miss KIND_CLUSTER_NAME " ; exit 1 ; }
	[ -f $(KIND_CONFIG_PATH) ] || { echo "error, miss file KIND_CONFIG_PATH=$(KIND_CONFIG_PATH)" ; exit 1 ; }
	- mkdir -p $(E2E_RUNTIME_DIR) || true
	NEW_KIND_YAML=$(E2E_RUNTIME_DIR)/kind_config_$(KIND_CLUSTER_NAME).yaml ;\
		INSERT_LINE=` grep "insert subnet inform" $(KIND_CONFIG_PATH) -n | awk -F':' '{print $$1}' ` ; \
		echo "insert after line $${INSERT_LINE}" ;\
		sed  ''"$${INSERT_LINE}"' a \  ipFamily: $(E2E_IP_FAMILY)' $(KIND_CONFIG_PATH) > $${NEW_KIND_YAML} ; \
		if [ "$(E2E_IP_FAMILY)" == "ipv4" ] ; then \
			sed -i  ''"$${INSERT_LINE}"' a \  podSubnet: "$(E2E_KIND_IPV4_POD_CIDR)"' $${NEW_KIND_YAML} ;\
			sed -i  ''"$${INSERT_LINE}"' a \  serviceSubnet: "$(E2E_KIND_IPV4_SERVICE_CIDR)"' $${NEW_KIND_YAML} ;\
		elif [ "$(E2E_IP_FAMILY)" == "ipv6" ] ; then \
			sed -i  ''"$${INSERT_LINE}"' a \  podSubnet: "$(E2E_KIND_IPV6_POD_CIDR)"' $${NEW_KIND_YAML} ; \
			sed -i  ''"$${INSERT_LINE}"' a \  serviceSubnet: "$(E2E_KIND_IPV6_SERVICE_CIDR)"' $${NEW_KIND_YAML} ; \
		else \
			sed -i  ''"$${INSERT_LINE}"' a \  podSubnet: "$(E2E_KIND_IPV4_POD_CIDR),$(E2E_KIND_IPV6_POD_CIDR)"' $${NEW_KIND_YAML}  ; \
			sed -i  ''"$${INSERT_LINE}"' a \  serviceSubnet: "$(E2E_KIND_IPV4_SERVICE_CIDR),$(E2E_KIND_IPV6_SERVICE_CIDR)"' $${NEW_KIND_YAML}  ; \
  		fi
	- sysctl -w net.ipv6.conf.all.disable_ipv6=0 || true
	- sysctl -w fs.inotify.max_user_watches=524288 || true
	- sysctl -w fs.inotify.max_user_instances=8192  || true
	- kind delete cluster --name  $(KIND_CLUSTER_NAME)
	kind create cluster --name  $(KIND_CLUSTER_NAME) --config $(E2E_RUNTIME_DIR)/kind_config_$(KIND_CLUSTER_NAME).yaml --kubeconfig $(KIND_KUBECONFIG)
	- kubectl --kubeconfig $(KIND_KUBECONFIG) taint nodes --all node-role.kubernetes.io/master- || true
	- kubectl --kubeconfig $(KIND_KUBECONFIG) taint nodes --all node-role.kubernetes.io/control-plane- || true
	@echo "===================== deploy prometheus CRD ========== "
	# https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml ; } \
	kubectl apply --kubeconfig $(KIND_KUBECONFIG)  -f ./yaml/monitoring.coreos.com_servicemonitors.yaml
	# https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/example/prometheus-operator-crd/monitoring.coreos.com_podmonitors.yaml ; } \
	kubectl apply --kubeconfig $(KIND_KUBECONFIG) -f ./yaml/monitoring.coreos.com_podmonitors.yaml
	# https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/example/prometheus-operator-crd/monitoring.coreos.com_prometheusrules.yaml ; } \
	kubectl apply --kubeconfig $(KIND_KUBECONFIG) -f ./yaml/monitoring.coreos.com_prometheusrules.yaml
	# https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/example/prometheus-operator-crd/monitoring.coreos.com_probes.yaml  ; } \
	kubectl apply --kubeconfig $(KIND_KUBECONFIG) -f ./yaml/monitoring.coreos.com_probes.yaml
	# https://raw.githubusercontent.com/grafana-operator/grafana-operator/master/deploy/manifests/latest/crds.yaml  ; } \
	kubectl apply --kubeconfig $(KIND_KUBECONFIG) -f ./yaml/grafanadashboards.yaml
	echo "show kubernetes node image " && docker ps
	@echo "========================================================"
	@echo "   deploy kind cluster $(KIND_CLUSTER_NAME)             "
	@echo "   export KUBECONFIG=$(KIND_KUBECONFIG)                 "
	@echo "   kubectl get pod -o wide -A                           "
	@echo "========================================================"


.PHONY: checkBin
checkBin:
	$(ROOT_DIR)/test/scripts/installE2eTools.sh

.PHONY: install_proscope
install_proscope:
	if [ -n "$(PYROSCOPE_LOCAL_PORT)" ] ; then \
  		echo "install proscope " ; \
		docker stop $(PYROSCOPE_CONTAINER_NAME) &>/dev/null || true ; \
		docker rm $(PYROSCOPE_CONTAINER_NAME) &>/dev/null || true ; \
		ServerAddress=$$(docker network inspect kind -f {{\(index\ $$.IPAM.Config\ 0\).Gateway}}) ; \
		echo "setup pyroscope on $${ServerAddress}:$(PYROSCOPE_LOCAL_PORT)" ; \
		docker run -d --name $(PYROSCOPE_CONTAINER_NAME) -p $(PYROSCOPE_LOCAL_PORT):4040 $(PYROSCOPE_IMAGE_NAME) server ; \
		echo "finish setuping pyroscope " ; \
    fi


.PHONY: install_calico
install_calico:
	echo "install calico"
	export CALICO_VERSION=$(CALICO_VERSION); \
	export INSTALL_TIME_OUT=$(INSTALL_TIME_OUT); \
	export E2E_IP_FAMILY=$(E2E_IP_FAMILY); \
	export E2E_KIND_KUBECONFIG_PATH=$(E2E_KIND_KUBECONFIG_PATH); \
	bash ./scripts/installCalico.sh



#==================

.PHONY: kind_install_ipset
kind_install_ipset: E2E_CLUSTER_NAME ?= $(E2E_KIND_CLUSTER_NAME)
kind_install_ipset:
	@ NODE_LIST=` docker ps | egrep " kindest/node.* $(E2E_CLUSTER_NAME)-(control|worker)" | awk '{print $$NF }' | tr '\n' ' ' ` ; \
		[ -n "$$NODE_LIST" ] || { echo "error, failed to find any kind nodes, please setup kind cluster $(E2E_CLUSTER_NAME) first" ; exit 1 ; } ; \
		echo "find cluster node: $${NODE_LIST}" ; \
		for NODE in  $${NODE_LIST} ; do \
		  echo "update apt sources list for node $$NODE " ; \
		  docker exec $${NODE} sed -i -e 's?archive?cn.archive?g;s?/security?/cn.archive?g;s?impish?jammy?g' /etc/apt/sources.list ; \
		  docker exec $${NODE} apt update; \
		  docker exec $${NODE} apt install -y ipset; \
        done ; \

# this will auto tag github ci image : agent:xxx -> github.com/spidernet-io/egressgateway/agent:xxx
.PHONY: check_images_ready
check_images_ready:
	echo "check image  " ; \
	IMAGE_LIST=` helm template test $(ROOT_DIR)/charts --set global.imageTagOverride=$(PROJECT_IMAGE_VERSION)  | grep " image: " | tr -d '"'| awk '{print $$2}' ` ; \
	if [ -z "$${IMAGE_LIST}" ] ; then \
		echo "warning, failed to find image from chart " ; \
		exit 1 ;\
	else \
		echo "find image from chart : $${IMAGE_LIST} " ; \
		for IMAGE in $${IMAGE_LIST} ; do \
		  	echo "try to find image $${IMAGE} " ; \
			EXIST=` docker images | awk '{printf("%s:%s\n",$$1,$$2)}' | grep "$${IMAGE}" ` || true ; \
			if [ -z "$${EXIST}" ] ; then \
					CI_IMAGE=$${IMAGE##*/} ; \
			  		echo "try to find github CI image $${CI_IMAGE} " ; \
			  		EXIST=` docker images | awk '{printf("%s:%s\n",$$1,$$2)}' | grep "$${CI_IMAGE}" ` || true ; \
			  		if [ -z "$${EXIST}" ] ; then \
			  			echo "error, failed to find image $${IMAGE}" ; \
			  			echo "error, failed to find image $${CI_IMAGE}" ; \
			  			exit 1 ; \
			  		fi ; \
			  		docker tag $${CI_IMAGE} $${IMAGE} ; \
			fi ;\
			echo "image exists: $${IMAGE}" ; \
		done ; \
		docker images ; \
	fi


# install egressgateway on global cluster
.PHONY: deploy_project
deploy_project: KIND_KUBECONFIG ?= $(E2E_KIND_KUBECONFIG_PATH)
deploy_project: KIND_CLUSTER_NAME ?= $(E2E_KIND_CLUSTER_NAME)
deploy_project:
	echo "try to load local image tag $(PROJECT_IMAGE_VERSION) " ; \
	IMAGE_LIST=` helm template test $(ROOT_DIR)/charts --set global.imageTagOverride=$(PROJECT_IMAGE_VERSION)  | grep " image: " | tr -d '"'| awk '{print $$2}' ` ; \
	if [ -z "$${IMAGE_LIST}" ] ; then \
		echo "warning, failed to find image from chart " ; \
	else \
		echo "found image from chart : $${IMAGE_LIST} " ; \
		for IMAGE in $${IMAGE_LIST} ; do \
			EXIST=` docker images | awk '{printf("%s:%s\n",$$1,$$2)}' | grep "$${IMAGE}" ` ; \
			if [ -z "$${EXIST}" ] ; then \
			  echo "docker pull $${IMAGE} to local" ; \
			  docker pull $${IMAGE} ; \
			fi ;\
			echo "load local image $${IMAGE} " ; \
			kind load docker-image $${IMAGE}  --name $(KIND_CLUSTER_NAME)  ; \
		done ; \
	fi
	- helm --kubeconfig=$(KIND_KUBECONFIG) uninstall -n kube-system project || true
	HELM_OPTION="" ; \
    	if [ -n "$(PYROSCOPE_LOCAL_PORT)" ] ; then \
			echo "add env" ; \
			ServerAddress=$$(docker network inspect kind -f {{\(index\ $$.IPAM.Config\ 0\).Gateway}}) ; \
			HELM_OPTION+=" --set egressgatewayAgent.extraEnv[0].name=ENV_PYROSCOPE_PUSH_SERVER_ADDRESS  --set egressgatewayAgent.extraEnv[0].value=http://$${ServerAddress}:$(PYROSCOPE_LOCAL_PORT) " ; \
			HELM_OPTION+=" --set egressgatewayController.extraEnv[0].name=ENV_PYROSCOPE_PUSH_SERVER_ADDRESS  --set egressgatewayController.extraEnv[0].value=http://$${ServerAddress}:$(PYROSCOPE_LOCAL_PORT) " ; \
		fi ; \
		HELM_OPTION+=" --set egressgatewayAgent.prometheus.enabled=true --set egressgatewayController.prometheus.enabled=true  " ; \
		HELM_OPTION+=" --set egressgatewayAgent.debug.logLevel=debug --set egressgatewayController.debug.logLevel=debug " ; \
		if [ "$(E2E_IP_FAMILY)" == "dual" ] ; then HELM_OPTION+=" --set feature.enableIPv4=true --set feature.enableIPv6=true" ; fi ; \
		if [ "$(E2E_IP_FAMILY)" == "ipv4" ] ; then HELM_OPTION+=" --set feature.enableIPv4=true --set feature.enableIPv6=false" ; fi ; \
		if [ "$(E2E_IP_FAMILY)" == "ipv6" ] ; then HELM_OPTION+=" --set feature.enableIPv4=false --set feature.enableIPv6=true" ; fi ; \
		if [ "$(PROJECT_FORWARD_METHOD)" == "active-passive" ] ; then HELM_OPTION+=" --set feature.forwardMethod=active-passive" ; fi ; \
		if [ "$(PROJECT_FORWARD_METHOD)" == "active-active" ] ; then HELM_OPTION+=" --set feature.forwardMethod=active-active" ; fi ; \
		helm --kubeconfig=$(KIND_KUBECONFIG) install project $(ROOT_DIR)/charts \
				-n kube-system --wait --debug \
				--set global.imageTagOverride=$(PROJECT_IMAGE_VERSION) \
				$${HELM_OPTION} \
				|| { KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) ./scripts/debugCluster.sh $(KIND_KUBECONFIG) "detail"   ; exit 1 ; } ; \
		exit 0

# run_nettool_server
.PHONY: run_nettool_server
run_nettool_server:
	echo "run nettool server bin"
	make -C $(ROOT_DIR) build_nettools_all_bin || { echo "failed build_nettools_all_bin"; exit 1; }
#	if [ "$(E2E_IP_FAMILY)" == "ipv6" ] ; then export SERVER_IP=$(SERVER_IP_V6) ; fi ; \
#	if [ "$(E2E_IP_FAMILY)" == "dual" ] ; then export SERVER_IP=$(SERVER_IP_V6) ; fi ; \
#	if [ "$(E2E_IP_FAMILY)" == "ipv4" ] ; then export SERVER_IP=$(SERVER_IP_V4) ; fi ; \

	export SERVER_IP=$(SERVER_IP_V4) ; \
    export MOD=$(MOD) ; \
    export TCP_PORT=$(TCP_PORT) ; \
    export UDP_PORT=$(UDP_PORT) ; \
    export WEB_PORT=$(WEB_PORT) ; \
    { $(SERVER_BIN) & } || { echo "failed to run nettools server"; exit 1; }

# install nettool client
.PHONY: install_nettool_client
install_nettool_client: IMAGE := $(REGISTER)/$(GIT_REPO)-nettools:$(NET_TOOLS_IMAGE_TAG)
install_nettool_client: KIND_KUBECONFIG ?= $(E2E_KIND_KUBECONFIG_PATH)
install_nettool_client: load_nettool_image
	echo "install nettool client pod"; \
	sed -e \
	's?<<NETTOOLS_IMAGE>>?$(IMAGE)?g;s?<<SERVER_IP>>?$(SERVER_IP)?g;s?<<MOD>>?$(MOD)?g;s?<<TCP_PORT>>?$(TCP_PORT)?g;s?<<UDP_PORT>>?$(UDP_PORT)?g;s?<<WEB_PORT>>?$(WEB_PORT)?g;s?<<NETTOOLS_CLIENT>>?$(NETTOOLS_CLIENT)?g;s?<<NODE_NAME>>?$(NODE_NAME)?g' \
	./yaml/testclient.yaml | kubectl --kubeconfig $(KIND_KUBECONFIG) apply -f - ; \
	kubectl --kubeconfig $(KIND_KUBECONFIG) wait --for condition=ready -l app=$(NETTOOLS_CLIENT) --timeout 10s po || { echo "timeout to wait pod running"; exit 1; }

# kind load nettool image
.PHONY: load_nettool_image
load_nettool_image: IMAGE_NAME := $(REGISTER)/$(GIT_REPO)-nettools
load_nettool_image: IMAGE_TAG := $(GIT_COMMIT_VERSION)
load_nettool_image: KIND_CLUSTER_NAME ?= $(E2E_KIND_CLUSTER_NAME)
load_nettool_image:
	make -C $(ROOT_DIR)	build_local_nettools_image ; \
	IMAGE=`docker image ls | grep $(IMAGE_NAME) | grep $(IMAGE_TAG)` ; \
	[ -z $${IMAGE} ] && { echo "failed to find nettool image"; exit 1; } ; \
	IMG=`echo $${IMAGE} | awk '{{printf("%s:%s\n",$$1,$$2)}}'`; \
	LOAD_IMG=$(IMAGE_NAME):$(NET_TOOLS_IMAGE_TAG) ; \
	docker image tag $${IMG} $${LOAD_IMG}; \
	kind load docker-image $${LOAD_IMG}  --name $(KIND_CLUSTER_NAME) || { echo "failed to load nettool image"; exit 1; }

# test kind is ok
.PHONY: install_example_app
install_example_app: KIND_KUBECONFIG ?= $(E2E_KIND_KUBECONFIG_PATH)
install_example_app: KIND_CLUSTER_NAME ?= $(E2E_KIND_CLUSTER_NAME)
install_example_app:
	@echo "---------- install example app"
	kubectl --kubeconfig=$(KIND_KUBECONFIG) apply -f yaml/testpod.yaml
	@ if ! kubectl rollout status  deployment/test --kubeconfig $(KIND_KUBECONFIG) -w --timeout=120s ; then \
			echo "error, failed to create a test pod" ; \
			exit 1 ; \
		fi ; \
		echo "succeeded to deploy test deployment "
	@echo "========================================================"
	@echo "   deploy kind cluster $(KIND_CLUSTER_NAME)             "
	@echo "   export KUBECONFIG=$(KIND_KUBECONFIG)                 "
	@echo "   kubectl get pod -o wide -A                           "
	@echo "========================================================"
	@ KUBECONFIG=$(KIND_KUBECONFIG)  kubectl get pod -o wide -A


.PHONY: clean
clean:
	-@ kind delete cluster --name $(E2E_KIND_CLUSTER_NAME)
	-@ rm -rf $(E2E_RUNTIME_DIR)
	-@ docker stop $(PYROSCOPE_CONTAINER_NAME) &>/dev/null
	-@ docker rm $(PYROSCOPE_CONTAINER_NAME) &>/dev/null
	-@ ps aux | grep $(SERVER_BIN) | grep -v color | awk '{print $$2}' | xargs kill



#============ e2e ====================
.PHONY: e2e_test
e2e_test: KIND_CLUSTER_NAME ?= $(E2E_KIND_CLUSTER_NAME)
e2e_test: KIND_KUBECONFIG ?= $(E2E_KIND_KUBECONFIG_PATH)
e2e_test:
	@echo -e "\033[35m Run e2e test on the cluster $(KIND_CLUSTER_NAME) \033[0m "
	@ echo -e "\033[35m [E2E] Run E2E with ginkgo label=$(E2E_GINKGO_LABELS) , timeout=$(E2E_TIMEOUT) GINKGO_OPTION=$(E2E_GINKGO_OPTION) \033[0m"
	@  NODE_LIST=` docker ps | egrep " kindest/node.* $(KIND_CLUSTER_NAME)-(control|worker)" | awk '{print $$NF }' ` ; \
		[ -n "$$NODE_LIST" ] || { echo "error, failed to find any kind nodes, please setup kind cluster $(KIND_CLUSTER_NAME) first" ; exit 1 ; } ; \
		NODE_LIST=` echo "$${NODE_LIST}" | tr -d ' ' | tr '\n' ',' ` ; \
		NODE_LIST=$${NODE_LIST%%,} ; \
		echo "find cluster node: $${NODE_LIST}" ; \
		export E2E_KIND_CLUSTER_NODE_LIST="$${NODE_LIST}" ; \
		export E2E_CLUSTER_NAME=$(KIND_CLUSTER_NAME) ; \
		if [ "$(E2E_IP_FAMILY)" == "ipv4" ] ; then \
			export E2E_IPV4_ENABLED=true ; export E2E_IPV6_ENABLED=false ; \
		elif [ "$(E2E_IP_FAMILY)" == "ipv6" ] ; then \
			export E2E_IPV4_ENABLED=false ; export E2E_IPV6_ENABLED=true ; \
		else \
			export E2E_IPV4_ENABLED=true ; export E2E_IPV6_ENABLED=true ; \
		fi ; \
		export E2E_KUBECONFIG_PATH=$(KIND_KUBECONFIG) ; [ -f "$(KIND_KUBECONFIG)" ] || { echo "error, does not exist KUBECONFIG $(E2E_KUBECONFIG)" ; exit 1 ; } ; \
		rm -f $(E2E_LOG_FILE) || true ; \
		echo "=========== before test `date` ===========" >> $(E2E_LOG_FILE) ; \
		./scripts/debugCluster.sh $(KIND_KUBECONFIG) "system" "$(E2E_LOG_FILE)" ; \
		RESULT=0 ; \
		$(ROOT_DIR)/tools/golang/ginkgo.sh \
			--race --timeout=$(E2E_TIMEOUT) --output-interceptor-mode=none  --slow-spec-threshold=15s \
			--json-report e2ereport.json --output-dir $(ROOT_DIR) --procs $(E2E_GINKGO_PROCS) \
			--label-filter="$(E2E_GINKGO_LABELS)" -randomize-suites -randomize-all  -vv --fail-fast  $(E2E_GINKGO_OPTION) \
			-r e2e/*  || RESULT=1  ; \
		echo "=========== after test `date` ===========" >> $(E2E_LOG_FILE) ; \
		./scripts/debugCluster.sh $(KIND_KUBECONFIG) "system" "$(E2E_LOG_FILE)" ; \
		KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) ./scripts/debugCluster.sh $(KIND_KUBECONFIG) "detail" "$(E2E_LOG_FILE)" ; \
		./scripts/debugCluster.sh $(KIND_KUBECONFIG) "error" "$(E2E_LOG_FILE)" || { echo "error, found error log, datarace/pacni/longlock !!!" ; RESULT=1 ; } ; \
		if (($${RESULT} != 0)) ; then \
		   echo "failed to run e2e test"  ; \
		   exit 1 ; \
		fi ; \
		echo "" ; \
		echo "============================================" ; \
		echo "succeeded to run all test" ; \
		echo "output report to e2ereport.json" ; \
		echo "output env log to $(E2E_LOG_FILE) "
