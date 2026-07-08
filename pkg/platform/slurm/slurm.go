// Package slurm holds shared client-side helpers for the `vcluster platform`
// slurm subcommands (connect, list, get, create, delete).
package slurm

import (
	"context"
	"fmt"
	"time"

	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ConditionSlurmTailnetReady is set on a SlurmInstance once the login access key
// is minted and the login proxy can join the tailnet, i.e. once SSH access is
// possible. The platform controller (loft-enterprise) owns setting this
// condition.
const ConditionSlurmTailnetReady = storagev1.SlurmInstanceSlurmTailnetReady

// Get fetches a single SlurmInstance in the given project.
func Get(ctx context.Context, managementClient kube.Interface, project, name string) (*managementv1.SlurmInstance, error) {
	return managementClient.Loft().ManagementV1().SlurmInstances(projectutil.ProjectNamespace(project)).Get(ctx, name, metav1.GetOptions{})
}

// List returns all SlurmInstances in the given project.
func List(ctx context.Context, managementClient kube.Interface, project string) (*managementv1.SlurmInstanceList, error) {
	return managementClient.Loft().ManagementV1().SlurmInstances(projectutil.ProjectNamespace(project)).List(ctx, metav1.ListOptions{})
}

// IsTailnetReady reports whether the SlurmTailnetReady condition is True.
func IsTailnetReady(instance *managementv1.SlurmInstance) bool {
	return conditionStatus(instance, ConditionSlurmTailnetReady) == corev1.ConditionTrue
}

// WaitForTailnetReady polls the SlurmInstance until its SlurmTailnetReady
// condition is True. It fails fast if the instance enters the Failed phase.
// When wait is false it returns the current instance immediately.
func WaitForTailnetReady(ctx context.Context, managementClient kube.Interface, project, name string, wait bool, log log.Logger) (*managementv1.SlurmInstance, error) {
	instance, err := Get(ctx, managementClient, project, name)
	if err != nil {
		return nil, err
	}
	if !wait || IsTailnetReady(instance) {
		return instance, nil
	}

	log.Infof("Waiting for Slurm instance %s to be ready for SSH access...", name)
	return pollUntilReady(ctx, managementClient, project, name)
}

func pollUntilReady(ctx context.Context, managementClient kube.Interface, project, name string) (*managementv1.SlurmInstance, error) {
	var instance *managementv1.SlurmInstance
	err := wait.PollUntilContextTimeout(ctx, time.Second, clihelper.Timeout(), true, func(ctx context.Context) (bool, error) {
		var err error
		instance, err = Get(ctx, managementClient, project, name)
		if err != nil {
			return false, err
		}
		if instance.Status.Phase == storagev1.SlurmInstancePhaseFailed {
			return false, fmt.Errorf("slurm instance %s failed: %s", name, instanceMessage(instance))
		}
		return IsTailnetReady(instance), nil
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func conditionStatus(instance *managementv1.SlurmInstance, conditionType agentstoragev1.ConditionType) corev1.ConditionStatus {
	for _, c := range instance.Status.Conditions {
		if c.Type == conditionType {
			return c.Status
		}
	}
	return corev1.ConditionUnknown
}

func instanceMessage(instance *managementv1.SlurmInstance) string {
	if instance.Status.Message != "" {
		return instance.Status.Message
	}
	if instance.Status.Reason != "" {
		return instance.Status.Reason
	}
	return string(instance.Status.Phase)
}
