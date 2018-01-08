package main

import (
	"fmt"

	as "github.com/banzaicloud/hollowtrees/actionserver"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/ht-k8s-action-plugin/conf"
	"github.com/banzaicloud/ht-k8s-action-plugin/plugin"
	"github.com/spf13/viper"
	"github.com/ericchiang/k8s"
	"context"
)

var log *logrus.Entry

func init() {
	log = conf.Logger().WithField("package", "main")
}

type K8sAlertHandler struct {
	// TODO: add k8s config?
}

func newK8sAlertHandler() *K8sAlertHandler {
	return &K8sAlertHandler{}
}

func (d *K8sAlertHandler) Handle(event *as.AlertEvent) (*as.ActionResult, error) {
	fmt.Printf("got GRPC request, handling alert: %v\n", event)
	err := plugin.RouteEvent(event)
	if err != nil {
		return nil, err
	}
	return &as.ActionResult{Status: "ok"}, nil
}

func main() {
	port := viper.GetInt("plugin.port")
	fmt.Printf("Starting Hollowtrees ActionServer on port %d", port)
	client, err := k8s.NewInClusterClient()
	if err != nil {
		log.Fatal(err)
	}
	nodes, err := client.CoreV1().ListNodes(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, node := range nodes.Items {
		fmt.Printf("name=%q schedulable=%t\n", *node.Metadata.Name, !*node.Spec.Unschedulable)
	}
	as.Serve(port, newK8sAlertHandler())
}
