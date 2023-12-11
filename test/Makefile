include ../Makefile.defs


.PHONY: init_kind_env
init_kind_env:
	echo $(cni)
	make init_one_kind -e KIND_CONFIG_PATH=./kindconfig/global-kind.yaml -e KIND_CLUSTER_NAME=$(E2E_KIND_CLUSTER_NAME) -e KIND_KUBECONFIG=$(E2E_KIND_KUBECONFIG_PATH)
	make install_proscope
	make run_nettool_server
	if [ "$(cni)" == "flannel" ] ; then \
  		make install_flannel ; \
  	elif [ "$(cni)" == "weave" ] ; then \
  	    make install_weave ; \
	elif [ "$(cni)" == "spiderpool" ] ; then \
  	    make install_spiderpool ; \
    else \
    	make install_calico ; \
    fi
	make install_kwok


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
	KIND_OPTION="" ; \
       		[ -n "$(E2E_KIND_NODE_IMAGE)" ] && KIND_OPTION=" --image $(E2E_KIND_NODE_IMAGE) " && echo "setup kind with E2E_KIND_NODE_IMAGE=$(E2E_KIND_NODE_IMAGE)"; \
       		kind create cluster --name  $(KIND_CLUSTER_NAME) --config $(E2E_RUNTIME_DIR)/kind_config_$(KIND_CLUSTER_NAME).yaml --kubeconfig $(KIND_KUBECONFIG) $${KIND_OPTION}
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
		echo "finish setup pyroscope" ; \
    fi


# install calico
.PHONY: install_calico
install_calico: KIND_CLUSTER_NAME ?= $(E2E_KIND_CLUSTER_NAME)
install_calico:
	echo "install calico"
	export CALICO_VERSION=$(CALICO_VERSION); \
	export DEFAULT_CALICO_VERSION=$(DEFAULT_CALICO_VERSION); \
	export HTTP_PROXY=$(HTTP_PROXY); \
	export CALICO_REGISTRY=$(CALICO_REGISTRY); \
	export INSTALL_TIME_OUT=$(INSTALL_TIME_OUT); \
	export E2E_IP_FAMILY=$(E2E_IP_FAMILY); \
	export E2E_KIND_KUBECONFIG_PATH=$(E2E_KIND_KUBECONFIG_PATH); \
	export KIND_CLUSTER_NAME=$(E2E_KIND_CLUSTER_NAME) ; \
	bash ./scripts/installCalico.sh

.PHONY: install_flannel
install_flannel:
	echo "install flannel"
	export FLANNEL_VERSION=$(FLANNEL_VERSION); \
	export HTTP_PROXY=$(HTTP_PROXY); \
	export FLANNEL_REGISTRY=$(FLANNEL_REGISTRY); \
	export INSTALL_TIME_OUT=$(INSTALL_TIME_OUT); \
	export E2E_IP_FAMILY=$(E2E_IP_FAMILY); \
	export KUBECONFIG=$(E2E_KIND_KUBECONFIG_PATH); \
	export KIND_CLUSTER_NAME=$(E2E_KIND_CLUSTER_NAME) ; \
	export E2E_KIND_IPV4_POD_CIDR=$(E2E_KIND_IPV4_POD_CIDR); \
	export E2E_KIND_IPV6_POD_CIDR=$(E2E_KIND_IPV6_POD_CIDR); \
	bash ./scripts/installFlannel.sh

.PHONY: install_weave
install_weave:
	echo "install weave"
	export WEAVE_VERSION=$(WEAVE_VERSION); \
	export HTTP_PROXY=$(HTTP_PROXY); \
	export FLANNEL_REGISTRY=$(FLANNEL_REGISTRY); \
	export INSTALL_TIME_OUT=$(INSTALL_TIME_OUT); \
	export E2E_IP_FAMILY=$(E2E_IP_FAMILY); \
	export KUBECONFIG=$(E2E_KIND_KUBECONFIG_PATH); \
	export KIND_CLUSTER_NAME=$(E2E_KIND_CLUSTER_NAME) ; \
	export E2E_KIND_IPV4_POD_CIDR=$(E2E_KIND_IPV4_POD_CIDR); \
	export E2E_KIND_IPV6_POD_CIDR=$(E2E_KIND_IPV6_POD_CIDR); \
	bash ./scripts/installWeave.sh

