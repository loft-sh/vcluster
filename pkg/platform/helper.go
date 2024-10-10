package platform

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/clientset/versioned/scheme"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/platform/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/loft-sh/vcluster/pkg/util"
	"gopkg.in/yaml.v2"
	authorizationv1 "k8s.io/api/authorization/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubectl/pkg/util/term"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var errNoClusterAccess = errors.New("the user has no access to any cluster")

var defaultPollInterval = 40 * time.Second

type VirtualClusterInstanceProject struct {
	VirtualCluster *managementv1.VirtualClusterInstance
	Project        *managementv1.Project
}

type SpaceInstanceProject struct {
	SpaceInstance *managementv1.SpaceInstance
	Project       *managementv1.Project
}

func SelectVirtualClusterTemplate(ctx context.Context, client Client, projectName, templateName string, log log.Logger) (*managementv1.VirtualClusterTemplate, error) {
	managementClient, err := client.Management()
	if err != nil {
		return nil, err
	}

	projectTemplates, err := managementClient.Loft().ManagementV1().Projects().ListTemplates(ctx, projectName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// select default template
	if templateName == "" && projectTemplates.DefaultVirtualClusterTemplate != "" {
		templateName = projectTemplates.DefaultVirtualClusterTemplate
	}

	// try to find template
	if templateName != "" {
		for _, virtualClusterTemplate := range projectTemplates.VirtualClusterTemplates {
			if virtualClusterTemplate.Name == templateName {
				return &virtualClusterTemplate, nil
			}
		}

		return nil, fmt.Errorf("couldn't find template %s as allowed template in project %s", templateName, projectName)
	} else if len(projectTemplates.VirtualClusterTemplates) == 0 {
		return nil, fmt.Errorf("there are no allowed virtual cluster templates in project %s", projectName)
	} else if len(projectTemplates.VirtualClusterTemplates) == 1 {
		return &projectTemplates.VirtualClusterTemplates[0], nil
	}

	templateNames := []string{}
	for _, template := range projectTemplates.VirtualClusterTemplates {
		templateNames = append(templateNames, clihelper.GetDisplayName(template.Name, template.Spec.DisplayName))
	}
	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a template to use",
		DefaultValue: templateNames[0],
		Options:      templateNames,
	})
	if err != nil {
		return nil, err
	}
	for _, template := range projectTemplates.VirtualClusterTemplates {
		if answer == clihelper.GetDisplayName(template.Name, template.Spec.DisplayName) {
			return &template, nil
		}
	}

	return nil, fmt.Errorf("answer not found")
}

func SelectSpaceTemplate(ctx context.Context, client Client, projectName, templateName string, log log.Logger) (*managementv1.SpaceTemplate, error) {
	managementClient, err := client.Management()
	if err != nil {
		return nil, err
	}

	projectTemplates, err := managementClient.Loft().ManagementV1().Projects().ListTemplates(ctx, projectName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// select default template
	if templateName == "" && projectTemplates.DefaultSpaceTemplate != "" {
		templateName = projectTemplates.DefaultSpaceTemplate
	}

	// try to find template
	if templateName != "" {
		for _, spaceTemplate := range projectTemplates.SpaceTemplates {
			if spaceTemplate.Name == templateName {
				return &spaceTemplate, nil
			}
		}

		return nil, fmt.Errorf("couldn't find template %s as allowed template in project %s", templateName, projectName)
	} else if len(projectTemplates.SpaceTemplates) == 0 {
		return nil, fmt.Errorf("there are no allowed space templates in project %s", projectName)
	} else if len(projectTemplates.SpaceTemplates) == 1 {
		return &projectTemplates.SpaceTemplates[0], nil
	}

	templateNames := []string{}
	for _, template := range projectTemplates.SpaceTemplates {
		templateNames = append(templateNames, clihelper.GetDisplayName(template.Name, template.Spec.DisplayName))
	}
	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a template to use",
		DefaultValue: templateNames[0],
		Options:      templateNames,
	})
	if err != nil {
		return nil, err
	}
	for _, template := range projectTemplates.SpaceTemplates {
		if answer == clihelper.GetDisplayName(template.Name, template.Spec.DisplayName) {
			return &template, nil
		}
	}

	return nil, fmt.Errorf("answer not found")
}

