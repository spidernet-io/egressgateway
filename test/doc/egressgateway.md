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
