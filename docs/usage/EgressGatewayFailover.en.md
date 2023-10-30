# EgressGateway Failover

## Controller Failover

When the EgressGateway controller fails over, you can control the number of Controller replicas by specifying the `controller.replicas` parameter during installation. If one of the replicas in multiple Controller replicas fails, the system will automatically elect another replica as the primary controller to ensure service continuity.

## Datapath Failover

When handling datapath failover, creating an `EgressGateway` can use `nodeSelector` to select a set of nodes as Egress Nodes. The Egress IP will be bound to one of these nodes. When a node fails or the Egress Agent on a node fails, the Egress IP will automatically move to another available node to ensure service continuity and reliability.

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressGateway
metadata: 
  name: egw1
spec:
  clusterDefault: true
  ippools:
    ipv4:
      - 10.6.1.55
      - 10.6.1.56 
    ipv4DefaultEIP: 10.6.1.56
    ipv6:
      - fd00::55
      - fd00::56
    ipv6DefaultEIP: fd00::55
  nodeSelector:
    selector:
      matchLabels:
        egress: "true"
status:
  nodeList:
    - eips: 
        - ipv4: 10.6.1.56
          ipv6: fd00::55
          policies:
            - name: policy1
              namespace: default
      name: workstation2
      status: Ready
    - name: workstation3
      status: Ready
```

The timeout for health checks and Egress IP failover can be tuned via Helm values configuration.

* `feature.tunnelMonitorPeriod` The egress controller check tunnel last update status at an interval set in seconds, default `5`.
* `feature.tunnelUpdatePeriod` The egress agent updates the tunnel status at an interval set in seconds, default `5`.
* `feature.eipEvictionTimeout` If the last updated time of the egress tunnel exceeds this time, move the Egress IP of the node to an available node, the unit is seconds, default is `15`.

Datapath Failover troubleshooting steps:

1. First, check the installation configuration file `values.yaml` of the EgressGateway application to ensure failover related configurations are set reasonably, in particular ensuring `eipEvictionTimeout` is greater than the sum of `tunnelMonitorPeriod` and `tunnelUpdatePeriod`.
2. Execute `kubectl get egt -w` to check the status of `EgressTunnel`. Check if the selected Node is in `HeartbeatTimeout` state, and if there are other `EgressTunnel` in `Ready` state.
3. If you want to check if there has been an IP switch caused by HeartbeatTimeout, you can retrieve the logs related to `update tunnel status to HeartbeatTimeout` in the controller container.
