package plugin

import (
	"encoding/json"
	"fmt"

	as "github.com/banzaicloud/hollowtrees/actionserver"
	"github.com/banzaicloud/ht-k8s-action-plugin/conf"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
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
		err := r.DrainNode(event.Data["instance"])
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *EventRouter) DrainNode(nodeName string) error {

	err := r.CordonNode(nodeName)
	if err != nil {
		return err
	}

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
	return nil
}

func (r *EventRouter) CordonNode(nodeName string) error {
	node, err := r.Clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("couldn't cordon node '%s': %s", nodeName, err.Error())
		return err
	}
	if node.Spec.Unschedulable {
		log.Infof("node '%s' is already unschedulable.", nodeName)
		return nil
	}
	oldData, err := json.Marshal(*node)
	node.Spec.Unschedulable = true
	newData, err := json.Marshal(*node)
	patch, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, *node)
	if err != nil {
		log.Errorf("couldn't cordon node '%s': %s", nodeName, err.Error())
		return err
	}
	node, err = r.Clientset.CoreV1().Nodes().Patch(nodeName, types.MergePatchType, patch)
	if err != nil {
		log.Errorf("couldn't cordon node '%s': %s", nodeName, err.Error())
		return err
	}
	log.Infof("node '%s' cordoned successfully: node.Spec.Unschedulable=%v\n", nodeName, node.Spec.Unschedulable)
	return nil
}
