# EgressGateway

## Install

```shell
helm repo add egressgateway https://spidernet-io.github.io/egressgateway/
helm install egressgateway egressgateway/egressgateway --namespace kube-system
```

## Parameters

### Global parameters

| Name                           | Description                                | Value           |
| ------------------------------ | ------------------------------------------ | --------------- |
| `global.imageRegistryOverride` | The global image registry override         | `""`            |
| `global.imageTagOverride`      | The global image tag override              | `""`            |
| `global.name`                  | instance name                              | `egressgateway` |
| `global.clusterDnsDomain`      | cluster dns domain                         | `cluster.local` |
| `global.commonAnnotations`     | Annotations to add to all deployed objects | `{}`            |
| `global.commonLabels`          | Labels to add to all deployed objects      | `{}`            |
| `global.configName`            | the configmap name                         | `egressgateway` |

### Feature parameters

| Name                                         | Description                                                                                                                | Value                   |
| -------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- | ----------------------- |
| `feature.enableIPv4`                         | Enable IPv4                                                                                                                | `true`                  |
| `feature.enableIPv6`                         | Enable IPv6                                                                                                                | `false`                 |
| `feature.datapathMode`                       | iptables mode, [`iptables`, `ebpf`]                                                                                        | `iptables`              |
| `feature.tunnelIpv4Subnet`                   | Tunnel IPv4 subnet                                                                                                         | `172.31.0.0/16`         |
| `feature.tunnelIpv6Subnet`                   | Tunnel IPv6 subnet                                                                                                         | `fd11::/112`            |
| `feature.tunnelDetectMethod`                 | Tunnel base on which interface [`defaultRouteInterface`, `interface=eth0`]                                                 | `defaultRouteInterface` |
| `feature.tunnelDetectCustomInterface`        | defines custom parent interface name per node basis.                                                                       | `[]`                    |
| `feature.enableGatewayReplyRoute`            | the gateway node reply route is enabled, which should be enabled for spiderpool                                            | `false`                 |
| `feature.gatewayReplyRouteTable`             | host Reply routing table number on gateway node                                                                            | `600`                   |
| `feature.gatewayReplyRouteMark`              | host iptables mark for reply packet on gateway node                                                                        | `39`                    |
| `feature.iptables.backendMode`               | Iptables mode can be specified as `nft` or `legacy`, with `auto` meaning automatic detection. The default value is `auto`. | `auto`                  |
| `feature.vxlan.name`                         | The name of VXLAN device                                                                                                   | `egress.vxlan`          |
| `feature.vxlan.port`                         | VXLAN port                                                                                                                 | `7789`                  |
| `feature.vxlan.id`                           | VXLAN ID                                                                                                                   | `100`                   |
| `feature.vxlan.disableChecksumOffload`       | Disable checksum offload                                                                                                   | `false`                 |
| `feature.clusterCIDR.autoDetect.podCidrMode` | cni cluster used, it can be specified as `k8s`, `calico`, `auto` or `""`. The default value is `auto`.                     | `auto`                  |
| `feature.clusterCIDR.autoDetect.clusterIP`   | if ignore service ip                                                                                                       | `true`                  |
| `feature.clusterCIDR.autoDetect.nodeIP`      | if ignore node ip                                                                                                          | `true`                  |
| `feature.clusterCIDR.extraCidr`              | CIDRs provided manually                                                                                                    | `[]`                    |
| `feature.maxNumberEndpointPerSlice`          | max number of endpoints per slice                                                                                          | `100`                   |
| `feature.announcedInterfacesToExclude`       | The list of network interface excluded for announcing Egress IP.                                                           | `["^cali.*","br-*"]`    |

### feature.gatewayFailover Enable gateway failover.

| Name                                          | Description                                                                                                                                                 | Value   |
| --------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `feature.gatewayFailover.enable`              | Enable gateway failover, default `false`.                                                                                                                   | `false` |
| `feature.gatewayFailover.tunnelMonitorPeriod` | The egress controller check tunnel last update status at an interval set in seconds, default `5`.                                                           | `5`     |
| `feature.gatewayFailover.tunnelUpdatePeriod`  | The egress agent updates the tunnel status at an interval set in seconds, default `5`.                                                                      | `5`     |
| `feature.gatewayFailover.eipEvictionTimeout`  | If the last updated time of the egress tunnel exceeds this time, move the Egress IP of the node to an available node, the unit is seconds, default is `15`. | `15`    |

