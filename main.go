package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	as "github.com/banzaicloud/hollowtrees/actionserver"
	"github.com/banzaicloud/ht-k8s-action-plugin/plugin"
	log "github.com/sirupsen/logrus"
)

var (
	logLevel          = flag.String("log.level", "info", "log level")
	bindAddr          = flag.String("bind.address", ":8080", "Bind address where the gRPC API is listening")
	clusterConfigRoot = flag.String("cluster.config.root", filepath.Join(os.Getenv("HOME"), ".kube"), "Root location that contains multiple k8s cluster config files")
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
	log.Infof("Root location of k8s configs: %s", *clusterConfigRoot)
	return &K8sAlertHandler{
		Router: &plugin.EventRouter{
			ClusterConfRoot: *clusterConfigRoot,
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
