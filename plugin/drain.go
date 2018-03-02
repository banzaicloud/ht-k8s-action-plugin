package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type NodeDrainer struct {
	node               string
	c                  *kubernetes.Clientset
	force              bool
	ignoreDaemonSets   bool
	deleteLocalData    bool
	gracePeriodSeconds int64
	timeout            time.Duration
}

type podFilter func(v1.Pod) (bool, error)

func NewDrainer(configRoot string, clusterName string, nodeName string) (*NodeDrainer, error) {
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
	return &NodeDrainer{
		node:               nodeName,
		c:                  clientset,
		deleteLocalData:    true,
		force:              true,
		ignoreDaemonSets:   true,
		gracePeriodSeconds: 30, // default k8s value
		timeout:            0,
	}, nil
}

func (d *NodeDrainer) DrainNode() error {
	err := d.CordonNode()
	if err != nil {
		return err
	}
	err = d.DeletePodsOnNode()
	if err != nil {
		return err
	}
	log.Infof("node '%s' was drained successfully", d.node)
	return nil
}

func (d *NodeDrainer) CordonNode() error {
	node, err := d.c.CoreV1().Nodes().Get(d.node, metav1.GetOptions{})
	if err != nil {
		log.Errorf("couldn't cordon node '%s': %s\n", d.node, err.Error())
		return err
	}
	if node.Spec.Unschedulable {
		log.Infof("node '%s' is already unschedulable.\n", d.node)
		return nil
	}
	oldData, err := json.Marshal(*node)
	node.Spec.Unschedulable = true
	newData, err := json.Marshal(*node)
	patch, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, *node)
	if err != nil {
		log.Errorf("couldn't cordon node '%s': %s\n", d.node, err.Error())
		return err
	}
	node, err = d.c.CoreV1().Nodes().Patch(d.node, types.MergePatchType, patch)
	if err != nil {
		log.Errorf("couldn't cordon node '%s' %s\n", d.node, err.Error())
		return err
	}
	log.Infof("node '%s' cordoned successfully: node.Spec.Unschedulable=%v\n", d.node, node.Spec.Unschedulable)
	return nil
}

func (d *NodeDrainer) DeletePodsOnNode() error {
	pods, err := d.findPodsToDelete(d.node)
	if err != nil {
		return err
	}

	if pods == nil || len(pods) == 0 {
		log.Infof("there are no pods to delete on the drained node")
		return nil
	}

	policyGroupVersion, err := d.evictionAvailable()
	if err != nil {
		return err
	}

	if len(policyGroupVersion) > 0 {
		log.Infof("eviction API is available, pods will be evicted")
		return d.evictPods(pods, policyGroupVersion)
	} else {
		log.Infof("eviction is not supported, pods will be deleted")
		return d.deletePods(pods)
	}

	return nil
}

func (d *NodeDrainer) findPodsToDelete(node string) ([]v1.Pod, error) {

	allPods, err := d.c.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node}).String()})
	if err != nil {
		return nil, err
	}

	var podsToDelete []v1.Pod
	var podErrors []error

	for _, pod := range allPods.Items {
		deletable := true
		for _, filter := range []podFilter{d.dsFilter, d.mirrorFilter, d.localStorageFilter, d.unreplicatedFilter} {
			ok, err := filter(pod)
			if err != nil {
				podErrors = append(podErrors, err)
			}
			deletable = deletable && ok
		}
		if deletable {
			podsToDelete = append(podsToDelete, pod)
		}
	}

	if len(podErrors) > 0 {
		var errMsg string
		for _, err := range podErrors {
			errMsg += fmt.Sprintf("%s\n", err.Error())
		}
		return nil, errors.New(errMsg)
	}
	return podsToDelete, nil
}

func (d *NodeDrainer) dsFilter(pod v1.Pod) (bool, error) {
	controllerRef := metav1.GetControllerOf(&pod)
	if controllerRef == nil || controllerRef.Kind != "DaemonSet" {
		return true, nil
	}
	if _, err := d.c.ExtensionsV1beta1().DaemonSets(pod.Namespace).Get(controllerRef.Name, metav1.GetOptions{}); err != nil {
		if k8serrors.IsNotFound(err) && d.force {
			log.Warnf("pod %s.%s is controlled by a DaemonSet but the DaemonSet is not found", pod.Namespace, pod.Name)
			return true, nil
		}
		return false, err
	}
	if !d.ignoreDaemonSets {
		return false, errors.New(fmt.Sprintf("pod %s.%s is controlled by a DaemonSet, node cannot be drained (set ignoreDaemonSets=true to drain)", pod.Namespace, pod.Name))
	}
	log.Warnf("pod %s.%s is controlled by a DaemonSet, it won't be deleted", pod.Namespace, pod.Name)
	return false, nil
}

func (d *NodeDrainer) mirrorFilter(pod v1.Pod) (bool, error) {
	if _, found := pod.ObjectMeta.Annotations[v1.MirrorPodAnnotationKey]; found {
		log.Warnf("%s.%s is a mirror pod, it won't be deleted", pod.Namespace, pod.Name)
		return false, nil
	}
	return true, nil
}

