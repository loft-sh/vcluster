package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/cmd/create"
	"github.com/loft-sh/loftctl/v4/pkg/client/helper"
	"github.com/loft-sh/loftctl/v4/pkg/client/naming"
	"github.com/loft-sh/loftctl/v4/pkg/config"
	"github.com/loft-sh/loftctl/v4/pkg/vcluster"
	"github.com/loft-sh/log"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/strvals"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	"golang.org/x/mod/semver"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreatePlatform(ctx context.Context, options *CreateOptions, globalFlags *flags.GlobalFlags, virtualClusterName string, log log.Logger) error {
	platformClient, err := platform.CreatePlatformClient()
	if err != nil {
		return err
	}

	// determine project & cluster name
	options.Cluster, options.Project, err = helper.SelectProjectOrCluster(ctx, platformClient, options.Cluster, options.Project, false, log)
	if err != nil {
		return err
	}

	virtualClusterNamespace := naming.ProjectNamespace(options.Project)
	managementClient, err := platformClient.Management()
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
	} else if virtualClusterInstance != nil && virtualClusterInstance.Spec.NetworkPeer {
		return fmt.Errorf("cannot upgrade a virtual cluster that was created via helm, please run 'vcluster use manager helm' or use the '--manager helm' flag")
	}

	// should create via template
	useTemplate, err := shouldCreateWithTemplate(ctx, platformClient, options, virtualClusterInstance)
	if err != nil {
		return fmt.Errorf("should use template: %w", err)
	}

	// create virtual cluster if necessary
	if useTemplate {
		if virtualClusterInstance == nil {
			// create via template
			virtualClusterInstance, err = createWithTemplate(ctx, platformClient, options, virtualClusterName, log)
			if err != nil {
				return err
			}
		} else {
			// upgrade via template
			virtualClusterInstance, err = upgradeWithTemplate(ctx, platformClient, options, virtualClusterInstance, log)
			if err != nil {
				return err
			}
		}
	} else {
		if virtualClusterInstance == nil {
			// create without template
			virtualClusterInstance, err = createWithoutTemplate(ctx, platformClient, options, virtualClusterName, globalFlags.Namespace, log)
			if err != nil {
				return err
			}
		} else {
			// upgrade via template
			virtualClusterInstance, err = upgradeWithoutTemplate(ctx, platformClient, options, virtualClusterInstance, log)
			if err != nil {
				return err
			}
		}
	}

	// wait until virtual cluster is ready
	virtualClusterInstance, err = vcluster.WaitForVirtualClusterInstance(ctx, managementClient, virtualClusterInstance.Namespace, virtualClusterInstance.Name, true, log)
	if err != nil {
		return err
	}
	log.Donef("Successfully created the virtual cluster %s in project %s", virtualClusterName, options.Project)

	// check if we should connect to the vcluster
	if options.Connect {
		return ConnectPlatform(ctx, &ConnectOptions{
			UpdateCurrent:         options.UpdateCurrent,
			KubeConfigContextName: options.KubeConfigContextName,
			KubeConfig:            "./kubeconfig.yaml",
			Project:               options.Project,
		}, globalFlags, virtualClusterName, nil, log)
	}

	log.Donef("Successfully created virtual cluster %s in project %s. \n- Use 'vcluster connect %s --project %s' to access the virtual cluster\n- Use `vcluster connect %s --project %s -- kubectl get ns` to run a command directly within the vcluster", virtualClusterName, options.Project, virtualClusterName, options.Project, virtualClusterName, options.Project)
	return nil
}

