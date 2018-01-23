### Kubernetes Hollowtrees action plugin

This is an action plugin for the [Hollowtrees](https://github.com/banzaicloud/hollowtrees) project.
It interacts with the Kubernetes API to handle different Hollowtrees events in a K8S cluster. 

When started it is listening on a GRPC interface and accepts Hollowtrees events.

### Quick start

The project can be build with `make`. To create the binary locally, run:
```
make build
```
It creates the binary in the `./bin` directory.

To create a docker image that can be run on Kubernetes run:
```
make docker-build
```
It creates a docker image with the name `banzaicloud/ht-k8s-action-plugin:$(GIT_REVISION)`

### Configuration

The following options can be configured when starting the action plugin. Configuration is done through a `plugin-config.toml` file that can be placed in `conf/` or near the binary:

```
[log]
    format = "text"
    level = "debug"

[plugin]
    port = "8887"
```

Instead of using a configuration file, environment variables can be used as well with the prefix `HTPLUGIN`. For example to configure the port where the application will listen, use the environment variable `HTPLUGIN_PLUGIN_PORT`.

The project is using an in-cluster config to interact with Kubernetes so it must be deployed accordingly.

To run:
```
kubectl run ht-k8s-action-plugin --image=banzaicloud/ht-k8s-action-plugin:$GITREV  --port=8887
```

### Event types that the plugin can understand:

`prometheus.server.alert.SpotTerminationNotice` - Prepares a node for termination by draining it similarily to `kubectl drain`. It cordons the Kubernetes node by making it unschedulable (`node.Spec.Unschedulable = true`) and evicts or deletes the pods on the node that will be terminated.
