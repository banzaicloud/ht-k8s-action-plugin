package plugin

import (
	"encoding/json"
	"sync"

	as "github.com/banzaicloud/hollowtrees/actionserver"
	"github.com/banzaicloud/ht-k8s-action-plugin/conf"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
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
	err = r.DeletePodsOnNode(nodeName)
	if err != nil {
		return err
	}
	log.Infof("node '%s' was drained successfully\n", nodeName)
	return nil
}

func (r *EventRouter) CordonNode(nodeName string) error {
	node, err := r.Clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("couldn't cordon node '%s': %s\n", nodeName, err.Error())
		return err
	}
	if node.Spec.Unschedulable {
		log.Infof("node '%s' is already unschedulable.\n", nodeName)
		return nil
	}
	oldData, err := json.Marshal(*node)
	node.Spec.Unschedulable = true
	newData, err := json.Marshal(*node)
	patch, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, *node)
	if err != nil {
		log.Errorf("couldn't cordon node '%s': %s\n", nodeName, err.Error())
		return err
	}
	node, err = r.Clientset.CoreV1().Nodes().Patch(nodeName, types.MergePatchType, patch)
	if err != nil {
		log.Errorf("couldn't cordon node '%s' %s\n", nodeName, err.Error())
		return err
	}
	log.Infof("node '%s' cordoned successfully: node.Spec.Unschedulable=%v\n", nodeName, node.Spec.Unschedulable)
	return nil
}

func (r *EventRouter) DeletePodsOnNode(nodeName string) error {
	pods, err := r.Clientset.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String()})
	if err != nil {
		log.Errorf("couldn't get pods for node %s\n", err.Error())
		return err
	}

	// TODO: use eviction API to gracefully drain node
	var wg sync.WaitGroup
	for _, pod := range pods.Items {
		wg.Add(1)
		go func(pod v1.Pod) {
			defer wg.Done()
			err = r.Clientset.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			if err != nil {
				log.Errorf("couldn't delete pod %s from node %s: %s\n", pod.Name, nodeName, err.Error())
				return
			}
			log.Infof("deleted pod %s from node %s\n", pod.Name, nodeName)
		}(pod)
	}
	wg.Wait()
	return nil
}