func createWithoutTemplate(ctx context.Context, platformClient platform.Client, options *CreateOptions, virtualClusterName, targetNamespace string, log log.Logger) (*managementv1.VirtualClusterInstance, error) {
	err := validateNoTemplateOptions(options)
	if err != nil {
		return nil, err
	}

	// merge values
	helmValues, err := mergeValues(platformClient, options, log)
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
				Template: &storagev1.VirtualClusterTemplateDefinition{
					VirtualClusterCommonSpec: agentstoragev1.VirtualClusterCommonSpec{
						HelmRelease: agentstoragev1.VirtualClusterHelmRelease{
							Chart: agentstoragev1.VirtualClusterHelmChart{
								Name:    options.ChartName,
								Repo:    options.ChartRepo,
								Version: options.ChartVersion,
							},
							Values: helmValues,
						},
						ForwardToken: true,
						Pro: agentstoragev1.VirtualClusterProSpec{
							Enabled: true,
						},
					},
				},
				ClusterRef: storagev1.VirtualClusterClusterRef{
					ClusterRef: storagev1.ClusterRef{
						Cluster:   options.Cluster,
						Namespace: targetNamespace,
					},
				},
			},
		},
	}

	// set links
	create.SetCustomLinksAnnotation(virtualClusterInstance, options.Links)

	// set labels
	_, err = create.UpdateLabels(virtualClusterInstance, options.Labels)
	if err != nil {
		return nil, err
	}

	// set annotations
	_, err = create.UpdateAnnotations(virtualClusterInstance, options.Annotations)
	if err != nil {
		return nil, err
	}

	// get management client
	managementClient, err := platformClient.Management()
	if err != nil {
		return nil, err
	}

	// create virtualclusterinstance
	log.Infof("Creating virtual cluster %s in project %s...", virtualClusterName, options.Project)
	virtualClusterInstance, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterInstance.Namespace).Create(ctx, virtualClusterInstance, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create virtual cluster: %w", err)
	}

	return virtualClusterInstance, nil
}

