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

| Name                                   | Description                                                                                         | Value                   |
| -------------------------------------- | --------------------------------------------------------------------------------------------------- | ----------------------- |
| `feature.enableIPv4`                   | Enable IPv4                                                                                         | `true`                  |
| `feature.enableIPv6`                   | Enable IPv6                                                                                         | `false`                 |
| `feature.startRouteTable`              | Start route table                                                                                   | `50`                    |
| `feature.iptables.backendMode`         | The mode for iptables (`legacy` or `nft`). If left blank, it's automatically detected. Default: "". | `""`                    |
| `feature.datapathMode`                 | iptables mode, [`iptables`, `ebpf`]                                                                 | `iptables`              |
| `feature.tunnelIpv4Subnet`             | Tunnel IPv4 subnet                                                                                  | `172.31.0.0/16`         |
| `feature.tunnelIpv6Subnet`             | Tunnel IPv6 subnet                                                                                  | `fd11::/112`            |
| `feature.tunnelDetectMethod`           | Tunnel base on which interface [`defaultRouteInterface`, `interface=eth0`]                          | `defaultRouteInterface` |
| `feature.forwardMethod`                | Tunnel base on which interface [`active-active`: require kernel >=4.4, `active-passive`]            | `active-passive`        |
| `feature.vxlan.name`                   | The name of VXLAN device                                                                            | `egress.vxlan`          |
| `feature.vxlan.port`                   | VXLAN port                                                                                          | `7789`                  |
| `feature.vxlan.id`                     | VXLAN ID                                                                                            | `100`                   |
| `feature.vxlan.disableChecksumOffload` | Disable checksum offload                                                                            | `true`                  |

### Egressgateway agent parameters