func SelectSpaceInstance(ctx context.Context, client Client, spaceName, projectName string, log log.Logger) (string, string, string, error) {
	managementClient, err := client.Management()
	if err != nil {
		return "", "", "", err
	}

	// gather projects and space instances to access
	var spaces []*SpaceInstanceProject
	if projectName != "" {
		project, err := managementClient.Loft().ManagementV1().Projects().Get(ctx, projectName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return "", "", "", fmt.Errorf("couldn't find or access project %s", projectName)
			}

			return "", "", "", err
		}

		// gather space instances in those projects
		if spaceName != "" {
			spaceInstance, err := getProjectSpaceInstance(ctx, managementClient, project, spaceName)
			if err != nil {
				return "", "", "", fmt.Errorf("couldn't find or access space %s", spaceName)
			}

			spaces = append(spaces, spaceInstance)
		} else {
			spaceInstances, err := getProjectSpaceInstances(ctx, managementClient, project.Name)
			if err != nil || len(spaceInstances) == 0 {
				return "", "", "", fmt.Errorf("no space instances found you have access to")
			}

			for _, spaceInstance := range spaceInstances {
				spaces = append(spaces, &SpaceInstanceProject{
					SpaceInstance: spaceInstance,
					Project:       project,
				})
			}
		}
	} else {
		projectsList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
		if err != nil || len(projectsList.Items) == 0 {
			return "", "", "", fmt.Errorf("no projects found you have access to")
		}

		spaceInstances, err := getProjectSpaceInstances(ctx, managementClient, "")
		if err != nil || len(spaceInstances) == 0 {
			return "", "", "", fmt.Errorf("no space instances found you have access to")
		}

		// gather space instances in those projects
		for _, spaceInstance := range spaceInstances {
			if spaceName != "" && spaceInstance.Name != spaceName {
				continue
			}

			// match project
			for _, project := range projectsList.Items {
				if project.Name == projectutil.ProjectFromNamespace(spaceInstance.Namespace) {
					p := project
					spaces = append(spaces, &SpaceInstanceProject{
						SpaceInstance: spaceInstance,
						Project:       &p,
					})
					break
				}
			}
		}
	}

	// get unformatted options
	var optionsUnformatted [][]string
	for _, space := range spaces {
		optionsUnformatted = append(optionsUnformatted, []string{"Space: " + clihelper.GetDisplayName(space.SpaceInstance.Name, space.SpaceInstance.Spec.DisplayName), "Project: " + clihelper.GetDisplayName(space.Project.Name, space.Project.Spec.DisplayName)})
	}

	// check if there are spaces
	if len(spaces) == 0 {
		if spaceName != "" {
			return "", "", "", fmt.Errorf("couldn't find or access space %s", spaceName)
		}
		return "", "", "", fmt.Errorf("couldn't find a space you have access to")
	} else if len(spaces) == 1 {
		return spaces[0].SpaceInstance.Spec.ClusterRef.Cluster, spaces[0].Project.Name, spaces[0].SpaceInstance.Name, nil
	}

	questionOptions := formatOptions("%s | %s", optionsUnformatted)
	selectedOption, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a space",
		DefaultValue: questionOptions[0],
		Options:      questionOptions,
	})
	if err != nil {
		return "", "", "", err
	}

	for idx, s := range questionOptions {
		if s == selectedOption {
			return spaces[idx].SpaceInstance.Spec.ClusterRef.Cluster, spaces[idx].Project.Name, spaces[idx].SpaceInstance.Name, nil
		}
	}

	return "", "", "", fmt.Errorf("couldn't find answer")
}

func SelectProjectOrCluster(ctx context.Context, client Client, clusterName, projectName string, allowClusterOnly bool, log log.Logger) (cluster string, project string, err error) {
	if projectName != "" {
		return clusterName, projectName, nil
	} else if allowClusterOnly && clusterName != "" {
		return clusterName, "", nil
	}

	managementClient, err := client.Management()
	if err != nil {
		return "", "", err
	}

	projectList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", "", err
	}

	projectNames := []string{}
	for _, project := range projectList.Items {
		projectNames = append(projectNames, clihelper.GetDisplayName(project.Name, project.Spec.DisplayName))
	}

	if len(projectNames) == 0 {
		cluster, err := SelectCluster(ctx, client, log)
		if err != nil {
			if errors.Is(err, errNoClusterAccess) {
				return "", "", fmt.Errorf("the user has no access to a project")
			}

			return "", "", err
		}

		return cluster, "", nil
	}

	var selectedProject *managementv1.Project
	if len(projectNames) == 1 {
		selectedProject = &projectList.Items[0]
	} else {
		answer, err := log.Question(&survey.QuestionOptions{
			Question:     "Please choose a project to use",
			DefaultValue: projectNames[0],
			Options:      projectNames,
		})
		if err != nil {
			return "", "", err
		}
		for idx, project := range projectList.Items {
			if answer == clihelper.GetDisplayName(project.Name, project.Spec.DisplayName) {
				selectedProject = &projectList.Items[idx]
			}
		}
		if selectedProject == nil {
			return "", "", fmt.Errorf("answer not found")
		}
	}

	if clusterName == "" {
		clusterName, err = SelectProjectCluster(ctx, client, selectedProject, log)
		return clusterName, selectedProject.Name, err
	}

	return clusterName, selectedProject.Name, nil
}

