# E2E Cases for EgressGatewayPolicy

| Case ID | Title                                                                                                              | Priority | Smoke | Status | Other |
|---------|--------------------------------------------------------------------------------------------------------------------|----------|-------|--------| ----- |
| P00001  | create egressgatewaypolicy, expected succeeded                                                                     | p2       | true  |        |       |
| P00002  | edit egressgatewaypolicy matchLabels and destSubnet, the export IP of the test pod will be changed correspondingly | p2       | true  |        |       |
| P00003  | delete egressgatewaypolicy, expected to deleted succeeded and the test pod's export IP will not be the gateway IP  | p2       | true  |        |       |