### Egressgateway agent parameters

| Name                                                 | Description                                                                                                     | Value                              |
| ---------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| `agent.name`                                         | The name of the egressgateway agent                                                                             | `egressgateway-agent`              |
| `agent.cmdBinName`                                   | The binary name of egressgateway agent                                                                          | `/usr/bin/agent`                   |
| `agent.hostNetwork`                                  | Enable the host network mode for the egressgateway agent Pod.                                                   | `true`                             |
| `agent.image.registry`                               | The image registry of egressgateway agent                                                                       | `ghcr.io`                          |
| `agent.image.repository`                             | The image repository of egressgateway agent                                                                     | `spidernet-io/egressgateway-agent` |
| `agent.image.pullPolicy`                             | The image pull policy of egressgateway agent                                                                    | `IfNotPresent`                     |
| `agent.image.digest`                                 | The image digest of egressgateway agent, which takes preference over tag                                        | `""`                               |
| `agent.image.tag`                                    | The image tag of egressgateway agent, overrides the image tag whose default is the chart appVersion.            | `v0.6.6`                           |
| `agent.image.imagePullSecrets`                       | the image pull secrets of egressgateway agent                                                                   | `[]`                               |
| `agent.serviceAccount.create`                        | Create the service account for the egressgateway agent                                                          | `true`                             |
| `agent.serviceAccount.annotations`                   | The annotations of egressgateway agent service account                                                          | `{}`                               |
| `agent.service.annotations`                          | The annotations for egressgateway agent service                                                                 | `{}`                               |
| `agent.service.type`                                 | The type of Service for egressgateway agent                                                                     | `ClusterIP`                        |
| `agent.priorityClassName`                            | The priority Class Name for egressgateway agent                                                                 | `system-node-critical`             |
| `agent.affinity`                                     | The affinity of egressgateway agent                                                                             | `{}`                               |
| `agent.extraArgs`                                    | The additional arguments of egressgateway agent container                                                       | `[]`                               |
| `agent.extraEnv`                                     | The additional environment variables of egressgateway agent container                                           | `[]`                               |
| `agent.extraVolumes`                                 | The additional volumes of egressgateway agent container                                                         | `[]`                               |
| `agent.extraVolumeMounts`                            | The additional hostPath mounts of egressgateway agent container                                                 | `[]`                               |
| `agent.podAnnotations`                               | The additional annotations of egressgateway agent pod                                                           | `{}`                               |
| `agent.podLabels`                                    | The additional label of egressgateway agent pod                                                                 | `{}`                               |
| `agent.resources.limits.cpu`                         | The cpu limit of egressgateway agent pod                                                                        | `500m`                             |
| `agent.resources.limits.memory`                      | The memory limit of egressgateway agent pod                                                                     | `512Mi`                            |
| `agent.resources.requests.cpu`                       | The cpu requests of egressgateway agent pod                                                                     | `100m`                             |
| `agent.resources.requests.memory`                    | The memory requests of egressgateway agent pod                                                                  | `128Mi`                            |
| `agent.securityContext`                              | The security Context of egressgateway agent pod                                                                 | `{}`                               |
| `agent.healthServer.port`                            | The http port for health checking of the egressgateway agent.                                                   | `5810`                             |
| `agent.healthServer.startupProbe.failureThreshold`   | The failure threshold of startup probe for egressgateway agent health checking                                  | `60`                               |
| `agent.healthServer.startupProbe.periodSeconds`      | The period seconds of startup probe for egressgateway agent health checking                                     | `2`                                |
| `agent.healthServer.livenessProbe.failureThreshold`  | The failure threshold of startup probe for egressgateway agent health checking                                  | `6`                                |
| `agent.healthServer.livenessProbe.periodSeconds`     | The period seconds of startup probe for egressgateway agent health checking                                     | `10`                               |
| `agent.healthServer.readinessProbe.failureThreshold` | The failure threshold of startup probe for egressgateway agent health checking                                  | `3`                                |
| `agent.healthServer.readinessProbe.periodSeconds`    | The period seconds of startup probe for egressgateway agent health checking                                     | `10`                               |
| `agent.prometheus.enabled`                           | Enable template agent to collect metrics                                                                        | `false`                            |
| `agent.prometheus.port`                              | The metrics port of template agent                                                                              | `5811`                             |
| `agent.prometheus.serviceMonitor.install`            | Install ServiceMonitor for egressgateway. This requires the prometheus CRDs to be available                     | `false`                            |
| `agent.prometheus.serviceMonitor.namespace`          | The namespace of ServiceMonitor. Default to the namespace of helm instance                                      | `""`                               |
| `agent.prometheus.serviceMonitor.annotations`        | The additional annotations of egressgateway agent ServiceMonitor                                                | `{}`                               |
| `agent.prometheus.serviceMonitor.labels`             | The additional label of egressgateway agent ServiceMonitor                                                      | `{}`                               |
| `agent.prometheus.prometheusRule.install`            | Install prometheusRule for template agent. This requires the prometheus CRDs to be available                    | `false`                            |
| `agent.prometheus.prometheusRule.namespace`          | The prometheus rule namespace. Default to the namespace of helm instance                                        | `""`                               |
| `agent.prometheus.prometheusRule.annotations`        | The additional annotations of egressgateway agent prometheusRule                                                | `{}`                               |
| `agent.prometheus.prometheusRule.labels`             | The additional label of egressgateway agent prometheusRule                                                      | `{}`                               |
| `agent.prometheus.grafanaDashboard.install`          | To install the Grafana dashboard for the egress gateway agent, the availability of Prometheus CRDs is required. | `false`                            |
| `agent.prometheus.grafanaDashboard.namespace`        | The grafana dashboard namespace. Default to the namespace of helm instance                                      | `""`                               |
| `agent.prometheus.grafanaDashboard.annotations`      | The additional annotations of egressgateway agent grafanaDashboard                                              | `{}`                               |
| `agent.prometheus.grafanaDashboard.labels`           | The additional label of egressgateway agent grafanaDashboard                                                    | `{}`                               |
| `agent.debug.logLevel`                               | The log level of egress gateway agent [`debug`, `info`, `warn`, `error`, `fatal`, `panic`]                      | `info`                             |
| `agent.debug.logEncoder`                             | Set the type of log encoder (`json`, `console`)                                                                 | `json`                             |
| `agent.debug.logWithCaller`                          | Enable or disable logging with caller information (`true`/`false`)                                              | `true`                             |
| `agent.debug.logUseDevMode`                          | Enable or disable development mode for logging (`true`/`false`)                                                 | `true`                             |
| `agent.debug.gopsPort`                               | The port used by gops tool for process monitoring and performance tuning.                                       | `5812`                             |
| `agent.debug.pyroscopeServerAddr`                    | The address of the Pyroscope server.                                                                            | `""`                               |

