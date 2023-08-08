# E2E Cases for EgressClusterInfo

| Case ID | Title                                                                                                                                | Priority | Smoke | Status | Other |
|---------|--------------------------------------------------------------------------------------------------------------------------------------|----------|-------|--------|-------|
| I00001  | Get egressClusterInfo `default`, the data in the status should be consistent with the data in the cluster                            | p2       | false |        |       |
| I00002  | Edit `spec.autoDetect.clusterIP` in egressClusterInfo `default`, `status.clusterIP` data update is correct                           | p2       | false |        |       |
| I00003  | Edit `spec.autoDetect.nodeIP` in egressClusterInfo `default`, `status.nodeIP` data update is correct                                 | p2       | false |        |       |
| I00004  | Edit `spec.autoDetect.podCidrMode` in egressClusterInfo `default`, `status.podCIDR` data update is correct                           | p2       | false |        |       |
| I00005  | Edit `spec.extraCidr` in egressClusterInfo `default`, `status.extraCidr` data is updated correctly                                   | p2       | false |        |       |
| I00006  | When `spec.autoDetect.clusterIP` is `calico`, `ippool` is added or deleted in the cluster, and `status.podCIDR` is updated correctly | p2       | false |        |       |
| I00007  | Remove egressClusterInfo `default`, will fail                                                                                        | p2       | false |        |       |