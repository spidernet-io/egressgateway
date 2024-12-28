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
