package lifecycle

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/kube"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PauseVCluster pauses a running vcluster
func PauseVCluster(ctx context.Context, kubeClient *kubernetes.Clientset, name, namespace string, log log.BaseLogger) error {
	// scale down vcluster itself
	labelSelector := "app=vcluster,release=" + name
	found, err := scaleDownStatefulSet(ctx, kubeClient, labelSelector, namespace, log)
	if err != nil {
		return err
	} else if !found {
		found, err = scaleDownDeployment(ctx, kubeClient, labelSelector, namespace, log)
		if err != nil {
			return err
		} else if !found {
			return errors.Errorf("couldn't find vcluster %s in namespace %s", name, namespace)
		}

		// scale down kube api server
		_, err = scaleDownDeployment(ctx, kubeClient, "app=vcluster-api,release="+name, namespace, log)
		if err != nil {
			return err
		}

		// scale down kube controller
		_, err = scaleDownDeployment(ctx, kubeClient, "app=vcluster-controller,release="+name, namespace, log)
		if err != nil {
			return err
		}

		// scale down etcd
		_, err = scaleDownStatefulSet(ctx, kubeClient, "app=vcluster-etcd,release="+name, namespace, log)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeletePods deletes all pods associated with a running vcluster
func DeletePods(ctx context.Context, kubeClient *kubernetes.Clientset, labelSelector, namespace string, log log.BaseLogger) error {
	list, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return err
	}

	if len(list.Items) > 0 {
		log.Infof("Relaunching %d vcluster pods", len(list.Items))
		for _, item := range list.Items {
			err = kubeClient.CoreV1().Pods(namespace).Delete(ctx, item.Name, metav1.DeleteOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					continue
				}
				return errors.Wrapf(err, "delete pod %s/%s", namespace, item.Name)
			}
		}
	}

	return nil
}

func DeleteMultiNamespaceVClusterWorkloads(ctx context.Context, client *kubernetes.Clientset, vclusterName, vclusterNamespace string, _ log.BaseLogger) error {
	// get all host namespaces managed by this multinamespace mode enabled vcluster
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(map[string]string{
			translate.MarkerLabel: translate.SafeConcatName(vclusterNamespace, "x", vclusterName),
		}),
	})
	if err != nil && !kerrors.IsForbidden(err) {
		return fmt.Errorf("list namespaces: %w", err)
	}
	if namespaces == nil {
		return errors.New("list namespaces: nil result")
	}

	// delete all pods inside the above returned namespaces
	for _, ns := range namespaces.Items {
		podList, podListErr := client.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
		if podListErr != nil {
			return errors.Wrapf(err, "error listing pods in namespace %s", ns.Name)
		}

		for _, pod := range podList.Items {
			err := client.CoreV1().Pods(ns.Name).Delete(ctx, pod.Name, metav1.DeleteOptions{})
			if err != nil {
				return errors.Wrapf(err, "error deleting pod %s/%s", ns.Name, pod.Name)
			}
		}
	}

	return nil
}

func scaleDownDeployment(ctx context.Context, kubeClient kubernetes.Interface, labelSelector, namespace string, log log.BaseLogger) (bool, error) {
	list, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, err
	} else if len(list.Items) == 0 {
		return false, nil
	}

	zero := int32(0)
	for _, item := range list.Items {
		if IsPaused(&item) {
			log.Infof("vcluster %s/%s is already paused", namespace, item.Name)
			return true, nil
		} else if item.Spec.Replicas != nil && *item.Spec.Replicas == 0 {
			continue
		}

		originalObject := item.DeepCopy()
		if item.Annotations == nil {
			item.Annotations = map[string]string{}
		}

		replicas := 1
		if item.Spec.Replicas != nil {
			replicas = int(*item.Spec.Replicas)
		}

		item.Annotations[constants.PausedAnnotation] = "true"
		item.Annotations[constants.PausedReplicasAnnotation] = strconv.Itoa(replicas)
		item.Annotations[constants.PausedDateAnnotation] = time.Now().Format("2006-01-02T15:04:05.000Z")
		item.Spec.Replicas = &zero

		patch := client.MergeFrom(originalObject)
		data, err := patch.Data(&item)
		if err != nil {
			return false, errors.Wrap(err, "create deployment patch")
		}

		// patch deployment
		log.Infof("Scale down deployment %s/%s...", namespace, item.Name)
		_, err = kubeClient.AppsV1().Deployments(namespace).Patch(ctx, item.Name, patch.Type(), data, metav1.PatchOptions{})
		if err != nil {
			return false, errors.Wrap(err, "patch deployment")
		}

		// wait until deployment is scaled down
		err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*3, true, func(ctx context.Context) (done bool, err error) {
			deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, item.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			return deployment.Status.Replicas == 0, nil
		})
		if err != nil {
			return false, errors.Wrap(err, "wait for deployment scaled down")
		}
	}

	return true, nil
}

