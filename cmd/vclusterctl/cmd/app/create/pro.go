package create

import (
	"context"
	"fmt"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v3/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/create"
	proclient "github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	"github.com/loft-sh/loftctl/v3/pkg/vcluster"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const LoftChartRepo = "https://charts.loft.sh"

func DeployProCluster(ctx context.Context, options *Options, proClient proclient.Client, virtualClusterName string, log log.Logger) error {
	// determine project & cluster name
	var err error
	options.Cluster, options.Project, err = helper.SelectProjectOrCluster(proClient, options.Cluster, options.Project, false, log)
	if err != nil {
		return err
	}

	virtualClusterNamespace := naming.ProjectNamespace(options.Project)
	managementClient, err := proClient.Management()
	if err != nil {
		return err
	}

	// make sure there is not existing virtual cluster
	var virtualClusterInstance *managementv1.VirtualClusterInstance
	virtualClusterInstance, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterNamespace).Get(ctx, virtualClusterName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("couldn't retrieve virtual cluster instance: %w", err)
	} else if err == nil && !virtualClusterInstance.DeletionTimestamp.IsZero() {
		log.Infof("Waiting until virtual cluster is deleted...")

		// wait until the virtual cluster instance is deleted
		waitErr := wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), false, func(ctx context.Context) (done bool, err error) {
			virtualClusterInstance, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterNamespace).Get(ctx, virtualClusterName, metav1.GetOptions{})
			if err != nil && !kerrors.IsNotFound(err) {
				return false, err
			} else if err == nil && virtualClusterInstance.DeletionTimestamp != nil {
				return false, nil
			}

			return true, nil
		})
		if waitErr != nil {
			return fmt.Errorf("get virtual cluster instance: %w", err)
		}

		virtualClusterInstance = nil
	} else if kerrors.IsNotFound(err) {
		virtualClusterInstance = nil
	}

	// if the virtual cluster already exists and flag is not set, we terminate
	if !options.Upgrade && virtualClusterInstance != nil {
		return fmt.Errorf("virtual cluster %s already exists in project %s", virtualClusterName, options.Project)
	}

	// create virtual cluster if necessary
	if virtualClusterInstance == nil {
		// create via template
		virtualClusterInstance, err = createWithTemplate(ctx, proClient, options, virtualClusterName, log)
		if err != nil {
			return err
		}
	} else if options.Upgrade {
		// upgrade via template
		virtualClusterInstance, err = upgradeWithTemplate(ctx, proClient, options, virtualClusterInstance, log)
		if err != nil {
			return err
		}
	}

	// wait until virtual cluster is ready
	virtualClusterInstance, err = vcluster.WaitForVirtualClusterInstance(ctx, managementClient, virtualClusterInstance.Namespace, virtualClusterInstance.Name, true, log)
	if err != nil {
		return err
	}
	log.Donef("Successfully created the virtual cluster %s in project %s", virtualClusterName, options.Project)

	return nil
}

func createWithTemplate(ctx context.Context, proClient proclient.Client, options *Options, virtualClusterName string, log log.Logger) (*managementv1.VirtualClusterInstance, error) {
	err := validateTemplateOptions(options)
	if err != nil {
		return nil, err
	}

	// resolve template
	virtualClusterTemplate, resolvedParameters, err := create.ResolveTemplate(
		proClient,
		options.Project,
		options.Template,
		options.TemplateVersion,
		options.SetParams,
		options.Params,
		log,
	)
	if err != nil {
		return nil, err
	}

	// create virtual cluster instance
	zone, offset := time.Now().Zone()
	virtualClusterInstance := &managementv1.VirtualClusterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: naming.ProjectNamespace(options.Project),
			Name:      virtualClusterName,
			Annotations: map[string]string{
				clusterv1.SleepModeTimezoneAnnotation: zone + "#" + strconv.Itoa(offset),
			},
		},
		Spec: managementv1.VirtualClusterInstanceSpec{
			VirtualClusterInstanceSpec: storagev1.VirtualClusterInstanceSpec{
				TemplateRef: &storagev1.TemplateRef{
					Name:    virtualClusterTemplate.Name,
					Version: options.TemplateVersion,
				},
				ClusterRef: storagev1.VirtualClusterClusterRef{
					ClusterRef: storagev1.ClusterRef{
						Cluster: options.Cluster,
					},
				},
				Parameters: resolvedParameters,
			},
		},
	}

	// set links
	create.SetCustomLinksAnnotation(virtualClusterInstance, options.Links)

	// get management client
	managementClient, err := proClient.Management()
	if err != nil {
		return nil, err
	}

	// create virtualclusterinstance
	log.Infof("Creating virtual cluster %s in project %s with template %s...", virtualClusterName, options.Project, virtualClusterTemplate.Name)
	virtualClusterInstance, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterInstance.Namespace).Create(ctx, virtualClusterInstance, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create virtual cluster: %w", err)
	}

	return virtualClusterInstance, nil
}

