package main

import (
	"flag"
	"fmt"

	as "github.com/banzaicloud/hollowtrees/actionserver"
	"github.com/banzaicloud/ht-k8s-action-plugin/plugin"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	logLevel = flag.String("log.level", "info", "log level")
	bindAddr = flag.String("bind.address", ":80", "Bind address where the gRPC API is listening")
)

func init() {
	flag.Parse()
	parsedLevel, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.WithError(err).Warnf("Couldn't parse log level, using default: %s", log.GetLevel())
	} else {
		log.SetLevel(parsedLevel)
		log.Debugf("Set log level to %s", parsedLevel)
	}
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
	fmt.Printf("Starting Hollowtrees ActionServer on %s\n", *bindAddr)
	as.Serve(*bindAddr, newK8sAlertHandler())
}
