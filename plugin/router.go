package plugin

import (
	as "github.com/banzaicloud/hollowtrees/actionserver"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/ht-k8s-action-plugin/conf"
	"github.com/ericchiang/k8s"
	"fmt"
	"context"
)

var log *logrus.Entry

func init() {
	log = conf.Logger().WithField("package", "plugin")
}

func RouteEvent(event *as.AlertEvent) error {
	switch event.EventType {
	case "prometheus.server.alert.SpotTerminationNotice":
		fmt.Println("I got a spot termination notice")
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
	}
	return nil
}
