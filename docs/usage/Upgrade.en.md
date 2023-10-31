# Upgrade

This document will guide you on how to use the `helm upgrade` command to upgrade EgressGateway.

### Basic command format

```shell
helm upgrade [RELEASE] [CHART] [flags]
```

Here, `[RELEASE]` represents the application name set during installation, `[CHART]` refers to the chart, and `[flags]` can specify additional parameters. To learn more about the parameters for `helm upgrade`, please refer to the [helm upgrade](https://helm.sh/docs/helm/helm_upgrade/) page.

### Version upgrade

Follow these steps to perform a version upgrade:

1. Before upgrading, run the following command to update the local Chart to the latest version:

    ```shell
    helm repo update
    ```

2. View the latest version:

    ```shell
    helm search repo egressgateway
    ```

3. Execute the upgrade command:

    ```shell
    helm upgrade \
      egress \
      egressgateway/egressgateway \
      --reuse-values \
      --version [version]
    ```

   Replace `[version]` with the version you want to update.

### Configuration upgrade

Follow these steps to perform a configuration upgrade:

1. To view the available values parameters, visit the [values](https://github.com/spidernet-io/egressgateway/tree/main/charts) documentation.

2. Update the configuration using the `--set` flags. The following example shows how to change the egress agent log level to debug level. By using the `--reuse-values` parameter, you can reuse the values from the previous release and merge any overrides from the command line.

    ```shell
    helm upgrade \
      egress \
      egressgateway/egressgateway \
      --set agent.debug.logLevel=debug \
      --reuse-values
    ```
