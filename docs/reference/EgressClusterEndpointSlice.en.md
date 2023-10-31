The EgressClusterEndpointSlice CRD is used to aggregate address information of Pods matched by EgressClusterPolicy. This resource is for internal use only, aiming to improve the performance of the control plane.

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressClusterEndpointSlice
metadata:
  generateName: cluster-policy-
  labels:
    spidernet.io/policy-name: cluster-policy          # (1)
  name: cluster-policy-zp667
  ownerReferences:
    - apiVersion: egressgateway.spidernet.io/v1beta1  # (2)
      blockOwnerDeletion: true
      controller: true
      kind: EgressClusterPolicy
      name: cluster-policy
      uid: fdca1dd5-9c3b-4d58-b043-451e10f15ea8
endpoints:                                             # (3)
  - ipv4:
      - 10.21.60.74                                    # (4)
    ipv6:
      - fd00:21::5328:9c2:3579:8cca                    # (5)
    node: workstation3                                 # (6)
    ns: ns1                                            # (7)
    pod: ns2-mock-app-5c4cd6bb87-g4fdj                 # (8)
```

1. This label value indicates the EgressClusterPolicy to which the EgressClusterEndpointSlice belongs.
2. By using `ownerReferences`, the CRD is associated with its parent resource, enabling automatic recycling of EgressClusterEndpointSlice when the EgressClusterPolicy is deleted.
3. The EgressClusterEndpointSlice object is used to summarize the address information of Pods matched by EgressClusterPolicy. By default, a new EgressClusterEndpointSlice is created when there are more than 100 matched results.
4. The IPv4 address list of Pods.
5. The IPv6 address list of Pods.
6. Information about the node where the Pods are located.
7. Information about the tenant to which the Pods belong.
8. The names of the Pods.