package plugin

import (
	"fmt"

	as "github.com/banzaicloud/hollowtrees/actionserver"
	"github.com/banzaicloud/ht-k8s-action-plugin/conf"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

type EventRouter struct {
	Clientset *kubernetes.Clientset
}

var log *logrus.Entry

func init() {
	log = conf.Logger().WithField("package", "plugin")
}

func (r *EventRouter) RouteEvent(event *as.AlertEvent) error {
	switch event.EventType {
	case "prometheus.server.alert.SpotTerminationNotice":
		log.Infof("Received %s", event.EventType)
		err := r.drainNode(event.Data["instance"])
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *EventRouter) drainNode(nodeName string) error {

	// TODO: cordon node: https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/drain.go#L701?

	podList, err := r.Clientset.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String()})
	if err != nil {
		log.Errorf("couldn't get pods for node: %s", err.Error())
		return err
	}

	// TODO: use eviction API to drain node

	for _, pod := range podList.Items {
		fmt.Println(pod.Name, pod.UID)
	}
}