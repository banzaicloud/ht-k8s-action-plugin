package main

import (
	"fmt"

	as "github.com/banzaicloud/hollowtrees/actionserver"
	"github.com/banzaicloud/ht-k8s-action-plugin/conf"
	"github.com/banzaicloud/ht-k8s-action-plugin/plugin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var log *logrus.Entry

func init() {
	log = conf.Logger().WithField("package", "main")
}

type K8sAlertHandler struct {
	Router *plugin.EventRouter
}

func newK8sAlertHandler() *K8sAlertHandler {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create in cluster configuration: %s\n", err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create kubernetes clientset: %s\n", err.Error())
	}
	return &K8sAlertHandler{
		Router: &plugin.EventRouter{
			Clientset: clientset,
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

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create in cluster configuration: %s\n", err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create kubernetes clientset: %s\n", err.Error())
	}

	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d nodes in the cluster\n", len(nodes.Items))
	for _, n := range nodes.Items {
		fmt.Println(n.Name, n.Status)
	}

	as.Serve(port, newK8sAlertHandler())
}
