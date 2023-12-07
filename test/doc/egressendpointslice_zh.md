<!--
# E2E Cases for EgressEndpointSlice

| Case ID | Title                                                                                                                                | Priority | Smoke | Status | Other |
|---------|--------------------------------------------------------------------------------------------------------------------------------------|----------|-------|--------|-------|
| S00001  | After repeatedly creating and deleting `pods` matched by a specific `policy`, check the egress IP addresses of all `pods`                         | p2       | false |        |       |

-->

# EgressEndpointSlice E2E 用例

| Case ID | Title                                                                                  | Priority | Smoke | Status | Other |
|---------|----------------------------------------------------------------------------------------|----------|-------|--------|-------|
| S00001  | 不断增删 `policy` 所匹配到的 `pods`，几轮增删后，判断所有 `pods` 的出口 `IP`                                    | p2       | false |        |       |