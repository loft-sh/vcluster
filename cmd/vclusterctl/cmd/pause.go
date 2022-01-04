package cmd

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	PausedAnnotation         = "loft.sh/paused"
	PausedReplicasAnnotation = "loft.sh/paused-replicas"
)

// PauseCmd holds the cmd flags
type PauseCmd struct {
	*flags.GlobalFlags
	Log log.Logger
}

// NewPauseCmd creates a new command
func NewPauseCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PauseCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "pause",
		Short: "Pauses a virtual cluster",
		Long: `
#######################################################
################### vcluster pause ####################
#######################################################
Pause will stop a virtual cluster and free all its used
computing resources.

Pause will scale down the virtual cluster and delete
all workloads created through the virtual cluster. Upon resume,
all workloads will be recreated. Other resources such 
as persistent volume claims, services etc. will not be affected.

Example:
vcluster pause test --namespace test
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
func (cmd *PauseCmd) Run(args []string) error {
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
	found, err := cmd.scaleDownStatefulSet(kubeClient, labelSelector)
	if err != nil {
		return err
	} else if !found {
		found, err = cmd.scaleDownDeployment(kubeClient, labelSelector)
		if err != nil {
			return err
		} else if !found {
			return errors.Errorf("couldn't find vcluster %s in namespace %s", args[0], cmd.Namespace)
		}

		// scale down kube api server
		_, err = cmd.scaleDownDeployment(kubeClient, "app=vcluster-api,release="+args[0])
		if err != nil {
			return err
		}

		// scale down kube controller
		_, err = cmd.scaleDownDeployment(kubeClient, "app=vcluster-controller,release="+args[0])
		if err != nil {
			return err
		}

		// scale down etcd
		_, err = cmd.scaleDownStatefulSet(kubeClient, "app=vcluster-etcd,release="+args[0])
		if err != nil {
			return err
		}
	}

	// delete vcluster workloads
	err = cmd.deleteVClusterWorkloads(kubeClient, "vcluster.loft.sh/managed-by="+args[0])
	if err != nil {
		return errors.Wrap(err, "delete vcluster workloads")
	}

	cmd.Log.Donef("Successfully paused vcluster %s/%s", cmd.Namespace, args[0])
	return nil
}

func (cmd *PauseCmd) deleteVClusterWorkloads(kubeClient kubernetes.Interface, labelSelector string) error {
	list, err := kubeClient.CoreV1().Pods(cmd.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return err
	}

	if len(list.Items) > 0 {
		cmd.Log.Infof("Delete %d vcluster workloads", len(list.Items))
		for _, item := range list.Items {
			err = kubeClient.CoreV1().Pods(cmd.Namespace).Delete(context.TODO(), item.Name, metav1.DeleteOptions{})
			if err != nil {
				return errors.Wrapf(err, "delete pod %s/%s", cmd.Namespace, item.Name)
			}
		}
	}

	return nil
}

func (cmd *PauseCmd) scaleDownDeployment(kubeClient kubernetes.Interface, labelSelector string) (bool, error) {
	list, err := kubeClient.AppsV1().Deployments(cmd.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, err
	} else if len(list.Items) == 0 {
		return false, nil
	}

	zero := int32(0)
	for _, item := range list.Items {
		if item.Annotations != nil && item.Annotations[PausedAnnotation] == "true" {
			cmd.Log.Infof("vcluster %s/%s is already paused", cmd.Namespace, item.Name)
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

		item.Annotations[PausedAnnotation] = "true"
		item.Annotations[PausedReplicasAnnotation] = strconv.Itoa(replicas)
		item.Spec.Replicas = &zero

		patch := client.MergeFrom(originalObject)
		data, err := patch.Data(&item)
		if err != nil {
			return false, errors.Wrap(err, "create deployment patch")
		}

		// patch deployment
		cmd.Log.Infof("Scale down deployment %s/%s...", cmd.Namespace, item.Name)
		_, err = kubeClient.AppsV1().Deployments(cmd.Namespace).Patch(context.TODO(), item.Name, patch.Type(), data, metav1.PatchOptions{})
		if err != nil {
			return false, errors.Wrap(err, "patch deployment")
		}

		// wait until deployment is scaled down
		err = wait.PollImmediate(time.Second, time.Minute*3, func() (done bool, err error) {
			deployment, err := kubeClient.AppsV1().Deployments(cmd.Namespace).Get(context.TODO(), item.Name, metav1.GetOptions{})
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

func (cmd *PauseCmd) scaleDownStatefulSet(kubeClient kubernetes.Interface, labelSelector string) (bool, error) {
	list, err := kubeClient.AppsV1().StatefulSets(cmd.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, err
	} else if len(list.Items) == 0 {
		return false, nil
	}

	zero := int32(0)
	for _, item := range list.Items {
		if item.Annotations != nil && item.Annotations[PausedAnnotation] == "true" {
			cmd.Log.Infof("vcluster %s/%s is already paused", cmd.Namespace, item.Name)
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

		item.Annotations[PausedAnnotation] = "true"
		item.Annotations[PausedReplicasAnnotation] = strconv.Itoa(replicas)
		item.Spec.Replicas = &zero

		patch := client.MergeFrom(originalObject)
		data, err := patch.Data(&item)
		if err != nil {
			return false, errors.Wrap(err, "create statefulSet patch")
		}

		// patch deployment
		cmd.Log.Infof("Scale down statefulSet %s/%s...", cmd.Namespace, item.Name)
		_, err = kubeClient.AppsV1().StatefulSets(cmd.Namespace).Patch(context.TODO(), item.Name, patch.Type(), data, metav1.PatchOptions{})
		if err != nil {
			return false, errors.Wrap(err, "patch statefulSet")
		}

		// wait until deployment is scaled down
		err = wait.PollImmediate(time.Second, time.Minute*3, func() (done bool, err error) {
			obj, err := kubeClient.AppsV1().StatefulSets(cmd.Namespace).Get(context.TODO(), item.Name, metav1.GetOptions{})
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