.PHONY: install_spiderpool
install_spiderpool:
	echo "install spiderpool"
	export SPIDERPOOL_VERSION=$(SPIDERPOOL_VERSION); \
	export HTTP_PROXY=$(HTTP_PROXY); \
	export SPIDERPOOL_REGISTRY=$(SPIDERPOOL_REGISTRY); \
	export INSTALL_TIME_OUT=$(INSTALL_TIME_OUT); \
	export E2E_IP_FAMILY=$(E2E_IP_FAMILY); \
	export KUBECONFIG=$(E2E_KIND_KUBECONFIG_PATH); \
	export KIND_CLUSTER_NAME=$(E2E_KIND_CLUSTER_NAME) ; \
	export E2E_KIND_IPV4_POD_CIDR=$(E2E_KIND_IPV4_POD_CIDR); \
	export E2E_KIND_IPV6_POD_CIDR=$(E2E_KIND_IPV6_POD_CIDR); \
	bash ./scripts/installSpiderpool.sh

# install kwok
.PHONY: install_kwok
install_kwok:
	if [ "$(E2E_INSTALL_KWOK)" == "true" ] ; then \
		echo "install kwok" ; \
		export INSTALL_TIME_OUT=$(INSTALL_TIME_OUT); \
		export KWOK_VERSION=$(KWOK_VERSION); \
		export E2E_KIND_KUBECONFIG_PATH=$(E2E_KIND_KUBECONFIG_PATH); \
		export https_proxy=$(HTTP_PROXY) ; \
		bash ./scripts/installKwok.sh ; \
	else \
	  	echo "does not install kwok" ; \
	fi


# run_nettool_server
.PHONY: run_nettool_server
run_nettool_server: load_nettool_image
	echo "delete nettool server container"
	-@  docker stop  $(NETTOOLS_SERVER_A) &>/dev/null
	-@  docker rm  $(NETTOOLS_SERVER_A) &>/dev/null
	-@  docker stop  $(NETTOOLS_SERVER_B) &>/dev/null
	-@  docker rm  $(NETTOOLS_SERVER_B) &>/dev/null
	echo "run nettool server container"
	{ docker run -itd --name $(NETTOOLS_SERVER_A)  --network kind $(NETTOOLS_IMAGE) $(NETTOOLS_SERVER_BIN) -protocol $(MOD) -tcpPort $(TCP_PORT) --udpPort $(UDP_PORT) -webPort $(WEB_PORT) & } || { echo "failed to run nettools server a"; exit 1; }; \
	{ docker run -itd --name $(NETTOOLS_SERVER_B)  --network kind $(NETTOOLS_IMAGE) $(NETTOOLS_SERVER_BIN) -protocol $(MOD) -tcpPort $(TCP_PORT) --udpPort $(UDP_PORT) -webPort $(WEB_PORT) & } || { echo "failed to run nettools server b"; exit 1; }; \
	make check_netttool_server

# kind load nettool image
.PHONY: load_nettool_image
load_nettool_image: IMAGE_NAME := $(REGISTER)/$(GIT_REPO)-nettools
load_nettool_image: IMAGE_TAG := $(GIT_COMMIT_VERSION)
load_nettool_image: KIND_CLUSTER_NAME ?= $(E2E_KIND_CLUSTER_NAME)
load_nettool_image:
	make -C $(ROOT_DIR)	build_local_nettools_image
	@echo "loading nettool image..."
	IMAGE="$$(docker image ls $(IMAGE_NAME):$(IMAGE_TAG) -q)" ; \
	echo "$${IMAGE}" ; \
	if [ -z "$${IMAGE}" ]; then \
	    echo "failed to find nettool image" ; \
            exit 1 ; \
	else \
	    LOAD_IMG="$(IMAGE_NAME):$(NET_TOOLS_IMAGE_TAG)" ; \
        	docker image tag "$${IMAGE}" "$${LOAD_IMG}" ; \
        	kind load docker-image "$${LOAD_IMG}" --name "$(KIND_CLUSTER_NAME)" || \
        	{ echo "failed to load nettool image"; exit 1; } ; \
	fi

