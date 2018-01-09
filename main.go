package main

import (
	"context"
	"fmt"

	as "github.com/banzaicloud/hollowtrees/actionserver"
	"github.com/banzaicloud/ht-k8s-action-plugin/conf"
	"github.com/banzaicloud/ht-k8s-action-plugin/plugin"
	"github.com/ericchiang/k8s"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log *logrus.Entry

func init() {
	log = conf.Logger().WithField("package", "main")
}

type K8sAlertHandler struct {
	Router *plugin.EventRouter
}

func newK8sAlertHandler() *K8sAlertHandler {
	client, err := k8s.NewInClusterClient()
	if err != nil {
		log.Fatal(err)
	}
	return &K8sAlertHandler{
		Router: &plugin.EventRouter{
			Client: client,
		},
	}
}

func (d *K8sAlertHandler) Handle(event *as.AlertEvent) (*as.ActionResult, error) {
	fmt.Printf("got GRPC request, handling alert: %v\n", event)
	err := d.Router.RouteEvent(event)
	if err != nil {
		return nil, err
	}
	return &as.ActionResult{Status: "ok"}, nil
}

func main() {
	port := viper.GetInt("plugin.port")
	fmt.Printf("Starting Hollowtrees ActionServer on port %d\n", port)


	client, err := k8s.NewInClusterClient()
	if err != nil {
		log.Fatal(err)
	}

	// get pods on node
	node, err := client.CoreV1().GetNode(context.Background(), "minikube")
	if err != nil {
		log.Errorf("error: %v", err)
	}
	fmt.Println(node)

	as.Serve(port, newK8sAlertHandler())
}
