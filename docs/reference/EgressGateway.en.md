The EgressGateway CRD is used to select a group of nodes as the Egress nodes of the cluster and configure the Egress IP pool for this group of nodes. The Egress IP can fall within this range. Cluster scope resource.

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressGateway
metadata:
  name: "eg1"
spec:
  ippools:                     
    ipv4:                       
      - "10.6.1.55"
      - "10.6.1.60-10.6.1.65"
      - "10.6.1.70/28"
    ipv6:                      
      - ""
    ipv4DefaultEIP: ""
    ipv6DefaultEIP: ""
  nodeSelector:
    selector:
      matchLabels:
        egress: "true"
    policy: "doing"
status:
  nodeList:
    - name: "node1"
      status: "Ready"
      epis:
        - ipv4: "10.6.1.55"
          ipv6: "fd00::55"
          policies:
            - name: "app"
              namespace: "default"
```

## Definition

### Metadata

| Field | Description                             | Schema | Validation |
|-------|-----------------------------------------|--------|------------|
| name  | The name of this EgressGateway resource | string | required   |

### Spec

| Field          | Description                                                | Schema                        | Validation | Values     | Default |
|----------------|------------------------------------------------------------|-------------------------------|------------|------------|---------|
| ippools        | Set the range of egress IP pool that EgressGateway can use | [ippools](#ippools)           | optional   |            |         |
| nodeSelector   | Match egress nodes by label                                | [nodeSelector](#nodeSelector) | require    |            |         |
| clusterDefault | Default EgressGateway for the cluster                      | bool                          | optional   | true/false | false   |

#### ippools

| Field          | Description                                                                                                                                                              | Schema   | Validation | Values                                          | Default |
|----------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|------------|-------------------------------------------------|---------|
| ipv4           | IPv4 pool                                                                                                                                                                | []string | optional   | `10.6.0.1` `10.6.0.1-10.6.0.10` ``10.6.0.1/26`` |         |
| ipv6           | IPv6 pool                                                                                                                                                                | []string | optional   | `fd::01` `fd01::01-fd01:0a` `fd10:01/64`        |         |
| ipv4DefaultEIP | Default egress IPv4, if the EgressPolicy does not specify EIP and the EIP assignment policy is `default`, the EIP assigned to this EgressPolicy will be `ipv4DefaultEIP` | string   | optional   |                                                 |         |
| ipv6DefaultEIP | Default egress IPv6, the rules are the same as `ipv6DefaultEIP`                                                                                                          | string   | optional   |                                                 |         |

### nodeSelector

| Field                | Description       | Schema            | Validation | Values | Default |
|----------------------|-------------------|-------------------|------------|--------|---------|
| selector.matchLabels | Node match labels | map[string]string | optional   |        |         |


### Status (subresource)

| Field    | Description     | Schema                | Validation | Values | Default |
|----------|-----------------|-----------------------|------------|--------|---------|
| nodeList | Match node list | [nodeList](#nodeList) | optional   |        |         |


#### nodeList

| Field  | Description                | Schema        | Validation | Values              | Default |
|--------|----------------------------|---------------|------------|---------------------|---------|
| name   | Name of the node           | string        | optional   |                     |         |
| status | Current status of the node | string        | optional   | `Ready`, `NotReady` |         |
| epis   | List of endpoint IPs       | [epis](#epis) | optional   |                     |         |

##### epis

| Field    | Description                                                               | Schema                | Validation | Values | Default |
|----------|---------------------------------------------------------------------------|-----------------------|------------|--------|---------|
| ipv4     | If EgressPolicy and EgressClusterPolicy use node IP, this field is empty. | string                | optional   |        |         |
| ipv6     | In the dual-stack situation, IPv4 and IPv6 are one-to-one corresponding.  | string                | optional   |        |         |
| policies | Policy list of the node                                                   | [policies](#policies) | optional   |        |         |

##### policies

| Field      | Description                  | Schema     | Validation | Values     | Default |
|------------|------------------------------|------------|------------|------------|---------|
| name       | Name of the policy           | string     | optional   |            |         |
| namespace  | Namespace of the policy      | string     | optional   |            |         |