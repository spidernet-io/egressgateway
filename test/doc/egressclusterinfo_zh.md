<!--
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

-->

# EgressClusterInfo E2E 用例

| Case ID | Title                                                                                  | Priority | Smoke | Status | Other |
|---------|----------------------------------------------------------------------------------------|----------|-------|--------|-------|
| I00001  | 获取 egressClusterInfo `default`，status 中数据应该与集群中数据一直                                    | p2       | false |        |       |
| I00002  | 编辑 egressClusterInfo `default` 中 `spec.autoDetect.clusterIP`，`status.clusterIP` 数据更新正确 | p2       | false |        |       |
| I00003  | 编辑 egressClusterInfo `default` 中 `spec.autoDetect.nodeIP`，`status.nodeIP` 数据更新正确       | p2       | false |        |       |
| I00004  | 编辑 egressClusterInfo `default` 中 `spec.autoDetect.podCidrMode`，`status.podCIDR` 数据更新正确 | p2       | false |        |       |
| I00005  | 编辑 egressClusterInfo `default` 中 `spec.extraCidr`，`status.extraCidr` 数据更新正确            | p2       | false |        |       |
| I00006  | 当 `spec.autoDetect.clusterIP` 为 `calico` 时，集群中添加或删除 `ippool`，`status.podCIDR` 更新正确     | p2       | false |        |       |
| I00007  | 删除 egressClusterInfo `default`，会失败                                                     | p2       | false |        |       |