func upgradeWithoutTemplate(ctx context.Context, platformClient platform.Client, options *CreateOptions, virtualClusterInstance *managementv1.VirtualClusterInstance, log log.Logger) (*managementv1.VirtualClusterInstance, error) {
	err := validateNoTemplateOptions(options)
	if err != nil {
		return nil, err
	}

	// merge values
	helmValues, err := mergeValues(platformClient, options, log)
	if err != nil {
		return nil, err
	}

	// update virtual cluster instance
	if virtualClusterInstance.Spec.Template == nil {
		return nil, fmt.Errorf("virtual cluster instance uses a template, cannot update virtual cluster")
	}

	oldVirtualCluster := virtualClusterInstance.DeepCopy()
	chartNameChanged := virtualClusterInstance.Spec.Template.HelmRelease.Chart.Name != options.ChartName
	if chartNameChanged {
		return nil, fmt.Errorf("cannot change chart name from '%s' to '%s', this operation is not allowed", virtualClusterInstance.Spec.Template.HelmRelease.Chart.Name, options.ChartName)
	}

	chartRepoChanged := virtualClusterInstance.Spec.Template.HelmRelease.Chart.Repo != options.ChartRepo
	chartVersionChanged := virtualClusterInstance.Spec.Template.HelmRelease.Chart.Version != options.ChartVersion
	valuesChanged := virtualClusterInstance.Spec.Template.HelmRelease.Values != helmValues

	// set links
	linksChanged := create.SetCustomLinksAnnotation(virtualClusterInstance, options.Links)

	// set labels
	labelsChanged, err := create.UpdateLabels(virtualClusterInstance, options.Labels)
	if err != nil {
		return nil, err
	}

	// set annotations
	annotationsChanged, err := create.UpdateAnnotations(virtualClusterInstance, options.Annotations)
	if err != nil {
		return nil, err
	}

	// check if update is needed
	if chartRepoChanged || chartVersionChanged || valuesChanged || linksChanged || labelsChanged || annotationsChanged {
		virtualClusterInstance.Spec.Template.HelmRelease.Chart.Repo = options.ChartRepo
		virtualClusterInstance.Spec.Template.HelmRelease.Chart.Version = options.ChartVersion
		virtualClusterInstance.Spec.Template.HelmRelease.Values = helmValues

		// get management client
		managementClient, err := platformClient.Management()
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

func shouldCreateWithTemplate(ctx context.Context, platformClient platform.Client, options *CreateOptions, virtualClusterInstance *managementv1.VirtualClusterInstance) (bool, error) {
	virtualClusterInstanceHasTemplate := virtualClusterInstance != nil && virtualClusterInstance.Spec.TemplateRef != nil
	virtualClusterInstanceHasNoTemplate := virtualClusterInstance != nil && virtualClusterInstance.Spec.TemplateRef == nil
	if virtualClusterInstanceHasTemplate || options.Template != "" {
		return true, nil
	} else if virtualClusterInstanceHasNoTemplate {
		return false, nil
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return false, err
	}

	project, err := managementClient.Loft().ManagementV1().Projects().Get(ctx, options.Project, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("get vCluster project: %w", err)
	}

	// check if there is a default template
	for _, template := range project.Spec.AllowedTemplates {
		if template.Kind == "VirtualClusterTemplate" && template.IsDefault {
			return true, nil
		}
	}

	// check if we can create without
	if project.Spec.RequireTemplate.Disabled {
		return false, nil
	}

	return true, nil
}

func createWithTemplate(ctx context.Context, platformClient platform.Client, options *CreateOptions, virtualClusterName string, log log.Logger) (*managementv1.VirtualClusterInstance, error) {
	err := validateTemplateOptions(options)
	if err != nil {
		return nil, err
	}

	// resolve template
	virtualClusterTemplate, resolvedParameters, err := create.ResolveTemplate(
		ctx,
		platformClient,
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

	// set labels
	_, err = create.UpdateLabels(virtualClusterInstance, options.Labels)
	if err != nil {
		return nil, err
	}

	// set annotations
	_, err = create.UpdateAnnotations(virtualClusterInstance, options.Annotations)
	if err != nil {
		return nil, err
	}

	// get management client
	managementClient, err := platformClient.Management()
	if err != nil {
		return nil, err
	}

	// create virtual cluster instance
	log.Infof("Creating virtual cluster %s in project %s with template %s...", virtualClusterName, options.Project, virtualClusterTemplate.Name)
	virtualClusterInstance, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterInstance.Namespace).Create(ctx, virtualClusterInstance, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create virtual cluster: %w", err)
	}

	return virtualClusterInstance, nil
}

func upgradeWithTemplate(ctx context.Context, platformClient platform.Client, options *CreateOptions, virtualClusterInstance *managementv1.VirtualClusterInstance, log log.Logger) (*managementv1.VirtualClusterInstance, error) {
	err := validateTemplateOptions(options)
	if err != nil {
		return nil, err
	}

	// resolve template
	virtualClusterTemplate, resolvedParameters, err := create.ResolveTemplate(
		ctx,
		platformClient,
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

	// set labels
	labelsChanged, err := create.UpdateLabels(virtualClusterInstance, options.Labels)
	if err != nil {
		return nil, err
	}

	// set annotations
	annotationsChanged, err := create.UpdateAnnotations(virtualClusterInstance, options.Annotations)
	if err != nil {
		return nil, err
	}

	// check if update is needed
	if templateRefChanged || paramsChanged || versionChanged || linksChanged || labelsChanged || annotationsChanged {
		virtualClusterInstance.Spec.TemplateRef.Name = virtualClusterTemplate.Name
		virtualClusterInstance.Spec.TemplateRef.Version = options.TemplateVersion
		virtualClusterInstance.Spec.Parameters = resolvedParameters

		// get management client
		managementClient, err := platformClient.Management()
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

func validateNoTemplateOptions(options *CreateOptions) error {
	if len(options.SetParams) > 0 {
		return fmt.Errorf("cannot use --set-param because the vcluster is not using a template. Use --set instead")
	}
	if options.Params != "" {
		return fmt.Errorf("cannot use --params because the vcluster is not using a template. Use --values instead")
	}
	if options.Template != "" {
		return fmt.Errorf("cannot use --template because the vcluster is not using a template")
	}
	if options.TemplateVersion != "" {
		return fmt.Errorf("cannot use --template-version because the vcluster is not using a template")
	}

	return nil
}

func validateTemplateOptions(options *CreateOptions) error {
	if len(options.SetValues) > 0 {
		return fmt.Errorf("cannot use --set because the vcluster is using a template. Please use --set-param instead")
	}
	if len(options.Values) > 0 {
		return fmt.Errorf("cannot use --values because the vcluster is using a template. Please use --params instead")
	}
	if options.KubernetesVersion != "" {
		return fmt.Errorf("cannot use --kubernetes-version because the vcluster is using a template")
	}
	if options.Distro != "" && options.Distro != "k3s" {
		return fmt.Errorf("cannot use --distro because the vcluster is using a template")
	}
	if options.ChartName != "vcluster" {
		return fmt.Errorf("cannot use --chart-name because the vcluster is using a template")
	}
	if options.ChartRepo != constants.LoftChartRepo {
		return fmt.Errorf("cannot use --chart-repo because the vcluster is using a template")
	}
	if options.ChartVersion != upgrade.GetVersion() {
		return fmt.Errorf("cannot use --chart-version because the vcluster is using a template")
	}

	return nil
}

func mergeValues(platformClient platform.Client, options *CreateOptions, log log.Logger) (string, error) {
	// merge values
	chartOptions, err := toChartOptions(platformClient, options, log)
	if err != nil {
		return "", err
	}
	chartValues, err := vclusterconfig.GetExtraValues(chartOptions)
	if err != nil {
		return "", err
	}

	// parse into map
	outValues, err := parseString(chartValues)
	if err != nil {
		return "", err
	}

	// merge values
	for _, valuesFile := range options.Values {
		out, err := os.ReadFile(valuesFile)
		if err != nil {
			return "", fmt.Errorf("reading values file %s: %w", valuesFile, err)
		}

		extraValues, err := parseString(string(out))
		if err != nil {
			return "", fmt.Errorf("parse values file %s: %w", valuesFile, err)
		}

		strvals.MergeMaps(outValues, extraValues)
	}

	// merge set
	for _, set := range options.SetValues {
		err = strvals.ParseIntoString(set, outValues)
		if err != nil {
			return "", fmt.Errorf("apply --set %s: %w", set, err)
		}
	}

	// out
	out, err := yaml.Marshal(outValues)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func parseString(str string) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(str), &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func toChartOptions(platformClient platform.Client, options *CreateOptions, log log.Logger) (*vclusterconfig.ExtraValuesOptions, error) {
	if !util.Contains(options.Distro, AllowedDistros) {
		return nil, fmt.Errorf("unsupported distro %s, please select one of: %s", options.Distro, strings.Join(AllowedDistros, ", "))
	}

	kubernetesVersion := vclusterconfig.KubernetesVersion{}
	if options.KubernetesVersion != "" {
		if options.KubernetesVersion[0] != 'v' {
			options.KubernetesVersion = "v" + options.KubernetesVersion
		}

		if !semver.IsValid(options.KubernetesVersion) {
			return nil, fmt.Errorf("please use valid semantic versioning format, e.g. vX.X")
		}

		majorMinorVer := semver.MajorMinor(options.KubernetesVersion)
		if splittedVersion := strings.Split(options.KubernetesVersion, "."); len(splittedVersion) > 2 {
			log.Warnf("currently we only support major.minor version (%s) and not the patch version (%s)", majorMinorVer, options.KubernetesVersion)
		}

		parsedVersion, err := vclusterconfig.ParseKubernetesVersionInfo(majorMinorVer)
		if err != nil {
			return nil, err
		}

		kubernetesVersion.Major = parsedVersion.Major
		kubernetesVersion.Minor = parsedVersion.Minor
	}

	// use default version if its development
	if options.ChartVersion == upgrade.DevelopmentVersion {
		options.ChartVersion = ""
	}

	return &vclusterconfig.ExtraValuesOptions{
		Distro:              options.Distro,
		Expose:              options.Expose,
		KubernetesVersion:   kubernetesVersion,
		DisableTelemetry:    cliconfig.GetConfig(log).TelemetryDisabled,
		InstanceCreatorType: "vclusterctl",
		PlatformInstanceID:  telemetry.GetPlatformInstanceID(platformClient.Self()),
		PlatformUserID:      telemetry.GetPlatformUserID(platformClient.Self()),
		MachineID:           telemetry.GetMachineID(log),
	}, nil
}
