<!--
# E2E Cases for EgressGateway
- all case about check the `eip` will include tcp, udp and web socket

| Case ID | Title                                                                                                                                         | Priority  | Smoke | Status | Other |
|---------|-----------------------------------------------------------------------------------------------------------------------------------------------|-----------|-------|--------|-------|
| G00001  | Creating an EgressGateway fails when using invalid `Ippools`                                                                                  | p2        | true  |        |       |
| G00002  | Creation of EgressGateway fails when `NodeSelector` is empty                                                                                  | p2        | true  |        |       |
| G00003  | Creation of EgressGateway fails when `DefaultEIP` is not in scope of `Ippools`                                                                | p2        | true  |        |       |
| G00004  | Creation of EgressGateway fails when the number of `Ippools.IPv4` and `Ippools.IPv6` does not match                                           | p2        | true  |        |       |
| G00005  | When `DefaultEIP` is empty, when creating EgressGateway, this field will randomly assign an IP from `Ippools`                                 | p2        | true  |        |       |
| G00006  | When `Ippools` is a single IP, the EgressGateway is successfully created and the `EgressGatewayStatus` check is passed                        | p2        | true  |        |       |
| G00007  | When `Ippools` is an IP range in `a-b` format, the EgressGateway is successfully created and the `EgressGatewayStatus` check passes           | p2        | true  |        |       |
| G00008  | When `Ippools` is in IP cidr format, EgressGateway is successfully created and `EgressGatewayStatus` check is passed                          | p2        | true  |        |       |
| G00009  | Updating EgressGateway fails when adding invalid IP addresses to `Ippools`                                                                    | p2        | true  |        |       |
| G00010  | Updating EgressGateway fails when removing IP addresses in use from `Ippools`                                                                 | p2        | true  |        |       |
| G00011  | Updating EgressGateway fails when adding different number of IPs to `Ippools.IPv4` and `Ippools.IPv6`                                         | p2        | true  |        |       |
| G00012  | Add legal IP address to `Ippools`, update EgressGateway successfully                                                                          | p2        | true  |        |       |
| G00013  | When `NodeSelector` is edited, `egressGatewayStatus` updates correctly as expected                                                            | p2        | true  |        |       |
| G00014  | Deleting an EgressGateway fails when there is a Policy (both at the namespace level and at the cluster level) that is using the EgressGateway | p2        | true  |        |       |
| G00015  | When EgressGateway is not used by Policy, the EgressGateway is deleted successfully                                                           | p2        | true  |        |       |
-->

# EgressGateway E2E 用例
- 用例中，所有有关 `eip` 校验的内容，都包含了 tcp，udp 和 web socket

| 用例编号   | 标题                                                                            | 优先级   | 冒烟   | 状态   | 其他  |
|--------|-------------------------------------------------------------------------------|-------|------|------|-----|
| G00001 | 使用不合法的 `Ippools` 时，创建 EgressGateway 会失败                                       | p2    | true |      |     |
| G00002 | 当 `NodeSelector` 为空时，创建 EgressGateway 会失败                                     | p2    | true |      |     |
| G00003 | 当 `DefaultEIP` 不在 `Ippools` 范围内，创建 EgressGateway 会失败                          | p2    | true |      |     |
| G00004 | 当 `Ippools.IPv4` 和 `Ippools.IPv6` 的数量不一致时，创建 EgressGateway 会失败                | p2    | true |      |     |
| G00005 | 当 `DefaultEIP` 为空，创建 EgressGateway 时，此字段会从 `Ippools` 中随机分配一个 IP               | p2    | true |      |     |
| G00006 | 当 `Ippools` 为单个 IP 时，创建 EgressGateway 成功，`EgressGatewayStatus` 检查通过           | p2    | true |      |     |
| G00007 | 当 `Ippools` 是 `a-b` 格式的 IP 范围时，创建 EgressGateway 成功，`EgressGatewayStatus` 检查通过 | p2    | true |      |     |
| G00008 | 当 `Ippools` 是 IP cidr 格式时，创建 EgressGateway 成功，`EgressGatewayStatus` 检查通过      | p2    | true |      |     |
| G00009 | 向 `Ippools` 中添加 不合法的 IP 地址时，更新 EgressGateway 会失败                              | p2    | true |      |     |
| G00010 | 从 `Ippools` 中删除 正在被使用的 IP 地址时，更新 EgressGateway 会失败                            | p2    | true |      |     |
| G00011 | 向 `Ippools.IPv4` 和 `Ippools.IPv6` 添加不同数量的 IP 时，更新 EgressGateway 会失败           | p2    | true |      |     |
| G00012 | 向 `Ippools` 中添加合法的 IP 地址，更新 EgressGateway 成功                                  | p2    | true |      |     |
| G00013 | 当 `NodeSelector` 被编辑后，`egressGatewayStatus` 会如预期更新正确                          | p2    | true |      |     |
| G00014 | 当存在 Policy （包括命名空间级别和集群级别）正在使用 EgressGateway 时，删除 EgressGateway 会失败           | p2    | true |      |     |
| G00015 | 当没有 Policy 使用 EgressGateway 时，删除 EgressGateway 成功                             | p2    | true |      |     |