| Name                                                              | Description                                                                                               | Value                              |
| ----------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| `egressgatewayAgent.name`                                         | The name of the egressgateway agent                                                                       | `egressgateway-agent`              |
| `egressgatewayAgent.cmdBinName`                                   | The binary name of egressgateway agent                                                                    | `/usr/bin/agent`                   |
| `egressgatewayAgent.hostNetwork`                                  | Enable the host network mode for the egressgateway agent Pod.                                             | `true`                             |
| `egressgatewayAgent.image.registry`                               | The image registry of egressgateway agent                                                                 | `ghcr.io`                          |
| `egressgatewayAgent.image.repository`                             | The image repository of egressgateway agent                                                               | `spidernet-io/egressgateway-agent` |
| `egressgatewayAgent.image.pullPolicy`                             | The image pull policy of egressgateway agent                                                              | `IfNotPresent`                     |
| `egressgatewayAgent.image.digest`                                 | The image digest of egressgateway agent, which takes preference over tag                                  | `""`                               |
| `egressgatewayAgent.image.tag`                                    | The image tag of egressgateway agent, overrides the image tag whose default is the chart appVersion.      | `v0.1.0`                           |
| `egressgatewayAgent.image.imagePullSecrets`                       | the image image pull secrets of egressgateway agent                                                       | `[]`                               |
| `egressgatewayAgent.serviceAccount.create`                        | Create the service account for the egressgateway agent                                                    | `true`                             |
| `egressgatewayAgent.serviceAccount.annotations`                   | The annotations of egressgateway agent service account                                                    | `{}`                               |
| `egressgatewayAgent.service.annotations`                          | The annotations for egressgateway agent service                                                           | `{}`                               |
| `egressgatewayAgent.service.type`                                 | The type of Service for egressgateway agent                                                               | `ClusterIP`                        |
| `egressgatewayAgent.priorityClassName`                            | The priority Class Name for egressgateway agent                                                           | `system-node-critical`             |
| `egressgatewayAgent.affinity`                                     | The affinity of egressgatewayAgent                                                                        | `{}`                               |
| `egressgatewayAgent.extraArgs`                                    | The additional arguments of egressgatewayAgent container                                                  | `[]`                               |
| `egressgatewayAgent.extraEnv`                                     | The additional environment variables of egressgatewayAgent container                                      | `[]`                               |
| `egressgatewayAgent.extraVolumes`                                 | The additional volumes of egressgatewayAgent container                                                    | `[]`                               |
| `egressgatewayAgent.extraVolumeMounts`                            | The additional hostPath mounts of egressgatewayAgent container                                            | `[]`                               |
| `egressgatewayAgent.podAnnotations`                               | The additional annotations of egressgatewayAgent pod                                                      | `{}`                               |
| `egressgatewayAgent.podLabels`                                    | The additional label of egressgatewayAgent pod                                                            | `{}`                               |
| `egressgatewayAgent.resources.limits.cpu`                         | The cpu limit of egressgatewayAgent pod                                                                   | `500m`                             |
| `egressgatewayAgent.resources.limits.memory`                      | The memory limit of egressgatewayAgent pod                                                                | `512Mi`                            |
| `egressgatewayAgent.resources.requests.cpu`                       | The cpu requests of egressgatewayAgent pod                                                                | `100m`                             |
| `egressgatewayAgent.resources.requests.memory`                    | The memory requests of egressgatewayAgent pod                                                             | `128Mi`                            |
| `egressgatewayAgent.securityContext`                              | The security Context of egressgatewayAgent pod                                                            | `{}`                               |
| `egressgatewayAgent.healthServer.port`                            | The http port for health checking of the egressgateway agent.                                             | `5810`                             |
| `egressgatewayAgent.healthServer.startupProbe.failureThreshold`   | The failure threshold of startup probe for egressgatewayAgent health checking                             | `60`                               |
| `egressgatewayAgent.healthServer.startupProbe.periodSeconds`      | The period seconds of startup probe for egressgatewayAgent health checking                                | `2`                                |
| `egressgatewayAgent.healthServer.livenessProbe.failureThreshold`  | The failure threshold of startup probe for egressgatewayAgent health checking                             | `6`                                |
| `egressgatewayAgent.healthServer.livenessProbe.periodSeconds`     | The period seconds of startup probe for egressgatewayAgent health checking                                | `10`                               |
| `egressgatewayAgent.healthServer.readinessProbe.failureThreshold` | The failure threshold of startup probe for egressgatewayAgent health checking                             | `3`                                |
| `egressgatewayAgent.healthServer.readinessProbe.periodSeconds`    | The period seconds of startup probe for egressgatewayAgent health checking                                | `10`                               |
| `egressgatewayAgent.prometheus.enabled`                           | Enable template agent to collect metrics                                                                  | `false`                            |
| `egressgatewayAgent.prometheus.port`                              | The metrics port of template agent                                                                        | `5811`                             |
| `egressgatewayAgent.prometheus.serviceMonitor.install`            | Install ServiceMonitor for egressgateway. This requires the prometheus CRDs to be available               | `false`                            |
| `egressgatewayAgent.prometheus.serviceMonitor.namespace`          | The namespace of ServiceMonitor. Default to the namespace of helm instance                                | `""`                               |
| `egressgatewayAgent.prometheus.serviceMonitor.annotations`        | The additional annotations of egressgateway agent ServiceMonitor                                          | `{}`                               |
| `egressgatewayAgent.prometheus.serviceMonitor.labels`             | The additional label of egressgateway agent ServiceMonitor                                                | `{}`                               |
| `egressgatewayAgent.prometheus.prometheusRule.install`            | Install prometheusRule for template agent. This requires the prometheus CRDs to be available              | `false`                            |
| `egressgatewayAgent.prometheus.prometheusRule.namespace`          | The prometheus rule namespace. Default to the namespace of helm instance                                  | `""`                               |
| `egressgatewayAgent.prometheus.prometheusRule.annotations`        | The additional annotations of egressgatewayAgent prometheusRule                                           | `{}`                               |
| `egressgatewayAgent.prometheus.prometheusRule.labels`             | The additional label of egressgatewayAgent prometheusRule                                                 | `{}`                               |
| `egressgatewayAgent.prometheus.grafanaDashboard.install`          | To install the Grafana dashboard for the template agent, the availability of Prometheus CRDs is required. | `false`                            |
| `egressgatewayAgent.prometheus.grafanaDashboard.namespace`        | The grafana dashboard namespace. Default to the namespace of helm instance                                | `""`                               |
| `egressgatewayAgent.prometheus.grafanaDashboard.annotations`      | The additional annotations of egressgatewayAgent grafanaDashboard                                         | `{}`                               |
| `egressgatewayAgent.prometheus.grafanaDashboard.labels`           | The additional label of egressgatewayAgent grafanaDashboard                                               | `{}`                               |
| `egressgatewayAgent.debug.logLevel`                               | The log level of template agent [debug, info, warn, error, fatal, panic]                                  | `info`                             |
| `egressgatewayAgent.debug.gopsPort`                               | The gops port of template agent                                                                           | `5812`                             |

### Egressgateway controller parameters