// SelectCluster lets the user select a cluster
func SelectCluster(ctx context.Context, client Client, log log.Logger) (string, error) {
	managementClient, err := client.Management()
	if err != nil {
		return "", err
	}

	clusterList, err := managementClient.Loft().ManagementV1().Clusters().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	clusterNames := []string{}
	for _, cluster := range clusterList.Items {
		clusterNames = append(clusterNames, clihelper.GetDisplayName(cluster.Name, cluster.Spec.DisplayName))
	}

	if len(clusterList.Items) == 0 {
		return "", errNoClusterAccess
	} else if len(clusterList.Items) == 1 {
		return clusterList.Items[0].Name, nil
	}

	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a cluster to use",
		DefaultValue: clusterNames[0],
		Options:      clusterNames,
	})
	if err != nil {
		return "", err
	}
	for _, cluster := range clusterList.Items {
		if answer == clihelper.GetDisplayName(cluster.Name, cluster.Spec.DisplayName) {
			return cluster.Name, nil
		}
	}
	return "", fmt.Errorf("answer not found")
}

// SelectProjectCluster lets the user select a cluster from the project's allowed clusters
func SelectProjectCluster(ctx context.Context, client Client, project *managementv1.Project, log log.Logger) (string, error) {
	if !term.IsTerminal(os.Stdin) {
		// Allow loft to schedule as before
		return "", nil
	}

	managementClient, err := client.Management()
	if err != nil {
		return "", err
	}

	clusterList, err := managementClient.Loft().ManagementV1().Projects().ListClusters(ctx, project.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	anyClusterOption := "Any cluster [The platform will select a cluster for you]"
	clusterNames := []string{}
	for _, allowedCluster := range project.Spec.AllowedClusters {
		if allowedCluster.Name == "*" {
			clusterNames = append(clusterNames, anyClusterOption)
			break
		}
	}

	for _, cluster := range clusterList.Clusters {
		clusterNames = append(clusterNames, clihelper.GetDisplayName(cluster.Name, cluster.Spec.DisplayName))
	}

	if len(clusterList.Clusters) == 0 {
		return "", errNoClusterAccess
	} else if len(clusterList.Clusters) == 1 {
		return clusterList.Clusters[0].Name, nil
	}

	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a cluster to use",
		DefaultValue: clusterNames[0],
		Options:      clusterNames,
	})
	if err != nil {
		return "", err
	}

	if answer == anyClusterOption {
		return "", nil
	}

	for _, cluster := range clusterList.Clusters {
		if answer == clihelper.GetDisplayName(cluster.Name, cluster.Spec.DisplayName) {
			return cluster.Name, nil
		}
	}
	return "", fmt.Errorf("answer not found")
}

func CanAccessProjectSecret(ctx context.Context, managementClient kube.Interface, namespace, name string) (bool, error) {
	return CanAccessInstance(ctx, managementClient, namespace, name, "projectsecrets")
}