### Egressgateway controller parameters

| Name                                                      | Description                                                                                                                          | Value                                   |
| --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------- |
| `controller.name`                                         | The egressgateway controller name                                                                                                    | `egressgateway-controller`              |
| `controller.replicas`                                     | The replicas number of egressgateway controller                                                                                      | `1`                                     |
| `controller.cmdBinName`                                   | The binary name of egressgateway controller                                                                                          | `/usr/bin/controller`                   |
| `controller.hostNetwork`                                  | Enable host network mode of egressgateway controller pod. Notice, if no CNI available before template installation, must enable this | `false`                                 |
| `controller.image.registry`                               | The image registry of egressgateway controller                                                                                       | `ghcr.io`                               |
| `controller.image.repository`                             | The image repository of egressgateway controller                                                                                     | `spidernet-io/egressgateway-controller` |
| `controller.image.pullPolicy`                             | The image pullPolicy of egressgateway controller                                                                                     | `IfNotPresent`                          |
| `controller.image.digest`                                 | The image digest of egressgatewayController, which takes preference over tag                                                         | `""`                                    |
| `controller.image.tag`                                    | The image tag of egressgateway controller, overrides the image tag whose default is the chart appVersion.                            | `v0.6.6`                                |
| `controller.image.imagePullSecrets`                       | The image pull secrets of egressgateway controller                                                                                   | `[]`                                    |
| `controller.serviceAccount.create`                        | Create the service account for the egressgateway controller                                                                          | `true`                                  |
| `controller.serviceAccount.annotations`                   | The annotations of egressgateway controller service account                                                                          | `{}`                                    |
| `controller.service.annotations`                          | The annotations for egressgateway controller service                                                                                 | `{}`                                    |
| `controller.service.type`                                 | The type for egressgateway controller service                                                                                        | `ClusterIP`                             |
| `controller.priorityClassName`                            | The priority class name for egressgateway controller                                                                                 | `system-node-critical`                  |
| `controller.affinity`                                     | The affinity of egressgateway controller                                                                                             | `{}`                                    |
| `controller.extraArgs`                                    | The additional arguments of egressgateway controller container                                                                       | `[]`                                    |
| `controller.extraEnv`                                     | The additional environment variables of egressgateway controller container                                                           | `[]`                                    |
| `controller.extraVolumes`                                 | The additional volumes of egressgateway controller container                                                                         | `[]`                                    |
| `controller.extraVolumeMounts`                            | The additional hostPath mounts of egressgateway controller container                                                                 | `[]`                                    |
| `controller.podAnnotations`                               | The additional annotations of egressgateway controller pod                                                                           | `{}`                                    |
| `controller.podLabels`                                    | The additional label of egressgateway controller pod                                                                                 | `{}`                                    |
| `controller.securityContext`                              | The security Context of egressgateway controller pod                                                                                 | `{}`                                    |
| `controller.resources.limits.cpu`                         | The cpu limit of egressgateway controller pod                                                                                        | `500m`                                  |
| `controller.resources.limits.memory`                      | The memory limit of egressgateway controller pod                                                                                     | `512Mi`                                 |
| `controller.resources.requests.cpu`                       | The cpu requests of egressgateway controller pod                                                                                     | `100m`                                  |
| `controller.resources.requests.memory`                    | The memory requests of egressgateway controller pod                                                                                  | `128Mi`                                 |
| `controller.podDisruptionBudget.enabled`                  | Enable podDisruptionBudget for egressgateway controller pod                                                                          | `false`                                 |
| `controller.podDisruptionBudget.minAvailable`             | Minimum number/percentage of pods that should remain scheduled.                                                                      | `1`                                     |
| `controller.healthServer.port`                            | The http Port for egressgatewayController, for health checking and http service                                                      | `5820`                                  |
| `controller.healthServer.startupProbe.failureThreshold`   | The failure threshold of startup probe for egressgateway controller health checking                                                  | `30`                                    |
| `controller.healthServer.startupProbe.periodSeconds`      | The period seconds of startup probe for egressgatewayController health checking                                                      | `2`                                     |
| `controller.healthServer.livenessProbe.failureThreshold`  | The failure threshold of startup probe for egressgateway controller health checking                                                  | `6`                                     |
| `controller.healthServer.livenessProbe.periodSeconds`     | The period seconds of startup probe for egressgatewayController health checking                                                      | `10`                                    |
| `controller.healthServer.readinessProbe.failureThreshold` | The failure threshold of startup probe for egressgateway controller health checking                                                  | `3`                                     |
| `controller.healthServer.readinessProbe.periodSeconds`    | The period seconds of startup probe for egressgateway controller health checking                                                     | `10`                                    |
| `controller.webhookPort`                                  | The http port for egressgatewayController webhook                                                                                    | `5822`                                  |
| `controller.prometheus.enabled`                           | Enable egress gateway controller to collect metrics                                                                                  | `false`                                 |
| `controller.prometheus.port`                              | The metrics port of egress gateway controller                                                                                        | `5821`                                  |
| `controller.prometheus.serviceMonitor.install`            | Install ServiceMonitor for egress gateway agent. This requires the prometheus CRDs to be available                                   | `false`                                 |
| `controller.prometheus.serviceMonitor.namespace`          | The serviceMonitor namespace. Default to the namespace of helm instance                                                              | `""`                                    |
| `controller.prometheus.serviceMonitor.annotations`        | The additional annotations of egressgatewayController serviceMonitor                                                                 | `{}`                                    |
| `controller.prometheus.serviceMonitor.labels`             | The additional label of egressgatewayController serviceMonitor                                                                       | `{}`                                    |
| `controller.prometheus.prometheusRule.install`            | Install prometheusRule for egress gateway agent. This requires the prometheus CRDs to be available                                   | `false`                                 |
| `controller.prometheus.prometheusRule.namespace`          | The prometheusRule namespace. Default to the namespace of helm instance                                                              | `""`                                    |
| `controller.prometheus.prometheusRule.annotations`        | The additional annotations of egressgatewayController prometheus rule                                                                | `{}`                                    |
| `controller.prometheus.prometheusRule.labels`             | The additional label of egressgateway controller prometheus rule                                                                     | `{}`                                    |
| `controller.prometheus.grafanaDashboard.install`          | Install grafana dashboard for egress gateway agent. This requires the prometheus CRDs to be available                                | `false`                                 |
| `controller.prometheus.grafanaDashboard.namespace`        | The grafanaDashboard namespace. Default to the namespace of helm instance                                                            | `""`                                    |
| `controller.prometheus.grafanaDashboard.annotations`      | The additional annotations of egressgatewayController grafanaDashboard                                                               | `{}`                                    |
| `controller.prometheus.grafanaDashboard.labels`           | The additional label of egressgatewayController grafanaDashboard                                                                     | `{}`                                    |
| `controller.debug.logLevel`                               | The log level of egress gateway controller [`debug`, `info`, `warn`, `error`, `fatal`, `panic`]                                      | `info`                                  |
| `controller.debug.logEncoder`                             | Set the type of log encoder (`json`, `console`)                                                                                      | `json`                                  |
| `controller.debug.logWithCaller`                          | Enable or disable logging with caller information (`true`/`false`)                                                                   | `true`                                  |
| `controller.debug.logUseDevMode`                          | Enable or disable development mode for logging (`true`/`false`)                                                                      | `true`                                  |
| `controller.debug.gopsPort`                               | The port used by gops tool for process monitoring and performance tuning.                                                            | `5824`                                  |
| `controller.debug.pyroscopeServerAddr`                    | The address of the Pyroscope server.                                                                                                 | `""`                                    |
| `controller.tls.method`                                   | the method for generating TLS certificates. [`provided`, `certmanager`, `auto`]                                                      | `auto`                                  |
| `controller.tls.secretName`                               | The secret name for storing TLS certificates                                                                                         | `egressgateway-controller-server-certs` |
| `controller.tls.certmanager.certValidityDuration`         | Generated certificates validity duration in days for 'certmanager' method                                                            | `365`                                   |
| `controller.tls.certmanager.issuerName`                   | Issuer name of cert manager 'certmanager'. If not specified, a CA issuer will be created.                                            | `""`                                    |
| `controller.tls.certmanager.extraDnsNames`                | Extra DNS names added to certificate when it's auto generated                                                                        | `[]`                                    |
| `controller.tls.certmanager.extraIPAddresses`             | Extra IP addresses added to certificate when it's auto generated                                                                     | `[]`                                    |
| `controller.tls.provided.tlsCert`                         | Encoded tls certificate for provided method                                                                                          | `""`                                    |
| `controller.tls.provided.tlsKey`                          | Encoded tls key for provided method                                                                                                  | `""`                                    |
| `controller.tls.provided.tlsCa`                           | Encoded tls CA for provided method                                                                                                   | `""`                                    |
| `controller.tls.auto.caExpiration`                        | CA expiration for auto method                                                                                                        | `73000`                                 |
| `controller.tls.auto.certExpiration`                      | Server cert expiration for auto method                                                                                               | `73000`                                 |
| `controller.tls.auto.extraIpAddresses`                    | Extra IP addresses of server certificate for auto method                                                                             | `[]`                                    |
| `controller.tls.auto.extraDnsNames`                       | Extra DNS names of server cert for auto method                                                                                       | `[]`                                    |
| `cleanup.enable`                                          | clean up resources when helm uninstall                                                                                               | `true`                                  |
