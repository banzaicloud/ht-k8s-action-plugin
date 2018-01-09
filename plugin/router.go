package plugin

import (
	"fmt"

	as "github.com/banzaicloud/hollowtrees/actionserver"
	"github.com/banzaicloud/ht-k8s-action-plugin/conf"
	"github.com/ericchiang/k8s"
	"github.com/sirupsen/logrus"
)

type EventRouter struct {
	Client *k8s.Client
}

var log *logrus.Entry

func init() {
	log = conf.Logger().WithField("package", "plugin")
}


func (r *EventRouter) RouteEvent(event *as.AlertEvent) error {
	switch event.EventType {
	case "prometheus.server.alert.SpotTerminationNotice":
		fmt.Println("I got a spot termination notice")
		//
		//// get pods on node
		//node, err := r.Client.CoreV1().GetNode(context.Background(), "minikube")
		//if err != nil {
		//	log.Errorf("error: %v", err)
		//}
		//fmt.Println(node)
		//var fs fieldSelectorOption = "spec.nodeName=minikube"
		//pods, err := r.Client.CoreV1().ListPods(context.Background(), "default", fs)
		//
		//for _, pod := range pods.Items {
		//	fmt.Printf("name=%s status=%s\n", *pod.Metadata.Name, *pod.Status.Message)
		//}

		//evict pods
		//nodes, err := r.Client.PolicyV1Beta1().CreateEviction() CoreV1().nodeListNodes(context.Background())
		//if err != nil {
		//	log.Fatal(err)
		//}

	}
	return nil
}