func CanAccessInstance(ctx context.Context, managementClient kube.Interface, namespace, name string, resource string) (bool, error) {
	selfSubjectAccessReview, err := managementClient.Loft().ManagementV1().SelfSubjectAccessReviews().Create(ctx, &managementv1.SelfSubjectAccessReview{
		Spec: managementv1.SelfSubjectAccessReviewSpec{
			SelfSubjectAccessReviewSpec: authorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Verb:      "use",
					Group:     managementv1.SchemeGroupVersion.Group,
					Version:   managementv1.SchemeGroupVersion.Version,
					Resource:  resource,
					Namespace: namespace,
					Name:      name,
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return false, err
	} else if !selfSubjectAccessReview.Status.Allowed || selfSubjectAccessReview.Status.Denied {
		return false, nil
	}
	return true, nil
}

func GetSpaceInstances(ctx context.Context, client Client) ([]*SpaceInstanceProject, error) {
	managementClient, err := client.Management()
	if err != nil {
		return nil, err
	}

	projectsList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	spaceInstances, err := getProjectSpaceInstances(ctx, managementClient, "")
	if err != nil {
		return nil, err
	}

	// gather space instances in those projects
	var retSpaces []*SpaceInstanceProject
	for _, spaceInstance := range spaceInstances {
		// match project
		for _, project := range projectsList.Items {
			if project.Name == projectutil.ProjectFromNamespace(spaceInstance.Namespace) {
				p := project
				retSpaces = append(retSpaces, &SpaceInstanceProject{
					SpaceInstance: spaceInstance,
					Project:       &p,
				})
				break
			}
		}
	}

	return retSpaces, nil
}

type ProjectProjectSecret struct {
	Project       string
	ProjectSecret managementv1.ProjectSecret
}

func (vci *VirtualClusterInstanceProject) IsInstanceSleeping() bool {
	return vci != nil && vci.VirtualCluster != nil && sleepmode.IsInstanceSleeping(vci.VirtualCluster)
}

func GetProjectSecrets(ctx context.Context, managementClient kube.Interface, projectNames ...string) ([]*ProjectProjectSecret, error) {
	var projects []*managementv1.Project
	if len(projectNames) == 0 {
		projectList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for idx := range projectList.Items {
			projectItem := projectList.Items[idx]
			projects = append(projects, &projectItem)
		}
	} else {
		for _, projectName := range projectNames {
			project, err := managementClient.Loft().ManagementV1().Projects().Get(ctx, projectName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			projects = append(projects, project)
		}
	}

	var retSecrets []*ProjectProjectSecret
	for _, project := range projects {
		projectSecrets, err := managementClient.Loft().ManagementV1().ProjectSecrets(projectutil.ProjectNamespace(project.Name)).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, projectSecret := range projectSecrets.Items {
			canAccess, err := CanAccessProjectSecret(ctx, managementClient, projectSecret.Namespace, projectSecret.Name)
			if err != nil {
				return nil, err
			} else if !canAccess {
				continue
			}

			retSecrets = append(retSecrets, &ProjectProjectSecret{
				ProjectSecret: projectSecret,
				Project:       project.Name,
			})
		}
	}

	return retSecrets, nil
}

func GetCurrentUser(ctx context.Context, managementClient kube.Interface) (*managementv1.UserInfo, *storagev1.EntityInfo, error) {
	self, err := managementClient.Loft().ManagementV1().Selves().Create(ctx, &managementv1.Self{}, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("get self: %w", err)
	} else if self.Status.User == nil && self.Status.Team == nil {
		return nil, nil, fmt.Errorf("no user or team name returned")
	}

	return self.Status.User, self.Status.Team, nil
}

func WaitForSpaceInstance(ctx context.Context, managementClient kube.Interface, namespace, name string, waitUntilReady bool, log log.Logger) (*managementv1.SpaceInstance, error) {
	pollInterval := min(clihelper.Timeout()/5, defaultPollInterval)
	now := time.Now()
	nextMessage := now.Add(pollInterval)
	spaceInstance, err := managementClient.Loft().ManagementV1().SpaceInstances(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if spaceInstance.Status.Phase == storagev1.InstanceSleeping {
		log.Info("Wait until space wakes up")
		defer log.Donef("Successfully woken up space %s", name)
		err := wakeupSpace(ctx, managementClient, spaceInstance)
		if err != nil {
			return nil, fmt.Errorf("error waking up space %s: %s", name, util.GetCause(err))
		}
	}

	if !waitUntilReady {
		return spaceInstance, nil
	}

	logged := false
	return spaceInstance, wait.PollUntilContextTimeout(ctx, time.Second, clihelper.Timeout(), true, func(ctx context.Context) (bool, error) {
		spaceInstance, err = managementClient.Loft().ManagementV1().SpaceInstances(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if spaceInstance.Status.Phase != storagev1.InstanceReady && spaceInstance.Status.Phase != storagev1.InstanceSleeping {
			if time.Now().After(nextMessage) {
				if logged {
					log.Infof("Cannot reach space because: %s (%s). Loft will continue waiting, but this operation may timeout", spaceInstance.Status.Message, spaceInstance.Status.Reason)
				} else {
					log.Info("Waiting for space to be available...")
				}
				nextMessage = time.Now().Add(pollInterval)
				logged = true
			}
			return false, nil
		}

		return true, nil
	})
}

func CreateVirtualClusterInstanceOptions(ctx context.Context, client Client, config string, projectName string, virtualClusterInstance *managementv1.VirtualClusterInstance, setActive bool) (kubeconfig.ContextOptions, error) {
	var cluster *managementv1.Cluster

	// skip finding cluster if virtual cluster is directly connected
	if !virtualClusterInstance.Spec.NetworkPeer {
		var err error
		cluster, err = findProjectCluster(ctx, client, projectName, virtualClusterInstance.Spec.ClusterRef.Cluster)
		if err != nil {
			return kubeconfig.ContextOptions{}, fmt.Errorf("find space instance cluster: %w", err)
		}
	}

	contextOptions := kubeconfig.ContextOptions{
		Name:       kubeconfig.VirtualClusterInstanceContextName(projectName, virtualClusterInstance.Name),
		ConfigPath: config,
		SetActive:  setActive,
	}
	contextOptions.Server = client.Config().Platform.Host + "/kubernetes/project/" + projectName + "/virtualcluster/" + virtualClusterInstance.Name
	contextOptions.InsecureSkipTLSVerify = client.Config().Platform.Insecure

	data, err := RetrieveCaData(cluster)
	if err != nil {
		return kubeconfig.ContextOptions{}, err
	}
	contextOptions.CaData = data
	return contextOptions, nil
}

func CreateSpaceInstanceOptions(ctx context.Context, client Client, config string, projectName string, spaceInstance *managementv1.SpaceInstance, setActive bool) (kubeconfig.ContextOptions, error) {
	cluster, err := findProjectCluster(ctx, client, projectName, spaceInstance.Spec.ClusterRef.Cluster)
	if err != nil {
		return kubeconfig.ContextOptions{}, fmt.Errorf("find space instance cluster: %w", err)
	}

	contextOptions := kubeconfig.ContextOptions{
		Name:             kubeconfig.SpaceInstanceContextName(projectName, spaceInstance.Name),
		ConfigPath:       config,
		CurrentNamespace: spaceInstance.Spec.ClusterRef.Namespace,
		SetActive:        setActive,
	}
	contextOptions.Server = client.Config().Platform.Host + "/kubernetes/project/" + projectName + "/space/" + spaceInstance.Name
	contextOptions.InsecureSkipTLSVerify = client.Config().Platform.Insecure

	data, err := RetrieveCaData(cluster)
	if err != nil {
		return kubeconfig.ContextOptions{}, err
	}
	contextOptions.CaData = data
	return contextOptions, nil
}

func ResolveVirtualClusterTemplate(
	ctx context.Context,
	client Client,
	project,
	template,
	templateVersion string,
	setParams []string,
	fileParams string,
	log log.Logger,
) (*managementv1.VirtualClusterTemplate, string, error) {
	// determine space template to use
	virtualClusterTemplate, err := SelectVirtualClusterTemplate(ctx, client, project, template, log)
	if err != nil {
		return nil, "", err
	}

	// get parameters
	var templateParameters []storagev1.AppParameter
	if len(virtualClusterTemplate.Spec.Versions) > 0 {
		if templateVersion == "" {
			latestVersion := GetLatestVersion(virtualClusterTemplate)
			if latestVersion == nil {
				return nil, "", fmt.Errorf("couldn't find any version in template")
			}

			templateParameters = latestVersion.(*storagev1.VirtualClusterTemplateVersion).Parameters
		} else {
			_, latestMatched, err := GetLatestMatchedVersion(virtualClusterTemplate, templateVersion)
			if err != nil {
				return nil, "", err
			} else if latestMatched == nil {
				return nil, "", fmt.Errorf("couldn't find any matching version to %s", templateVersion)
			}

			templateParameters = latestMatched.(*storagev1.VirtualClusterTemplateVersion).Parameters
		}
	} else {
		templateParameters = virtualClusterTemplate.Spec.Parameters
	}

	// resolve space template parameters
	resolvedParameters, err := ResolveTemplateParameters(setParams, templateParameters, fileParams)
	if err != nil {
		return nil, "", err
	}

	return virtualClusterTemplate, resolvedParameters, nil
}

func ResolveTemplateParameters(set []string, parameters []storagev1.AppParameter, fileName string) (string, error) {
	var parametersFile map[string]interface{}
	if fileName != "" {
		out, err := os.ReadFile(fileName)
		if err != nil {
			return "", fmt.Errorf("read parameters file: %w", err)
		}

		parametersFile = map[string]interface{}{}
		err = yaml.Unmarshal(out, &parametersFile)
		if err != nil {
			return "", fmt.Errorf("parse parameters file: %w", err)
		}
	}

	return fillParameters(parameters, set, parametersFile)
}

func SetDeepValue(parameters interface{}, path string, value interface{}) {
	if parameters == nil {
		return
	}

	pathSegments := strings.Split(path, ".")
	switch t := parameters.(type) {
	case map[string]interface{}:
		if len(pathSegments) == 1 {
			t[pathSegments[0]] = value
			return
		}

		_, ok := t[pathSegments[0]]
		if !ok {
			t[pathSegments[0]] = map[string]interface{}{}
		}

		SetDeepValue(t[pathSegments[0]], strings.Join(pathSegments[1:], "."), value)
	}
}

func GetDeepValue(parameters interface{}, path string) interface{} {
	if parameters == nil {
		return nil
	}

	pathSegments := strings.Split(path, ".")
	switch t := parameters.(type) {
	case map[string]interface{}:
		val, ok := t[pathSegments[0]]
		if !ok {
			return nil
		} else if len(pathSegments) == 1 {
			return val
		}

		return GetDeepValue(val, strings.Join(pathSegments[1:], "."))
	case []interface{}:
		index, err := strconv.Atoi(pathSegments[0])
		if err != nil {
			return nil
		} else if index < 0 || index >= len(t) {
			return nil
		}

		val := t[index]
		if len(pathSegments) == 1 {
			return val
		}

		return GetDeepValue(val, strings.Join(pathSegments[1:], "."))
	}

	return nil
}

func VerifyValue(value string, parameter storagev1.AppParameter) (interface{}, error) {
	switch parameter.Type {
	case "":
		fallthrough
	case "password":
		fallthrough
	case "string":
		fallthrough
	case "multiline":
		if parameter.DefaultValue != "" && value == "" {
			value = parameter.DefaultValue
		}

		if parameter.Required && value == "" {
			return nil, fmt.Errorf("parameter %s (%s) is required", parameter.Label, parameter.Variable)
		}
		for _, option := range parameter.Options {
			if option == value {
				return value, nil
			}
		}
		if parameter.Validation != "" {
			regEx, err := regexp.Compile(parameter.Validation)
			if err != nil {
				return nil, fmt.Errorf("compile validation regex %s: %w", parameter.Validation, err)
			}

			if !regEx.MatchString(value) {
				return nil, fmt.Errorf("parameter %s (%s) needs to match regex %s", parameter.Label, parameter.Variable, parameter.Validation)
			}
		}
		if parameter.Invalidation != "" {
			regEx, err := regexp.Compile(parameter.Invalidation)
			if err != nil {
				return nil, fmt.Errorf("compile invalidation regex %s: %w", parameter.Invalidation, err)
			}

			if regEx.MatchString(value) {
				return nil, fmt.Errorf("parameter %s (%s) cannot match regex %s", parameter.Label, parameter.Variable, parameter.Invalidation)
			}
		}

		return value, nil
	case "boolean":
		if parameter.DefaultValue != "" && value == "" {
			boolValue, err := strconv.ParseBool(parameter.DefaultValue)
			if err != nil {
				return nil, fmt.Errorf("parse default value for parameter %s (%s): %w", parameter.Label, parameter.Variable, err)
			}

			return boolValue, nil
		}
		if parameter.Required && value == "" {
			return nil, fmt.Errorf("parameter %s (%s) is required", parameter.Label, parameter.Variable)
		}

		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("parse value for parameter %s (%s): %w", parameter.Label, parameter.Variable, err)
		}
		return boolValue, nil
	case "number":
		if parameter.DefaultValue != "" && value == "" {
			intValue, err := strconv.Atoi(parameter.DefaultValue)
			if err != nil {
				return nil, fmt.Errorf("parse default value for parameter %s (%s): %w", parameter.Label, parameter.Variable, err)
			}

			return intValue, nil
		}
		if parameter.Required && value == "" {
			return nil, fmt.Errorf("parameter %s (%s) is required", parameter.Label, parameter.Variable)
		}
		num, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("parse value for parameter %s (%s): %w", parameter.Label, parameter.Variable, err)
		}
		if parameter.Min != nil && num < *parameter.Min {
			return nil, fmt.Errorf("parameter %s (%s) cannot be smaller than %d", parameter.Label, parameter.Variable, *parameter.Min)
		}
		if parameter.Max != nil && num > *parameter.Max {
			return nil, fmt.Errorf("parameter %s (%s) cannot be greater than %d", parameter.Label, parameter.Variable, *parameter.Max)
		}

		return num, nil
	}

	return nil, fmt.Errorf("unrecognized type %s for parameter %s (%s)", parameter.Type, parameter.Label, parameter.Variable)
}

func RetrieveCaData(cluster *managementv1.Cluster) ([]byte, error) {
	if cluster == nil || cluster.Annotations == nil || cluster.Annotations[LoftDirectClusterEndpointCaData] == "" {
		return nil, nil
	}

	data, err := base64.StdEncoding.DecodeString(cluster.Annotations[LoftDirectClusterEndpointCaData])
	if err != nil {
		return nil, fmt.Errorf("error decoding cluster %s annotation: %w", LoftDirectClusterEndpointCaData, err)
	}

	return data, nil
}

// ListVClusters lists all virtual clusters across all projects if virtualClusterName and projectName are empty.
// The list can be narrowed down by the given virtual cluster name and project name.
func ListVClusters(ctx context.Context, client Client, virtualClusterName, projectName string) ([]*VirtualClusterInstanceProject, error) {
	managementClient, err := client.Management()
	if err != nil {
		return nil, err
	}

	// gather projects and virtual cluster instances to access
	virtualClusters := []*VirtualClusterInstanceProject{}
	if projectName != "" {
		project, err := managementClient.Loft().ManagementV1().Projects().Get(ctx, projectName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, fmt.Errorf("couldn't find or access project %s", projectName)
			}

			return nil, err
		}

		// gather space instances in those projects
		if virtualClusterName != "" {
			virtualClusterInstance, err := getProjectVirtualClusterInstance(ctx, managementClient, project, virtualClusterName)
			if err == nil {
				virtualClusters = append(virtualClusters, virtualClusterInstance)
			}
		} else {
			virtualClusterInstances, err := getProjectVirtualClusterInstances(ctx, managementClient, project.Name)
			if err != nil {
				return nil, err
			}

			for _, virtualClusterInstance := range virtualClusterInstances {
				virtualClusters = append(virtualClusters, &VirtualClusterInstanceProject{
					VirtualCluster: virtualClusterInstance,
					Project:        project,
				})
			}
		}
	} else {
		projectsList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
		if err != nil || len(projectsList.Items) == 0 {
			return nil, err
		}

		virtualClusterInstances, err := getProjectVirtualClusterInstances(ctx, managementClient, "")
		if err != nil || len(virtualClusterInstances) == 0 {
			return nil, err
		}

		// gather space instances in those projects
		for _, virtualClusterInstance := range virtualClusterInstances {
			if virtualClusterName != "" && virtualClusterInstance.Name != virtualClusterName {
				continue
			}

			// match project
			for _, project := range projectsList.Items {
				if project.Name == projectutil.ProjectFromNamespace(virtualClusterInstance.Namespace) {
					p := project
					virtualClusters = append(virtualClusters, &VirtualClusterInstanceProject{
						VirtualCluster: virtualClusterInstance,
						Project:        &p,
					})
					break
				}
			}
		}
	}

	return virtualClusters, nil
}

func WaitForVirtualClusterInstance(ctx context.Context, managementClient kube.Interface, namespace, name string, waitUntilReady bool, log log.Logger) (*managementv1.VirtualClusterInstance, error) {
	now := time.Now()
	pollInterval := min(clihelper.Timeout()/5, defaultPollInterval)
	nextMessage := now.Add(pollInterval)
	virtualClusterInstance, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if virtualClusterInstance.Status.Phase == storagev1.InstanceSleeping {
		log.Info("Wait until vcluster instance wakes up")
		defer log.Donef("virtual cluster %s wakeup successful", name)
		err := wakeupVCluster(ctx, managementClient, virtualClusterInstance)
		if err != nil {
			return nil, fmt.Errorf("error waking up vcluster %s: %s", name, util.GetCause(err))
		}
	}

	if !waitUntilReady {
		return virtualClusterInstance, nil
	}

	logged := false
	return virtualClusterInstance, wait.PollUntilContextTimeout(ctx, time.Second, clihelper.Timeout(), true, func(ctx context.Context) (bool, error) {
		virtualClusterInstance, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if virtualClusterInstance.Status.Phase != storagev1.InstanceReady && virtualClusterInstance.Status.Phase != storagev1.InstanceSleeping {
			if time.Now().After(nextMessage) {
				if logged {
					log.Infof("Cannot reach virtual cluster because: %s (%s). Loft will continue waiting, but this operation may timeout", virtualClusterInstance.Status.Message, virtualClusterInstance.Status.Reason)
				} else {
					log.Info("Waiting for virtual cluster to be available...")
				}
				nextMessage = time.Now().Add(pollInterval)
				logged = true
			}
			return false, nil
		}

		return true, nil
	})
}

func fillParameters(parameters []storagev1.AppParameter, set []string, values map[string]interface{}) (string, error) {
	if values == nil {
		values = map[string]interface{}{}
	}

	// parse set array
	setMap, err := parseSet(parameters, set)
	if err != nil {
		return "", err
	}

	// apply parameters
	for _, parameter := range parameters {
		strVal, ok := setMap[parameter.Variable]
		if !ok {
			val := GetDeepValue(values, parameter.Variable)
			if val != nil {
				switch t := val.(type) {
				case string:
					strVal = t
				case int:
					strVal = strconv.Itoa(t)
				case bool:
					strVal = strconv.FormatBool(t)
				default:
					return "", fmt.Errorf("unrecognized type for parameter %s (%s) in file: %v", parameter.Label, parameter.Variable, t)
				}
			}
		}

		outVal, err := VerifyValue(strVal, parameter)
		if err != nil {
			return "", fmt.Errorf("validate parameters %w", err)
		}

		SetDeepValue(values, parameter.Variable, outVal)
	}

	out, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal parameters: %w", err)
	}

	return string(out), nil
}

func parseSet(parameters []storagev1.AppParameter, set []string) (map[string]string, error) {
	setValues := map[string]string{}
	for _, s := range set {
		splitted := strings.Split(s, "=")
		if len(splitted) <= 1 {
			return nil, fmt.Errorf("error parsing --set %s: need parameter=value format", s)
		}

		key := splitted[0]
		value := strings.Join(splitted[1:], "=")
		found := false
		for _, parameter := range parameters {
			if parameter.Variable == key {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("parameter %s doesn't exist on template", key)
		}

		setValues[key] = value
	}

	return setValues, nil
}

func wakeupVCluster(ctx context.Context, managementClient kube.Interface, virtualClusterInstance *managementv1.VirtualClusterInstance) error {
	// Update instance to wake up
	oldVirtualClusterInstance := virtualClusterInstance.DeepCopy()
	if virtualClusterInstance.Annotations == nil {
		virtualClusterInstance.Annotations = map[string]string{}
	}
	delete(virtualClusterInstance.Annotations, clusterv1.SleepModeForceAnnotation)
	delete(virtualClusterInstance.Annotations, clusterv1.SleepModeForceDurationAnnotation)
	virtualClusterInstance.Annotations[clusterv1.SleepModeLastActivityAnnotation] = strconv.FormatInt(time.Now().Unix(), 10)

	// Calculate patch
	patch := crclient.MergeFrom(oldVirtualClusterInstance)
	patchData, err := patch.Data(virtualClusterInstance)
	if err != nil {
		return err
	}

	_, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterInstance.Namespace).Patch(ctx, virtualClusterInstance.Name, patch.Type(), patchData, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}

func findProjectCluster(ctx context.Context, client Client, projectName, clusterName string) (*managementv1.Cluster, error) {
	managementClient, err := client.Management()
	if err != nil {
		return nil, err
	}

	projectClusters, err := managementClient.Loft().ManagementV1().Projects().ListClusters(ctx, projectName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("list project clusters: %w", err)
	}

	for _, cluster := range projectClusters.Clusters {
		if cluster.Name == clusterName {
			return &cluster, nil
		}
	}

	return nil, fmt.Errorf("couldn't find cluster %s in project %s", clusterName, projectName)
}

func wakeupSpace(ctx context.Context, managementClient kube.Interface, spaceInstance *managementv1.SpaceInstance) error {
	// Update instance to wake up
	oldSpaceInstance := spaceInstance.DeepCopy()
	if spaceInstance.Annotations == nil {
		spaceInstance.Annotations = map[string]string{}
	}
	delete(spaceInstance.Annotations, clusterv1.SleepModeForceAnnotation)
	delete(spaceInstance.Annotations, clusterv1.SleepModeForceDurationAnnotation)
	spaceInstance.Annotations[clusterv1.SleepModeLastActivityAnnotation] = strconv.FormatInt(time.Now().Unix(), 10)

	// Calculate patch
	patch := crclient.MergeFrom(oldSpaceInstance)
	patchData, err := patch.Data(spaceInstance)
	if err != nil {
		return err
	}

	// Patch the instance
	_, err = managementClient.Loft().ManagementV1().SpaceInstances(spaceInstance.Namespace).Patch(ctx, spaceInstance.Name, patch.Type(), patchData, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}

func formatOptions(format string, options [][]string) []string {
	if len(options) == 0 {
		return []string{}
	}

	columnLengths := make([]int, len(options[0]))
	for _, row := range options {
		for i, column := range row {
			if len(column) > columnLengths[i] {
				columnLengths[i] = len(column)
			}
		}
	}

	retOptions := []string{}
	for _, row := range options {
		columns := []interface{}{}
		for i := range row {
			value := row[i]
			if columnLengths[i] > len(value) {
				value = value + strings.Repeat(" ", columnLengths[i]-len(value))
			}

			columns = append(columns, value)
		}

		retOptions = append(retOptions, fmt.Sprintf(format, columns...))
	}

	return retOptions
}

func getProjectSpaceInstance(ctx context.Context, managementClient kube.Interface, project *managementv1.Project, spaceName string) (*SpaceInstanceProject, error) {
	spaceInstance := &managementv1.SpaceInstance{}
	err := managementClient.Loft().ManagementV1().RESTClient().
		Get().
		Resource("spaceinstances").
		Namespace(projectutil.ProjectNamespace(project.Name)).
		Name(spaceName).
		VersionedParams(&metav1.GetOptions{}, scheme.ParameterCodec).
		Param("extended", "true").
		Do(ctx).
		Into(spaceInstance)
	if err != nil {
		return nil, err
	}

	if !spaceInstance.Status.CanUse {
		return nil, fmt.Errorf("no use access")
	}

	return &SpaceInstanceProject{
		SpaceInstance: spaceInstance,
		Project:       project,
	}, nil
}

func getProjectSpaceInstances(ctx context.Context, managementClient kube.Interface, projectName string) ([]*managementv1.SpaceInstance, error) {
	spaceInstanceList := &managementv1.SpaceInstanceList{}
	request := managementClient.Loft().ManagementV1().RESTClient().
		Get().
		Resource("spaceinstances").
		VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
		Param("extended", "true")
	if projectName != "" {
		request = request.Namespace(projectutil.ProjectNamespace(projectName))
	}

	err := request.Do(ctx).Into(spaceInstanceList)
	if err != nil {
		return nil, err
	}

	var spaces []*managementv1.SpaceInstance
	for _, spaceInstance := range spaceInstanceList.Items {
		if !spaceInstance.Status.CanUse {
			continue
		}

		s := spaceInstance
		spaces = append(spaces, &s)
	}
	return spaces, nil
}

func getProjectVirtualClusterInstance(ctx context.Context, managementClient kube.Interface, project *managementv1.Project, virtualClusterName string) (*VirtualClusterInstanceProject, error) {
	virtualClusterInstance := &managementv1.VirtualClusterInstance{}
	err := managementClient.Loft().ManagementV1().RESTClient().
		Get().
		Resource("virtualclusterinstances").
		Namespace(projectutil.ProjectNamespace(project.Name)).
		Name(virtualClusterName).
		VersionedParams(&metav1.GetOptions{}, scheme.ParameterCodec).
		Param("extended", "true").
		Do(ctx).
		Into(virtualClusterInstance)
	if err != nil {
		return nil, err
	}

	if !virtualClusterInstance.Status.CanUse {
		return nil, fmt.Errorf("no use access")
	}

	return &VirtualClusterInstanceProject{
		VirtualCluster: virtualClusterInstance,
		Project:        project,
	}, nil
}

func getProjectVirtualClusterInstances(ctx context.Context, managementClient kube.Interface, projectName string) ([]*managementv1.VirtualClusterInstance, error) {
	virtualClusterInstanceList := &managementv1.VirtualClusterInstanceList{}
	request := managementClient.Loft().ManagementV1().RESTClient().
		Get().
		Resource("virtualclusterinstances").
		VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
		Param("extended", "true")
	if projectName != "" {
		request = request.Namespace(projectutil.ProjectNamespace(projectName))
	}
	err := request.Do(ctx).Into(virtualClusterInstanceList)
	if err != nil {
		return nil, err
	}

	var virtualClusters []*managementv1.VirtualClusterInstance
	for _, virtualClusterInstance := range virtualClusterInstanceList.Items {
		if !virtualClusterInstance.Status.CanUse {
			continue
		}

		v := virtualClusterInstance
		virtualClusters = append(virtualClusters, &v)
	}
	return virtualClusters, nil
}