# check nettools server is ok
.PHONY: check_netttool_server
check_netttool_server:
	echo "check nettools server is ok"
	make -C $(ROOT_DIR) build_nettools_cilent_bin || { echo "failed to make nettool client bin"; exit 1; }
	export TCP_PORT=$(TCP_PORT); \
	export UDP_PORT=$(UDP_PORT); \
	export WEB_PORT=$(WEB_PORT); \
	export NETTOOLS_SERVER_A=$(NETTOOLS_SERVER_A); \
	export NETTOOLS_SERVER_B=$(NETTOOLS_SERVER_B); \
	export CLIENT=$(NETTOOLS_CLIENT_BIN); \
	export E2E_IP_FAMILY=$(E2E_IP_FAMILY); \
	bash ./scripts/check-nettools-server.sh

#==================

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


# install spidernet on global cluster
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
	- helm --kubeconfig=$(KIND_KUBECONFIG) uninstall -n $(E2E_NAMESPACE) project || true
	HELM_OPTION="" ; \
    	if [ -n "$(PYROSCOPE_LOCAL_PORT)" ] ; then \
			echo "add env" ; \
			ServerAddress=$$(docker network inspect kind -f {{\(index\ $$.IPAM.Config\ 0\).Gateway}}) ; \
			HELM_OPTION+=" --set agent.debug.pyroscopeServerAddr=http://$${ServerAddress}:$(PYROSCOPE_LOCAL_PORT) " ; \
			HELM_OPTION+=" --set controller.debug.pyroscopeServerAddr=http://$${ServerAddress}:$(PYROSCOPE_LOCAL_PORT) " ; \
		fi ; \
		HELM_OPTION+=" --set agent.prometheus.enabled=true --set controller.prometheus.enabled=true  " ; \
		HELM_OPTION+=" --set agent.debug.logLevel=debug --set controller.debug.logLevel=debug " ; \
		HELM_OPTION+=" --set controller.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key=node-role.kubernetes.io/control-plane  " ; \
		HELM_OPTION+=" --set controller.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator=Exists  " ; \
		if [ "$(E2E_IP_FAMILY)" == "dual" ] ; then HELM_OPTION+=" --set feature.enableIPv4=true --set feature.enableIPv6=true" ; fi ; \
		if [ "$(E2E_IP_FAMILY)" == "ipv4" ] ; then HELM_OPTION+=" --set feature.enableIPv4=true --set feature.enableIPv6=false" ; fi ; \
		if [ "$(E2E_IP_FAMILY)" == "ipv6" ] ; then HELM_OPTION+=" --set feature.enableIPv4=false --set feature.enableIPv6=true" ; fi ; \
		helm --kubeconfig=$(KIND_KUBECONFIG) install project $(ROOT_DIR)/charts \
				-n $(E2E_NAMESPACE) --create-namespace --wait --debug \
				--set global.imageTagOverride=$(PROJECT_IMAGE_VERSION) \
				$${HELM_OPTION} \
				|| { KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) ./scripts/debugCluster.sh $(KIND_KUBECONFIG) "detail"  $(E2E_NAMESPACE) ; exit 1 ; } ; \
		exit 0


