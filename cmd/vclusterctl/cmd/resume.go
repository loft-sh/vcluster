package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

// ResumeCmd holds the cmd flags
type ResumeCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	kubeClient *kubernetes.Clientset
}

// NewResumeCmd creates a new command
func NewResumeCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ResumeCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "resume",
		Short: "Resumes a virtual cluster",
		Long: `
#######################################################
################### vcluster resume ###################
#######################################################
Resume will start a vcluster after it was paused. 
vcluster will recreate all the workloads after it has 
started automatically.

Example:
vcluster resume test --namespace test
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(args)
		},
	}
	return cobraCmd
}

// Run executes the functionality
func (cmd *ResumeCmd) Run(args []string) error {
	err := cmd.prepare(args[0])
	if err != nil {
		return err
	}

	err = resumeVCluster(cmd.kubeClient, args[0], cmd.Namespace, cmd.Log)
	if err != nil {
		return err
	}

	cmd.Log.Donef("Successfully resumed vcluster %s in namespace %s", args[0], cmd.Namespace)
	return nil
}

func (cmd *ResumeCmd) prepare(vClusterName string) error {
	vCluster, err := find.GetVCluster(cmd.Context, vClusterName, cmd.Namespace)
	if err != nil {
		return err
	}

	// load the rest config
	kubeConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%v), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	cmd.Namespace = vCluster.Namespace
	cmd.kubeClient = kubeClient
	return nil
}

func resumeVCluster(kubeClient *kubernetes.Clientset, name, namespace string, log log.Logger) error {
	// scale down vcluster itself
	labelSelector := "app=vcluster,release=" + name
	found, err := scaleUpStatefulSet(kubeClient, labelSelector, namespace, log)
	if err != nil {
		return err
	} else if !found {
		found, err = scaleUpDeployment(kubeClient, labelSelector, namespace, log)
		if err != nil {
			return err
		} else if !found {
			return errors.Errorf("couldn't find a paused vcluster %s in namespace %s. Make sure the vcluster exists and was paused previously", name, namespace)
		}

		// scale down kube api server
		_, err = scaleUpDeployment(kubeClient, "app=vcluster-api,release="+name, namespace, log)
		if err != nil {
			return err
		}

		// scale down kube controller
		_, err = scaleUpDeployment(kubeClient, "app=vcluster-controller,release="+name, namespace, log)
		if err != nil {
			return err
		}

		// scale down etcd
		_, err = scaleUpStatefulSet(kubeClient, "app=vcluster-etcd,release="+name, namespace, log)
		if err != nil {
			return err
		}
	}

	return nil
}

func scaleUpDeployment(kubeClient kubernetes.Interface, labelSelector string, namespace string, log log.Logger) (bool, error) {
	list, err := kubeClient.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, err
	} else if len(list.Items) == 0 {
		return false, nil
	}

	for _, item := range list.Items {
		if item.Annotations == nil || item.Annotations[constants.PausedAnnotation] != "true" {
			return false, nil
		}

		originalObject := item.DeepCopy()

		replicas := 1
		if item.Annotations[PausedReplicasAnnotation] != "" {
			replicas, err = strconv.Atoi(item.Annotations[PausedReplicasAnnotation])
			if err != nil {
				log.Warnf("error parsing old replicas: %v", err)
				replicas = 1
			}
		}

		replicas32 := int32(replicas)
		delete(item.Annotations, constants.PausedAnnotation)
		delete(item.Annotations, PausedReplicasAnnotation)
		item.Spec.Replicas = &replicas32

		patch := client.MergeFrom(originalObject)
		data, err := patch.Data(&item)
		if err != nil {
			return false, errors.Wrap(err, "create deployment patch")
		}

		// patch deployment
		_, err = kubeClient.AppsV1().Deployments(namespace).Patch(context.TODO(), item.Name, patch.Type(), data, metav1.PatchOptions{})
		if err != nil {
			return false, errors.Wrap(err, "patch deployment")
		}
	}

	return true, nil
}

func scaleUpStatefulSet(kubeClient kubernetes.Interface, labelSelector string, namespace string, log log.Logger) (bool, error) {
	list, err := kubeClient.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, err
	} else if len(list.Items) == 0 {
		return false, nil
	}

	for _, item := range list.Items {
		if item.Annotations == nil || item.Annotations[constants.PausedAnnotation] != "true" {
			return false, nil
		}

		originalObject := item.DeepCopy()

		replicas := 1
		if item.Annotations[PausedReplicasAnnotation] != "" {
			replicas, err = strconv.Atoi(item.Annotations[PausedReplicasAnnotation])
			if err != nil {
				log.Warnf("error parsing old replicas: %v", err)
				replicas = 1
			}
		}

		replicas32 := int32(replicas)
		delete(item.Annotations, constants.PausedAnnotation)
		delete(item.Annotations, PausedReplicasAnnotation)
		item.Spec.Replicas = &replicas32

		patch := client.MergeFrom(originalObject)
		data, err := patch.Data(&item)
		if err != nil {
			return false, errors.Wrap(err, "create statefulSet patch")
		}

		// patch deployment
		_, err = kubeClient.AppsV1().StatefulSets(namespace).Patch(context.TODO(), item.Name, patch.Type(), data, metav1.PatchOptions{})
		if err != nil {
			return false, errors.Wrap(err, "patch statefulSet")
		}
	}

	return true, nil
}
