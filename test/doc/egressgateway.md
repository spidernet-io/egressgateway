# E2E Cases for EgressGateway

| Case ID | Title                                                                                             | Priority | Smoke | Status | Other |
|---------|---------------------------------------------------------------------------------------------------|----------|-------|--------| ----- |
| G00001  | create egressgateway with valid and invalid parameters                                            | p2       | true  |        |       |
| G00002  | edit egressgateway spec.nodeSelector, the status and gateway IP should be changed correspondingly | p2       | true  |        |       |
| G00003  | delete egressgateway, expected to deleted succeeded and the export IP will not the gateway IP     | p2       | true  |        |       |