# test kind is ok
.PHONY: install_example_app
install_example_app: KIND_KUBECONFIG ?= $(E2E_KIND_KUBECONFIG_PATH)
install_example_app: KIND_CLUSTER_NAME ?= $(E2E_KIND_CLUSTER_NAME)
install_example_app:
	@echo "---------- debug local file: /etc/bash.bashrc"
	@cat /etc/bash.bashrc
	@echo "---------- install example app"
	echo "NETTOOLS_IMAGE=$(NETTOOLS_IMAGE)"
	export NETTOOLS_IMAGE=$(NETTOOLS_IMAGE) && yq -i '.spec.template.spec.containers[0].image = strenv(NETTOOLS_IMAGE)' yaml/testpod.yaml
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
	-@ docker images | grep egressgateway | awk '{print $$3}' | xargs -I {} docker rmi {}


#============ e2e ====================
.PHONY: e2e_test
e2e_test: KIND_CLUSTER_NAME ?= $(E2E_KIND_CLUSTER_NAME)
e2e_test: KIND_KUBECONFIG ?= $(E2E_KIND_KUBECONFIG_PATH)
e2e_test:
	@echo -e "\033[35mRun e2e testing on the cluster $(KIND_CLUSTER_NAME) \033[0m"
	@echo -e "\033[35m label=$(E2E_GINKGO_LABELS)\n timeout=$(E2E_TIMEOUT) \n options=$(E2E_GINKGO_OPTION) \n egress_namespace=$(E2E_NAMESPACE) \033[0m"
	@echo -e "\033[35m[STEP1]\033[0m check KUBECONFIG file path"; \
		export KUBECONFIG=$(KIND_KUBECONFIG) ; [ -f "$(KIND_KUBECONFIG)" ] || { echo -e "error, does not exist KUBECONFIG $(E2E_KUBECONFIG)" ; exit 1 ; } ; \
		echo -e "success" ; \
		echo -e "\033[35m[STEP2]\033[0m clean old e2e log file"; \
		[ -f "$(E2E_LOG_FILE)" ] && rm -f "$(E2E_LOG_FILE)"; \
		echo -e "success" ; \
		echo -e "\033[35m[STEP3]\033[0m run debugCluster.sh"; \
		echo "=========== before test `date` ===========" >> $(E2E_LOG_FILE) ; \
		RESULT=0 ; \
		echo -e "\033[35m[STEP4]\033[0m run ginkgo.sh"; \
		export IMAGE=$(NETTOOLS_IMAGE); \
		$(ROOT_DIR)/tools/golang/ginkgo.sh \
			--race --timeout=$(E2E_TIMEOUT) --output-interceptor-mode=none --poll-progress-after=20s \
			--json-report e2ereport.json --output-dir $(ROOT_DIR) --procs $(E2E_GINKGO_PROCS) \
			--label-filter="$(E2E_GINKGO_LABELS)" -randomize-suites -randomize-all -vv --fail-fast  $(E2E_GINKGO_OPTION) \
			-r e2e/* || RESULT=1 ; \
		echo "=========== after test `date` ===========" >> $(E2E_LOG_FILE) ; \
		./scripts/debugCluster.sh $(KIND_KUBECONFIG) "system" "$(E2E_LOG_FILE)" $(E2E_NAMESPACE) ; \
		KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) ./scripts/debugCluster.sh $(KIND_KUBECONFIG) "detail" "$(E2E_LOG_FILE)" $(E2E_NAMESPACE) ; \
		./scripts/debugCluster.sh $(KIND_KUBECONFIG) "error" "$(E2E_LOG_FILE)" $(E2E_NAMESPACE) || { echo "error, found error log, datarace/pacni/longlock !!!" ; RESULT=1 ; } ; \
		if (($${RESULT} != 0)) ; then \
		   echo "failed to run e2e test"  ; \
		   exit 1 ; \
		fi ; \
		echo "" ; \
		echo "============================================" ; \
		echo "succeeded to run all test" ; \
		echo "output report to e2ereport.json" ; \
		echo "output env log to $(E2E_LOG_FILE) "
