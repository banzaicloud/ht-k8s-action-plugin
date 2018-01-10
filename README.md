## Hollowtrees Kubernetes action plugin

Hollowtrees **action plugins** are microservices that can be configured in the [Hollowtrees rule engine](https://github.com/banzaicloud/hollowtrees) to react to specific events that occur in a cloud native environment.
Plugins are listening on a *GRPC* interface.
This specific plugin is used to interact with Kubernetes on specific event triggers.
It must be deployed inside a Kubernetes cluster, because it uses the `InCluster` configuration to interact with the Kubernetes API.

### Supported events

Currently only one type of event is supported: `prometheus.server.alert.SpotTerminationNotice`

* when this event is sent to the plugin, it makes the node unschedulable and evicts all running pods to prepare for the node termination