func scaleDownStatefulSet(ctx context.Context, kubeClient kubernetes.Interface, labelSelector, namespace string, log log.BaseLogger) (bool, error) {
	list, err := kubeClient.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, err
	} else if len(list.Items) == 0 {
		return false, nil
	}

	zero := int32(0)
	for _, item := range list.Items {
		if IsPaused(&item) {
			log.Infof("vcluster %s/%s is already paused", namespace, item.Name)
			return true, nil
		} else if item.Spec.Replicas != nil && *item.Spec.Replicas == 0 {
			continue
		}

		originalObject := item.DeepCopy()
		if item.Annotations == nil {
			item.Annotations = map[string]string{}
		}

		replicas := 1
		if item.Spec.Replicas != nil {
			replicas = int(*item.Spec.Replicas)
		}

		item.Annotations[constants.PausedAnnotation] = "true"
		item.Annotations[constants.PausedReplicasAnnotation] = strconv.Itoa(replicas)
		item.Annotations[constants.PausedDateAnnotation] = time.Now().Format("2006-01-02T15:04:05.000Z")
		item.Spec.Replicas = &zero

		patch := client.MergeFrom(originalObject)
		data, err := patch.Data(&item)
		if err != nil {
			return false, errors.Wrap(err, "create statefulSet patch")
		}

		// patch deployment
		log.Infof("Scale down statefulSet %s/%s...", namespace, item.Name)
		_, err = kubeClient.AppsV1().StatefulSets(namespace).Patch(ctx, item.Name, patch.Type(), data, metav1.PatchOptions{})
		if err != nil {
			return false, errors.Wrap(err, "patch statefulSet")
		}

		// wait until deployment is scaled down
		err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*3, true, func(ctx context.Context) (done bool, err error) {
			obj, err := kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, item.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			return obj.Status.Replicas == 0, nil
		})
		if err != nil {
			return false, errors.Wrap(err, "wait for statefulSet scaled down")
		}
	}

	return true, nil
}

// ResumeVCluster resumes a paused vcluster
func ResumeVCluster(ctx context.Context, kubeClient *kubernetes.Clientset, name, namespace string, log log.BaseLogger) error {
	// scale up vcluster itself
	labelSelector := "app=vcluster,release=" + name
	found, err := scaleUpStatefulSet(ctx, kubeClient, labelSelector, namespace, log)
	if err != nil {
		return err
	} else if !found {
		found, err = scaleUpDeployment(ctx, kubeClient, labelSelector, namespace, log)
		if err != nil {
			return err
		} else if !found {
			return errors.Errorf("couldn't find a paused vcluster %s in namespace %s. Make sure the vcluster exists and was paused previously", name, namespace)
		}

		// scale up kube api server
		_, err = scaleUpDeployment(ctx, kubeClient, "app=vcluster-api,release="+name, namespace, log)
		if err != nil {
			return err
		}

		// scale up kube controller
		_, err = scaleUpDeployment(ctx, kubeClient, "app=vcluster-controller,release="+name, namespace, log)
		if err != nil {
			return err
		}

		// scale up etcd
		_, err = scaleUpStatefulSet(ctx, kubeClient, "app=vcluster-etcd,release="+name, namespace, log)
		if err != nil {
			return err
		}
	}

	return nil
}

func scaleUpDeployment(ctx context.Context, kubeClient kubernetes.Interface, labelSelector string, namespace string, log log.BaseLogger) (bool, error) {
	list, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, err
	} else if len(list.Items) == 0 {
		return false, nil
	}

	for _, item := range list.Items {
		if !IsPaused(&item) {
			return false, nil
		}

		originalObject := item.DeepCopy()

		replicas := 1
		if item.Annotations[constants.PausedReplicasAnnotation] != "" {
			replicas, err = strconv.Atoi(item.Annotations[constants.PausedReplicasAnnotation])
			if err != nil {
				log.Errorf("error parsing old replicas: %v", err)
				replicas = 1
			}
		}

		replicas32 := int32(replicas)
		delete(item.Annotations, constants.PausedAnnotation)
		delete(item.Annotations, constants.PausedReplicasAnnotation)
		delete(item.Annotations, constants.PausedDateAnnotation)
		item.Spec.Replicas = &replicas32

		patch := client.MergeFrom(originalObject)
		data, err := patch.Data(&item)
		if err != nil {
			return false, errors.Wrap(err, "create deployment patch")
		}

		// patch deployment
		_, err = kubeClient.AppsV1().Deployments(namespace).Patch(ctx, item.Name, patch.Type(), data, metav1.PatchOptions{})
		if err != nil {
			return false, errors.Wrap(err, "patch deployment")
		}
	}

	return true, nil
}

func scaleUpStatefulSet(ctx context.Context, kubeClient kubernetes.Interface, labelSelector string, namespace string, log log.BaseLogger) (bool, error) {
	list, err := kubeClient.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, err
	} else if len(list.Items) == 0 {
		return false, nil
	}

	for _, item := range list.Items {
		if !IsPaused(&item) {
			return false, nil
		}

		originalObject := item.DeepCopy()

		replicas := 1
		if item.Annotations[constants.PausedReplicasAnnotation] != "" {
			replicas, err = strconv.Atoi(item.Annotations[constants.PausedReplicasAnnotation])
			if err != nil {
				log.Errorf("error parsing old replicas: %v", err)
				replicas = 1
			}
		}

		replicas32 := int32(replicas)
		delete(item.Annotations, constants.PausedAnnotation)
		delete(item.Annotations, constants.PausedReplicasAnnotation)
		delete(item.Annotations, constants.PausedDateAnnotation)
		item.Spec.Replicas = &replicas32

		patch := client.MergeFrom(originalObject)
		data, err := patch.Data(&item)
		if err != nil {
			return false, errors.Wrap(err, "create statefulSet patch")
		}

		// patch deployment
		_, err = kubeClient.AppsV1().StatefulSets(namespace).Patch(ctx, item.Name, patch.Type(), data, metav1.PatchOptions{})
		if err != nil {
			return false, errors.Wrap(err, "patch statefulSet")
		}
	}

	return true, nil
}

func IsPaused(annotated kube.Annotated) bool {
	return annotated != nil && annotated.GetAnnotations()[constants.PausedAnnotation] == "true"
}
