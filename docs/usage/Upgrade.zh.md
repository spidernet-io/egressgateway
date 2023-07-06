本文档将指导你如何使用 `helm upgrade` 命令完成 EgressGateway 的升级。

### 基本命令格式

```shell
helm upgrade [RELEASE] [CHART] [flags]
```

其中，`[RELEASE]` 代表安装时设置的应用名称，`[CHART]` 指的是图表，而 `[flags]` 可以指定额外的参数。如需了解更多有关 `helm upgrade` 的参数，请参阅 [helm upgrade](https://helm.sh/docs/helm/helm_upgrade/) 页面。

### 版本升级

按照以下步骤执行版本升级：

1. 在升级之前执行以下命令以将本地 Chart 升级至最新版本：

    ```shell
    helm repo update
    ```

2. 查看最新的版本：

    ```shell
    helm search repo egressgateway
    ```

3. 执行升级命令：

    ```shell
    helm upgrade \
      egress \
      egressgateway/egressgateway \
      --reuse-values \
      --version [version]
    ```

   将 `[version]` 替换为你希望更新的版本。

### 配置升级

按照以下步骤执行配置升级：

1. 查看可用的 values 参数，请访问 [values](https://github.com/spidernet-io/egressgateway/tree/main/charts) 说明文档。

2. 使用 `--set` flags 更新配置。以下示例展示了如何将 egress agent 日志等级更改为 debug 级别。通过 `--reuse-values` 参数，你可以在升级时重用上一个 release 的值并合并来自命令行的任何覆盖。

    ```shell
    helm upgrade \
      egress \
      egressgateway/egressgateway \
      --set agent.debug.logLevel=debug \
      --reuse-values
    ```
