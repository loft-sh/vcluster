package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

// ResumeCmd holds the cmd flags
type ResumeCmd struct {
	*flags.GlobalFlags
	Log log.Logger
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
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	})

	// load the rest config
	kubeConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%v), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	if cmd.Namespace == "" {
		cmd.Namespace, _, err = kubeClientConfig.Namespace()
		if err != nil {
			return err
		} else if cmd.Namespace == "" {
			cmd.Namespace = "default"
		}
	}

	// scale down vcluster itself
	labelSelector := "app=vcluster,release=" + args[0]
	found, err := cmd.scaleUpStatefulSet(kubeClient, labelSelector)
	if err != nil {
		return err
	} else if !found {
		found, err = cmd.scaleUpDeployment(kubeClient, labelSelector)
		if err != nil {
			return err
		} else if !found {
			return errors.Errorf("couldn't find a paused vcluster %s in namespace %s. Make sure the vcluster exists and was paused previously", args[0], cmd.Namespace)
		}

		// scale down kube api server
		_, err = cmd.scaleUpDeployment(kubeClient, "app=vcluster-api,release="+args[0])
		if err != nil {
			return err
		}

		// scale down kube controller
		_, err = cmd.scaleUpDeployment(kubeClient, "app=vcluster-controller,release="+args[0])
		if err != nil {
			return err
		}

		// scale down etcd
		_, err = cmd.scaleUpStatefulSet(kubeClient, "app=vcluster-etcd,release="+args[0])
		if err != nil {
			return err
		}
	}

	cmd.Log.Donef("Successfully resumed vcluster %s in namespace %s", args[0], cmd.Namespace)
	return nil
}

func (cmd *ResumeCmd) scaleUpDeployment(kubeClient kubernetes.Interface, labelSelector string) (bool, error) {
	list, err := kubeClient.AppsV1().Deployments(cmd.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, err
	} else if len(list.Items) == 0 {
		return false, nil
	}

	for _, item := range list.Items {
		if item.Annotations == nil || item.Annotations[PausedAnnotation] != "true" {
			return false, nil
		}

		originalObject := item.DeepCopy()

		replicas := 1
		if item.Annotations[PausedReplicasAnnotation] != "" {
			replicas, err = strconv.Atoi(item.Annotations[PausedReplicasAnnotation])
			if err != nil {
				cmd.Log.Warnf("error parsing old replicas: %v", err)
				replicas = 1
			}
		}

		replicas32 := int32(replicas)
		delete(item.Annotations, PausedAnnotation)
		delete(item.Annotations, PausedReplicasAnnotation)
		item.Spec.Replicas = &replicas32

		patch := client.MergeFrom(originalObject)
		data, err := patch.Data(&item)
		if err != nil {
			return false, errors.Wrap(err, "create deployment patch")
		}

		// patch deployment
		_, err = kubeClient.AppsV1().Deployments(cmd.Namespace).Patch(context.TODO(), item.Name, patch.Type(), data, metav1.PatchOptions{})
		if err != nil {
			return false, errors.Wrap(err, "patch deployment")
		}
	}

	return true, nil
}

func (cmd *ResumeCmd) scaleUpStatefulSet(kubeClient kubernetes.Interface, labelSelector string) (bool, error) {
	list, err := kubeClient.AppsV1().StatefulSets(cmd.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, err
	} else if len(list.Items) == 0 {
		return false, nil
	}

	for _, item := range list.Items {
		if item.Annotations == nil || item.Annotations[PausedAnnotation] != "true" {
			return false, nil
		}

		originalObject := item.DeepCopy()

		replicas := 1
		if item.Annotations[PausedReplicasAnnotation] != "" {
			replicas, err = strconv.Atoi(item.Annotations[PausedReplicasAnnotation])
			if err != nil {
				cmd.Log.Warnf("error parsing old replicas: %v", err)
				replicas = 1
			}
		}

		replicas32 := int32(replicas)
		delete(item.Annotations, PausedAnnotation)
		delete(item.Annotations, PausedReplicasAnnotation)
		item.Spec.Replicas = &replicas32

		patch := client.MergeFrom(originalObject)
		data, err := patch.Data(&item)
		if err != nil {
			return false, errors.Wrap(err, "create statefulSet patch")
		}

		// patch deployment
		_, err = kubeClient.AppsV1().StatefulSets(cmd.Namespace).Patch(context.TODO(), item.Name, patch.Type(), data, metav1.PatchOptions{})
		if err != nil {
			return false, errors.Wrap(err, "patch statefulSet")
		}
	}

	return true, nil
}
