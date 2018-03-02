package plugin

import (
	"encoding/json"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Drainer struct {
	c *kubernetes.Clientset
}

func NewDrainer(configRoot string, clusterName string) (*Drainer, error) {
	var kubeConfig string
	if clusterName == "" {
		kubeConfig = filepath.Join(configRoot, "config")
	} else {
		kubeConfig = filepath.Join(configRoot, clusterName, "config")
	}
	log.Infof("Using kubernetes config: %s", kubeConfig)
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		log.Errorf("Failed to build k8s config: %s", err.Error())
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("Failed to create kubernetes clientset: %s\n", err.Error())
		return nil, err
	}
	return &Drainer{
		c: clientset,
	}, nil
}

func (d *Drainer) DrainNode(nodeName string) error {
	err := d.CordonNode(nodeName)
	if err != nil {
		return err
	}
	err = d.DeletePodsOnNode(nodeName)
	if err != nil {
		return err
	}
	log.Infof("node '%s' was drained successfully\n", nodeName)
	return nil
}

func (d *Drainer) CordonNode(nodeName string) error {
	node, err := d.c.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
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
	node, err = d.c.CoreV1().Nodes().Patch(nodeName, types.MergePatchType, patch)
	if err != nil {
		log.Errorf("couldn't cordon node '%s' %s\n", nodeName, err.Error())
		return err
	}
	log.Infof("node '%s' cordoned successfully: node.Spec.Unschedulable=%v\n", nodeName, node.Spec.Unschedulable)
	return nil
}

func (d *Drainer) DeletePodsOnNode(node string) error {
	pods, err := d.selectPodsToDelete(node)
	// TODO: use eviction API to gracefully drain node if supported
	var wg sync.WaitGroup
	for _, pod := range pods {
		wg.Add(1)
		go func(pod v1.Pod) {
			defer wg.Done()
			err = d.c.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			if err != nil {
				log.Errorf("couldn't delete pod %s from node %s: %s\n", pod.Name, node, err.Error())
				return
			}
			log.Infof("deleted pod %s from node %s\n", pod.Name, node)
		}(pod)
	}
	wg.Wait()
	return nil
}

func (d *Drainer) selectPodsToDelete(node string) ([]v1.Pod, error) {
	// TODO: filter pods of daemonsets, mirrorPods, localStorage, unreplicated
	pods, err := d.c.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node}).String()})
	if err != nil {
		log.Errorf("couldn't get pods for node %s\n", err.Error())
		return nil, err
	}
	return pods.Items, nil
}