func upgradeWithTemplate(ctx context.Context, proClient proclient.Client, options *Options, virtualClusterInstance *managementv1.VirtualClusterInstance, log log.Logger) (*managementv1.VirtualClusterInstance, error) {
	err := validateTemplateOptions(options)
	if err != nil {
		return nil, err
	}

	// resolve template
	virtualClusterTemplate, resolvedParameters, err := create.ResolveTemplate(
		proClient,
		options.Project,
		options.Template,
		options.TemplateVersion,
		options.SetParams,
		options.Params,
		log,
	)
	if err != nil {
		return nil, err
	}

	// update virtual cluster instance
	if virtualClusterInstance.Spec.TemplateRef == nil {
		return nil, fmt.Errorf("virtual cluster instance doesn't use a template, cannot update virtual cluster")
	}

	oldVirtualCluster := virtualClusterInstance.DeepCopy()
	templateRefChanged := virtualClusterInstance.Spec.TemplateRef.Name != virtualClusterTemplate.Name
	paramsChanged := virtualClusterInstance.Spec.Parameters != resolvedParameters
	versionChanged := (options.TemplateVersion != "" && virtualClusterInstance.Spec.TemplateRef.Version != options.TemplateVersion)
	linksChanged := create.SetCustomLinksAnnotation(virtualClusterInstance, options.Links)

	// check if update is needed
	if templateRefChanged || paramsChanged || versionChanged || linksChanged {
		virtualClusterInstance.Spec.TemplateRef.Name = virtualClusterTemplate.Name
		virtualClusterInstance.Spec.TemplateRef.Version = options.TemplateVersion
		virtualClusterInstance.Spec.Parameters = resolvedParameters

		// get management client
		managementClient, err := proClient.Management()
		if err != nil {
			return nil, err
		}

		patch := client.MergeFrom(oldVirtualCluster)
		patchData, err := patch.Data(virtualClusterInstance)
		if err != nil {
			return nil, fmt.Errorf("calculate update patch: %w", err)
		}
		log.Infof("Updating virtual cluster %s in project %s...", virtualClusterInstance.Name, options.Project)
		virtualClusterInstance, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterInstance.Namespace).Patch(ctx, virtualClusterInstance.Name, patch.Type(), patchData, metav1.PatchOptions{})
		if err != nil {
			return nil, fmt.Errorf("patch virtual cluster: %w", err)
		}
	} else {
		log.Infof("Skip updating virtual cluster...")
	}

	return virtualClusterInstance, nil
}

func validateTemplateOptions(options *Options) error {
	if len(options.SetValues) > 0 {
		return fmt.Errorf("cannot use --set because the vcluster is using a template. Please use --set-param instead")
	}
	if len(options.Values) > 0 {
		return fmt.Errorf("cannot use --values because the vcluster is using a template. Please use --params instead")
	}
	if options.Isolate {
		return fmt.Errorf("cannot use --isolate because the vcluster is using a template")
	}
	if options.KubernetesVersion != "" {
		return fmt.Errorf("cannot use --kubernetes-version because the vcluster is using a template")
	}
	if options.Distro != "" {
		return fmt.Errorf("cannot use --distro because the vcluster is using a template")
	}
	if options.ChartName != "vcluster" {
		return fmt.Errorf("cannot use --chart-name because the vcluster is using a template")
	}
	if options.ChartRepo != LoftChartRepo {
		return fmt.Errorf("cannot use --chart-repo because the vcluster is using a template")
	}
	if options.ChartVersion != upgrade.GetVersion() {
		return fmt.Errorf("cannot use --chart-version because the vcluster is using a template")
	}

	return nil
}