func (d *NodeDrainer) localStorageFilter(pod v1.Pod) (bool, error) {
	localStorage := false
	for _, volume := range pod.Spec.Volumes {
		if volume.EmptyDir != nil {
			localStorage = true
			break
		}
	}
	if !localStorage {
		return true, nil
	}
	if !d.deleteLocalData {
		return false, errors.New(fmt.Sprintf("pod %s.%s has local storage, node cannot be drained (set deleteLocalData=true to drain)", pod.Namespace, pod.Name))
	}
	log.Warnf("pod %s.%s has local storage, and it will be deleted because deleteLocalData is set", pod.Namespace, pod.Name)
	return true, nil
}

func (d *NodeDrainer) unreplicatedFilter(pod v1.Pod) (bool, error) {
	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		return true, nil
	}
	controllerRef := metav1.GetControllerOf(&pod)
	if controllerRef != nil {
		return true, nil
	}
	if d.force {
		log.Warnf("pod %s.%s is unreplicated, but it will be deleted because force is set", pod.Namespace, pod.Name)
		return true, nil
	}
	return false, errors.New(fmt.Sprintf("pod %s.%s is unreplicated, node cannot be drained (set force=true to drain)", pod.Namespace, pod.Name))
}

func (d *NodeDrainer) evictionAvailable() (string, error) {
	discoveryClient := d.c.Discovery()
	groupList, err := discoveryClient.ServerGroups()
	if err != nil {
		return "", err
	}
	foundPolicyGroup := false
	var policyGroupVersion string
	for _, group := range groupList.Groups {
		if group.Name == "policy" {
			foundPolicyGroup = true
			policyGroupVersion = group.PreferredVersion.GroupVersion
			break
		}
	}
	if !foundPolicyGroup {
		return "", nil
	}
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion("v1")
	if err != nil {
		return "", err
	}
	for _, resource := range resourceList.APIResources {
		if resource.Name == "pods/eviction" && resource.Kind == "Eviction" {
			return policyGroupVersion, nil
		}
	}
	return "", nil
}

func (d *NodeDrainer) deletePods(pods []v1.Pod) error {
	// TODO: make delete parallel
	var globalTimeout time.Duration
	if d.timeout == 0 {
		globalTimeout = time.Duration(math.MaxInt64)
	} else {
		globalTimeout = d.timeout
	}
	for _, pod := range pods {
		err := d.c.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: &d.gracePeriodSeconds,
		})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}
	_, err := d.waitUntilDeleted(pods, time.Second*1, globalTimeout)
	return err
}

func (d *NodeDrainer) evictPods(pods []v1.Pod, policyGroupVersion string) error {

	// TODO: review timeouts, grace period
	// TODO: let's use waitgroups instead?
	doneCh := make(chan bool, len(pods))
	errCh := make(chan error, 1)

	for _, pod := range pods {
		go func(pod v1.Pod, doneCh chan bool, errCh chan error) {
			var err error
			for {
				eviction := &policyv1.Eviction{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Eviction",
						APIVersion: policyGroupVersion,
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: pod.Namespace,
						Name:      pod.Name,
					},
					DeleteOptions: &metav1.DeleteOptions{
						GracePeriodSeconds: &d.gracePeriodSeconds,
					},
				}
				err = d.c.PolicyV1beta1().Evictions(eviction.Namespace).Evict(eviction)
				if err == nil {
					break
				} else if k8serrors.IsNotFound(err) {
					doneCh <- true
					return
				} else if k8serrors.IsTooManyRequests(err) {
					time.Sleep(5 * time.Second)
				} else {
					errCh <- fmt.Errorf("error when evicting pod %q: %v", pod.Name, err)
					return
				}
			}
			_, err = d.waitUntilDeleted([]v1.Pod{pod}, time.Second*1, time.Duration(math.MaxInt64))
			if err == nil {
				doneCh <- true
			} else {
				errCh <- fmt.Errorf("error when waiting for pod %q terminating: %v", pod.Name, err)
			}
		}(pod, doneCh, errCh)
	}

	doneCount := 0
	var globalTimeout time.Duration
	if d.timeout == 0 {
		globalTimeout = time.Duration(math.MaxInt64)
	} else {
		globalTimeout = d.timeout
	}
	for {
		select {
		case err := <-errCh:
			return err
		case <-doneCh:
			doneCount++
			if doneCount == len(pods) {
				return nil
			}
		case <-time.After(globalTimeout):
			return fmt.Errorf("drain did not complete within %v", globalTimeout)
		}
	}
}

func (d *NodeDrainer) waitUntilDeleted(pods []v1.Pod, interval, timeout time.Duration) ([]v1.Pod, error) {
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		pendingPods := 0
		for _, pod := range pods {
			p, err := d.c.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
			if k8serrors.IsNotFound(err) || (p != nil && p.ObjectMeta.UID != pod.ObjectMeta.UID) {
				log.Infof("pod %s.%s is deleted", pod.Namespace, pod.Name)
				continue
			} else if err != nil {
				return false, err
			} else {
				pendingPods++
			}
		}
		if pendingPods == 0 {
			return true, nil
		}
		return false, nil
	})
	return pods, err
}