| Name                                                                   | Description                                                                                                                          | Value                                   |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------- |
| `egressgatewayController.name`                                         | The egressgateway controller name                                                                                                    | `egressgateway-controller`              |
| `egressgatewayController.replicas`                                     | The replicas number of egressgateway controller                                                                                      | `1`                                     |
| `egressgatewayController.cmdBinName`                                   | The binary name name of egressgateway controller                                                                                     | `/usr/bin/controller`                   |
| `egressgatewayController.hostNetwork`                                  | Enable host network mode of egressgateway controller pod. Notice, if no CNI available before template installation, must enable this | `false`                                 |
| `egressgatewayController.image.registry`                               | The image registry of egressgateway controller                                                                                       | `ghcr.io`                               |
| `egressgatewayController.image.repository`                             | The image repository of egressgateway controller                                                                                     | `spidernet-io/egressgateway-controller` |
| `egressgatewayController.image.pullPolicy`                             | The image pullPolicy of egressgateway controller                                                                                     | `IfNotPresent`                          |
| `egressgatewayController.image.digest`                                 | The image digest of egressgatewayController, which takes preference over tag                                                         | `""`                                    |
| `egressgatewayController.image.tag`                                    | The image tag of egressgatewayController, overrides the image tag whose default is the chart appVersion.                             | `v0.1.0`                                |
| `egressgatewayController.image.imagePullSecrets`                       | The image image pull secrets of egressgateway controller                                                                             | `[]`                                    |
| `egressgatewayController.serviceAccount.create`                        | Create the service account for the egressgatewayController                                                                           | `true`                                  |
| `egressgatewayController.serviceAccount.annotations`                   | The annotations of egressgatewayController service account                                                                           | `{}`                                    |
| `egressgatewayController.service.annotations`                          | The annotations for egressgatewayController service                                                                                  | `{}`                                    |
| `egressgatewayController.service.type`                                 | The type for egressgatewayController service                                                                                         | `ClusterIP`                             |
| `egressgatewayController.priorityClassName`                            | The priority class name for egressgateway controller                                                                                 | `system-node-critical`                  |
| `egressgatewayController.affinity`                                     | The affinity of egressgateway controller                                                                                             | `{}`                                    |
| `egressgatewayController.extraArgs`                                    | The additional arguments of egressgateway controller container                                                                       | `[]`                                    |
| `egressgatewayController.extraEnv`                                     | The additional environment variables of egressgateway controller container                                                           | `[]`                                    |
| `egressgatewayController.extraVolumes`                                 | The additional volumes of egressgateway controller container                                                                         | `[]`                                    |
| `egressgatewayController.extraVolumeMounts`                            | The additional hostPath mounts of egressgatewayController container                                                                  | `[]`                                    |
| `egressgatewayController.podAnnotations`                               | The additional annotations of egressgateway controller pod                                                                           | `{}`                                    |
| `egressgatewayController.podLabels`                                    | The additional label of egressgateway controller pod                                                                                 | `{}`                                    |
| `egressgatewayController.securityContext`                              | The security Context of egressgateway controller pod                                                                                 | `{}`                                    |
| `egressgatewayController.resources.limits.cpu`                         | The cpu limit of egressgatewayController pod                                                                                         | `500m`                                  |
| `egressgatewayController.resources.limits.memory`                      | The memory limit of egressgatewayController pod                                                                                      | `512Mi`                                 |
| `egressgatewayController.resources.requests.cpu`                       | The cpu requests of egressgatewayController pod                                                                                      | `100m`                                  |
| `egressgatewayController.resources.requests.memory`                    | The memory requests of egressgatewayController pod                                                                                   | `128Mi`                                 |
| `egressgatewayController.podDisruptionBudget.enabled`                  | Enable podDisruptionBudget for egressgatewayController pod                                                                           | `false`                                 |
| `egressgatewayController.podDisruptionBudget.minAvailable`             | Minimum number/percentage of pods that should remain scheduled.                                                                      | `1`                                     |
| `egressgatewayController.healthServer.port`                            | The http Port for egressgatewayController, for health checking and http service                                                      | `5820`                                  |
| `egressgatewayController.healthServer.startupProbe.failureThreshold`   | The failure threshold of startup probe for egressgatewayController health checking                                                   | `30`                                    |
| `egressgatewayController.healthServer.startupProbe.periodSeconds`      | The period seconds of startup probe for egressgatewayController health checking                                                      | `2`                                     |
| `egressgatewayController.healthServer.livenessProbe.failureThreshold`  | The failure threshold of startup probe for egressgatewayController health checking                                                   | `6`                                     |
| `egressgatewayController.healthServer.livenessProbe.periodSeconds`     | The period seconds of startup probe for egressgatewayController health checking                                                      | `10`                                    |
| `egressgatewayController.healthServer.readinessProbe.failureThreshold` | The failure threshold of startup probe for egressgatewayController health checking                                                   | `3`                                     |
| `egressgatewayController.healthServer.readinessProbe.periodSeconds`    | The period seconds of startup probe for egressgatewayController health checking                                                      | `10`                                    |
| `egressgatewayController.webhookPort`                                  | The http port for egressgatewayController webhook                                                                                    | `5822`                                  |
| `egressgatewayController.prometheus.enabled`                           | Enable template Controller to collect metrics                                                                                        | `false`                                 |
| `egressgatewayController.prometheus.port`                              | The metrics port of template Controller                                                                                              | `5821`                                  |
| `egressgatewayController.prometheus.serviceMonitor.install`            | Install ServiceMonitor for template agent. This requires the prometheus CRDs to be available                                         | `false`                                 |
| `egressgatewayController.prometheus.serviceMonitor.namespace`          | The serviceMonitor namespace. Default to the namespace of helm instance                                                              | `""`                                    |
| `egressgatewayController.prometheus.serviceMonitor.annotations`        | The additional annotations of egressgatewayController serviceMonitor                                                                 | `{}`                                    |
| `egressgatewayController.prometheus.serviceMonitor.labels`             | The additional label of egressgatewayController serviceMonitor                                                                       | `{}`                                    |
| `egressgatewayController.prometheus.prometheusRule.install`            | Install prometheusRule for template agent. This requires the prometheus CRDs to be available                                         | `false`                                 |
| `egressgatewayController.prometheus.prometheusRule.namespace`          | The prometheusRule namespace. Default to the namespace of helm instance                                                              | `""`                                    |
| `egressgatewayController.prometheus.prometheusRule.annotations`        | The additional annotations of egressgatewayController prometheus rule                                                                | `{}`                                    |
| `egressgatewayController.prometheus.prometheusRule.labels`             | The additional label of egressgateway controller prometheus rule                                                                     | `{}`                                    |
| `egressgatewayController.prometheus.grafanaDashboard.install`          | Install grafana dashboard for template agent. This requires the prometheus CRDs to be available                                      | `false`                                 |
| `egressgatewayController.prometheus.grafanaDashboard.namespace`        | The grafanaDashboard namespace. Default to the namespace of helm instance                                                            | `""`                                    |
| `egressgatewayController.prometheus.grafanaDashboard.annotations`      | The additional annotations of egressgatewayController grafanaDashboard                                                               | `{}`                                    |
| `egressgatewayController.prometheus.grafanaDashboard.labels`           | The additional label of egressgatewayController grafanaDashboard                                                                     | `{}`                                    |
| `egressgatewayController.debug.logLevel`                               | The log level of template Controller [debug, info, warn, error, fatal, panic]                                                        | `info`                                  |
| `egressgatewayController.debug.gopsPort`                               | The gops port of template Controller                                                                                                 | `5824`                                  |
| `egressgatewayController.tls.method`                                   | the method for generating TLS certificates. [ provided , certmanager , auto]                                                         | `auto`                                  |
| `egressgatewayController.tls.secretName`                               | The secret name for storing TLS certificates                                                                                         | `template-controller-server-certs`      |
| `egressgatewayController.tls.certmanager.certValidityDuration`         | Generated certificates validity duration in days for 'certmanager' method                                                            | `365`                                   |
| `egressgatewayController.tls.certmanager.issuerName`                   | Issuer name of cert manager 'certmanager'. If not specified, a CA issuer will be created.                                            | `""`                                    |
| `egressgatewayController.tls.certmanager.extraDnsNames`                | Extra DNS names added to certificate when it's auto generated                                                                        | `[]`                                    |
| `egressgatewayController.tls.certmanager.extraIPAddresses`             | Extra IP addresses added to certificate when it's auto generated                                                                     | `[]`                                    |
| `egressgatewayController.tls.provided.tlsCert`                         | Encoded tls certificate for provided method                                                                                          | `""`                                    |
| `egressgatewayController.tls.provided.tlsKey`                          | Encoded tls key for provided method                                                                                                  | `""`                                    |
| `egressgatewayController.tls.provided.tlsCa`                           | Encoded tls CA for provided method                                                                                                   | `""`                                    |
| `egressgatewayController.tls.auto.caExpiration`                        | CA expiration for auto method                                                                                                        | `73000`                                 |
| `egressgatewayController.tls.auto.certExpiration`                      | Server cert expiration for auto method                                                                                               | `73000`                                 |
| `egressgatewayController.tls.auto.extraIpAddresses`                    | Extra IP addresses of server certificate for auto method                                                                             | `[]`                                    |
| `egressgatewayController.tls.auto.extraDnsNames`                       | Extra DNS names of server cert for auto method                                                                                       | `[]`                                    |
