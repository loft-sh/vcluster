package helper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/loft-sh/api/v4/pkg/client/clientset_generated/clientset/scheme"
	authorizationv1 "k8s.io/api/authorization/v1"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/loftctl/v4/pkg/client"
	"github.com/loft-sh/loftctl/v4/pkg/clihelper"
	"github.com/loft-sh/loftctl/v4/pkg/kube"
	"github.com/loft-sh/loftctl/v4/pkg/kubeconfig"
	"github.com/loft-sh/loftctl/v4/pkg/projectutil"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/mgutz/ansi"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/util/term"
)

var errNoClusterAccess = errors.New("the user has no access to any cluster")

type VirtualClusterInstanceProject struct {
	VirtualCluster *managementv1.VirtualClusterInstance
	Project        *managementv1.Project
}

type SpaceInstanceProject struct {
	SpaceInstance *managementv1.SpaceInstance
	Project       *managementv1.Project
}

func SelectVirtualClusterTemplate(ctx context.Context, baseClient client.Client, projectName, templateName string, log log.Logger) (*managementv1.VirtualClusterTemplate, error) {
	managementClient, err := baseClient.Management()
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

func SelectSpaceTemplate(ctx context.Context, baseClient client.Client, projectName, templateName string, log log.Logger) (*managementv1.SpaceTemplate, error) {
	managementClient, err := baseClient.Management()
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

func SelectVirtualClusterInstanceOrVirtualCluster(ctx context.Context, baseClient client.Client, virtualClusterName, spaceName, projectName, clusterName string, log log.Logger) (string, string, string, string, error) {
	if clusterName != "" || spaceName != "" {
		virtualCluster, space, cluster, err := SelectVirtualClusterAndSpaceAndClusterName(ctx, baseClient, virtualClusterName, spaceName, clusterName, log)
		return cluster, "", space, virtualCluster, err
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return "", "", "", "", err
	}

	// gather projects and virtual cluster instances to access
	var projects []*managementv1.Project
	if projectName != "" {
		project, err := managementClient.Loft().ManagementV1().Projects().Get(ctx, projectName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return "", "", "", "", fmt.Errorf("couldn't find or access project %s", projectName)
			}

			return "", "", "", "", err
		}

		projects = append(projects, project)
	} else {
		projectsList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
		if err != nil || len(projectsList.Items) == 0 {
			virtualCluster, space, cluster, err := SelectVirtualClusterAndSpaceAndClusterName(ctx, baseClient, virtualClusterName, spaceName, clusterName, log)
			return cluster, "", space, virtualCluster, err
		}

		for _, p := range projectsList.Items {
			proj := p
			projects = append(projects, &proj)
		}
	}

	// gather space instances in those projects
	var virtualClusters []*VirtualClusterInstanceProject
	for _, p := range projects {
		if virtualClusterName != "" {
			virtualClusterInstance, err := getProjectVirtualClusterInstance(ctx, managementClient, p, virtualClusterName)
			if err != nil {
				continue
			}

			virtualClusters = append(virtualClusters, virtualClusterInstance)
		} else {
			projectVirtualClusters, err := getProjectVirtualClusterInstances(ctx, managementClient, p)
			if err != nil {
				continue
			}

			virtualClusters = append(virtualClusters, projectVirtualClusters...)
		}
	}

	// get unformatted options
	var optionsUnformatted [][]string
	for _, virtualCluster := range virtualClusters {
		optionsUnformatted = append(optionsUnformatted, []string{"vcluster: " + clihelper.GetDisplayName(virtualCluster.VirtualCluster.Name, virtualCluster.VirtualCluster.Spec.DisplayName), "Project: " + clihelper.GetDisplayName(virtualCluster.Project.Name, virtualCluster.Project.Spec.DisplayName)})
	}

	// check if there are virtualclusters
	if len(virtualClusters) == 0 {
		if virtualClusterName != "" {
			return "", "", "", "", fmt.Errorf("couldn't find or access virtual cluster %s", virtualClusterName)
		}
		return "", "", "", "", fmt.Errorf("couldn't find a virtual cluster you have access to")
	} else if len(virtualClusters) == 1 {
		return "", virtualClusters[0].Project.Name, "", virtualClusters[0].VirtualCluster.Name, nil
	}

	questionOptions := formatOptions("%s | %s", optionsUnformatted)
	selectedOption, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a virtual cluster",
		DefaultValue: questionOptions[0],
		Options:      questionOptions,
	})
	if err != nil {
		return "", "", "", "", err
	}

	for idx, s := range questionOptions {
		if s == selectedOption {
			return "", virtualClusters[idx].Project.Name, "", virtualClusters[idx].VirtualCluster.Name, nil
		}
	}

	return "", "", "", "", fmt.Errorf("couldn't find answer")
}

func SelectSpaceInstanceOrSpace(ctx context.Context, baseClient client.Client, spaceName, projectName, clusterName string, log log.Logger) (string, string, string, error) {
	if clusterName != "" {
		space, cluster, err := SelectSpaceAndClusterName(ctx, baseClient, spaceName, clusterName, log)
		return cluster, "", space, err
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return "", "", "", err
	}

	// gather projects and space instances to access
	var projects []*managementv1.Project
	if projectName != "" {
		project, err := managementClient.Loft().ManagementV1().Projects().Get(ctx, projectName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return "", "", "", fmt.Errorf("couldn't find or access project %s", projectName)
			}

			return "", "", "", err
		}

		projects = append(projects, project)
	} else {
		projectsList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
		if err != nil || len(projectsList.Items) == 0 {
			space, cluster, err := SelectSpaceAndClusterName(ctx, baseClient, spaceName, clusterName, log)
			return cluster, "", space, err
		}

		for _, p := range projectsList.Items {
			proj := p
			projects = append(projects, &proj)
		}
	}

	// gather space instances in those projects
	var spaces []*SpaceInstanceProject
	for _, p := range projects {
		if spaceName != "" {
			spaceInstance, err := getProjectSpaceInstance(ctx, managementClient, p, spaceName)
			if err != nil {
				continue
			}

			spaces = append(spaces, spaceInstance)
		} else {
			projectSpaceInstances, err := getProjectSpaceInstances(ctx, managementClient, p)
			if err != nil {
				continue
			}

			spaces = append(spaces, projectSpaceInstances...)
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

func SelectProjectOrCluster(ctx context.Context, baseClient client.Client, clusterName, projectName string, allowClusterOnly bool, log log.Logger) (cluster string, project string, err error) {
	if projectName != "" {
		return clusterName, projectName, nil
	} else if allowClusterOnly && clusterName != "" {
		return clusterName, "", nil
	}

	managementClient, err := baseClient.Management()
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
		cluster, err := SelectCluster(ctx, baseClient, log)
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
		clusterName, err = SelectProjectCluster(ctx, baseClient, selectedProject, log)
		return clusterName, selectedProject.Name, err
	}

	return clusterName, selectedProject.Name, nil
}

// SelectCluster lets the user select a cluster
func SelectCluster(ctx context.Context, baseClient client.Client, log log.Logger) (string, error) {
	managementClient, err := baseClient.Management()
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
func SelectProjectCluster(ctx context.Context, baseClient client.Client, project *managementv1.Project, log log.Logger) (string, error) {
	if !term.IsTerminal(os.Stdin) {
		// Allow loft to schedule as before
		return "", nil
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return "", err
	}

	clusterList, err := managementClient.Loft().ManagementV1().Projects().ListClusters(ctx, project.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	anyClusterOption := "Any Cluster [Loft Selects Cluster]"
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

// SelectUserOrTeam lets the user select an user or team in a cluster
func SelectUserOrTeam(ctx context.Context, baseClient client.Client, clusterName string, log log.Logger) (*clusterv1.EntityInfo, *clusterv1.EntityInfo, error) {
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, nil, err
	}

	clusterAccess, err := managementClient.Loft().ManagementV1().Clusters().ListAccess(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	var user *clusterv1.EntityInfo
	if len(clusterAccess.Users) > 0 {
		user = &clusterAccess.Users[0].Info
	}

	teams := []*clusterv1.EntityInfo{}
	for _, team := range clusterAccess.Teams {
		t := team
		teams = append(teams, &t.Info)
	}

	if user == nil && len(teams) == 0 {
		return nil, nil, fmt.Errorf("the user has no access to cluster %s", clusterName)
	} else if user != nil && len(teams) == 0 {
		return user, nil, nil
	} else if user == nil && len(teams) == 1 {
		return nil, teams[0], nil
	}

	names := []string{}
	if user != nil {
		names = append(names, "User "+clihelper.DisplayName(user))
	}
	for _, t := range teams {
		names = append(names, "Team "+clihelper.DisplayName(t))
	}

	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a user or team to use",
		DefaultValue: names[0],
		Options:      names,
	})
	if err != nil {
		return nil, nil, err
	}

	if user != nil && "User "+clihelper.DisplayName(user) == answer {
		return user, nil, nil
	}
	for _, t := range teams {
		if "Team "+clihelper.DisplayName(t) == answer {
			return nil, t, nil
		}
	}

	return nil, nil, fmt.Errorf("answer not found")
}

type ClusterUserOrTeam struct {
	Team          bool
	ClusterMember managementv1.ClusterMember
}

func SelectClusterUserOrTeam(ctx context.Context, baseClient client.Client, clusterName, userName, teamName string, log log.Logger) (*ClusterUserOrTeam, error) {
	if userName != "" && teamName != "" {
		return nil, fmt.Errorf("team and user specified, please only choose one")
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, err
	}

	members, err := managementClient.Loft().ManagementV1().Clusters().ListMembers(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("retrieve cluster members: %w", err)
	}

	matchedMembers := []ClusterUserOrTeam{}
	optionsUnformatted := [][]string{}
	for _, user := range members.Users {
		if teamName != "" {
			continue
		} else if userName != "" && user.Info.Name != userName {
			continue
		}

		matchedMembers = append(matchedMembers, ClusterUserOrTeam{
			ClusterMember: user,
		})
		displayName := user.Info.DisplayName
		if displayName == "" {
			displayName = user.Info.Name
		}

		optionsUnformatted = append(optionsUnformatted, []string{"User: " + displayName, "Kube User: " + user.Info.Name})
	}
	for _, team := range members.Teams {
		if userName != "" {
			continue
		} else if teamName != "" && team.Info.Name != teamName {
			continue
		}

		matchedMembers = append(matchedMembers, ClusterUserOrTeam{
			Team:          true,
			ClusterMember: team,
		})
		displayName := team.Info.DisplayName
		if displayName == "" {
			displayName = team.Info.Name
		}

		optionsUnformatted = append(optionsUnformatted, []string{"Team: " + displayName, "Kube Team: " + team.Info.Name})
	}

	questionOptions := formatOptions("%s | %s", optionsUnformatted)
	if len(questionOptions) == 0 {
		if userName == "" && teamName == "" {
			return nil, fmt.Errorf("couldn't find any space")
		} else if userName != "" {
			return nil, fmt.Errorf("couldn't find user %s in cluster %s", ansi.Color(userName, "white+b"), ansi.Color(clusterName, "white+b"))
		}

		return nil, fmt.Errorf("couldn't find team %s in cluster %s", ansi.Color(teamName, "white+b"), ansi.Color(clusterName, "white+b"))
	} else if len(questionOptions) == 1 {
		return &matchedMembers[0], nil
	}

	selectedMember, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a user or team",
		DefaultValue: questionOptions[0],
		Options:      questionOptions,
	})
	if err != nil {
		return nil, err
	}

	for idx, s := range questionOptions {
		if s == selectedMember {
			return &matchedMembers[idx], nil
		}
	}

	return nil, fmt.Errorf("selected question option not found")
}

func GetVirtualClusterInstances(ctx context.Context, baseClient client.Client) ([]*VirtualClusterInstanceProject, error) {
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, err
	}

	projectList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var retVClusters []*VirtualClusterInstanceProject
	for _, project := range projectList.Items {
		p := &project

		virtualClusterInstances, err := getProjectVirtualClusterInstances(ctx, managementClient, p)
		if err != nil {
			return nil, err
		}

		retVClusters = append(retVClusters, virtualClusterInstances...)
	}

	return retVClusters, nil
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

func GetSpaceInstances(ctx context.Context, baseClient client.Client) ([]*SpaceInstanceProject, error) {
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, err
	}

	projectList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var retSpaces []*SpaceInstanceProject
	for _, project := range projectList.Items {
		p := &project

		spaceInstances, err := getProjectSpaceInstances(ctx, managementClient, p)
		if err != nil {
			return nil, err
		}

		retSpaces = append(retSpaces, spaceInstances...)
	}

	return retSpaces, nil
}

type ProjectProjectSecret struct {
	ProjectSecret managementv1.ProjectSecret
	Project       string
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

type ClusterSpace struct {
	clusterv1.Space
	Cluster string
}

// GetSpaces returns all spaces accessible by the user or team
func GetSpaces(ctx context.Context, baseClient client.Client, log log.Logger) ([]ClusterSpace, error) {
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, err
	}

	clusterList, err := managementClient.Loft().ManagementV1().Clusters().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	spaceList := []ClusterSpace{}
	for _, cluster := range clusterList.Items {
		clusterClient, err := baseClient.Cluster(cluster.Name)
		if err != nil {
			return nil, err
		}

		spaces, err := clusterClient.Agent().ClusterV1().Spaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			if kerrors.IsForbidden(err) {
				continue
			}

			log.Warnf("Error retrieving spaces from cluster %s: %v", clihelper.GetDisplayName(cluster.Name, cluster.Spec.DisplayName), err)
			continue
		}

		for _, space := range spaces.Items {
			spaceList = append(spaceList, ClusterSpace{
				Space:   space,
				Cluster: cluster.Name,
			})
		}
	}
	sort.Slice(spaceList, func(i, j int) bool {
		return spaceList[i].Name < spaceList[j].Name
	})

	return spaceList, nil
}

type ClusterVirtualCluster struct {
	clusterv1.VirtualCluster
	Cluster string
}

// GetVirtualClusters returns all virtual clusters the user has access to
func GetVirtualClusters(ctx context.Context, baseClient client.Client, log log.Logger) ([]ClusterVirtualCluster, error) {
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, err
	}

	clusterList, err := managementClient.Loft().ManagementV1().Clusters().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	virtualClusterList := []ClusterVirtualCluster{}
	for _, cluster := range clusterList.Items {
		clusterClient, err := baseClient.Cluster(cluster.Name)
		if err != nil {
			return nil, err
		}

		virtualClusters, err := clusterClient.Agent().ClusterV1().VirtualClusters("").List(ctx, metav1.ListOptions{})
		if err != nil {
			if kerrors.IsForbidden(err) {
				continue
			}

			log.Warnf("Error retrieving virtual clusters from cluster %s: %v", clihelper.GetDisplayName(cluster.Name, cluster.Spec.DisplayName), err)
			continue
		}

		for _, virtualCluster := range virtualClusters.Items {
			virtualClusterList = append(virtualClusterList, ClusterVirtualCluster{
				VirtualCluster: virtualCluster,
				Cluster:        cluster.Name,
			})
		}
	}
	sort.Slice(virtualClusterList, func(i, j int) bool {
		return virtualClusterList[i].Name < virtualClusterList[j].Name
	})

	return virtualClusterList, nil
}

// SelectSpaceAndClusterName selects a space and cluster name
func SelectSpaceAndClusterName(ctx context.Context, baseClient client.Client, spaceName, clusterName string, log log.Logger) (string, string, error) {
	spaces, err := GetSpaces(ctx, baseClient, log)
	if err != nil {
		return "", "", err
	}

	currentContext, err := kubeconfig.CurrentContext()
	if err != nil {
		return "", "", fmt.Errorf("loading kubernetes config: %w", err)
	}

	isLoftContext, cluster, namespace, vCluster := kubeconfig.ParseContext(currentContext)
	matchedSpaces := []ClusterSpace{}
	questionOptionsUnformatted := [][]string{}
	defaultIndex := 0
	for _, space := range spaces {
		if spaceName != "" && space.Space.Name != spaceName {
			continue
		} else if clusterName != "" && space.Cluster != clusterName {
			continue
		} else if len(matchedSpaces) > 20 {
			break
		}

		if isLoftContext && vCluster == "" && cluster == space.Cluster && namespace == space.Space.Name {
			defaultIndex = len(questionOptionsUnformatted)
		}

		matchedSpaces = append(matchedSpaces, space)
		spaceName := space.Space.Name
		if space.Space.Annotations != nil && space.Space.Annotations["loft.sh/display-name"] != "" {
			spaceName = space.Space.Annotations["loft.sh/display-name"] + " (" + spaceName + ")"
		}

		questionOptionsUnformatted = append(questionOptionsUnformatted, []string{spaceName, space.Cluster})
	}

	questionOptions := formatOptions("Space: %s | Cluster: %s", questionOptionsUnformatted)
	if len(questionOptions) == 0 {
		if spaceName == "" {
			return "", "", fmt.Errorf("couldn't find any space")
		} else if clusterName != "" {
			return "", "", fmt.Errorf("couldn't find space %s in cluster %s", ansi.Color(spaceName, "white+b"), ansi.Color(clusterName, "white+b"))
		}

		return "", "", fmt.Errorf("couldn't find space %s", ansi.Color(spaceName, "white+b"))
	} else if len(questionOptions) == 1 {
		return matchedSpaces[0].Space.Name, matchedSpaces[0].Cluster, nil
	}

	selectedSpace, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a space",
		DefaultValue: questionOptions[defaultIndex],
		Options:      questionOptions,
	})
	if err != nil {
		return "", "", err
	}

	for idx, s := range questionOptions {
		if s == selectedSpace {
			clusterName = matchedSpaces[idx].Cluster
			spaceName = matchedSpaces[idx].Space.Name
			break
		}
	}

	return spaceName, clusterName, nil
}

func GetCurrentUser(ctx context.Context, managementClient kube.Interface) (*managementv1.UserInfo, *clusterv1.EntityInfo, error) {
	self, err := managementClient.Loft().ManagementV1().Selves().Create(ctx, &managementv1.Self{}, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("get self: %w", err)
	} else if self.Status.User == nil && self.Status.Team == nil {
		return nil, nil, fmt.Errorf("no user or team name returned")
	}

	return self.Status.User, self.Status.Team, nil
}

func SelectVirtualClusterAndSpaceAndClusterName(ctx context.Context, baseClient client.Client, virtualClusterName, spaceName, clusterName string, log log.Logger) (string, string, string, error) {
	virtualClusters, err := GetVirtualClusters(ctx, baseClient, log)
	if err != nil {
		return "", "", "", err
	}

	currentContext, err := kubeconfig.CurrentContext()
	if err != nil {
		return "", "", "", fmt.Errorf("loading kubernetes config: %w", err)
	}

	isLoftContext, cluster, namespace, vCluster := kubeconfig.ParseContext(currentContext)
	matchedVClusters := []ClusterVirtualCluster{}
	questionOptionsUnformatted := [][]string{}
	defaultIndex := 0
	for _, virtualCluster := range virtualClusters {
		if virtualClusterName != "" && virtualCluster.VirtualCluster.Name != virtualClusterName {
			continue
		} else if spaceName != "" && virtualCluster.VirtualCluster.Namespace != spaceName {
			continue
		} else if clusterName != "" && virtualCluster.Cluster != clusterName {
			continue
		}

		if isLoftContext && vCluster == virtualCluster.VirtualCluster.Name && cluster == virtualCluster.Cluster && namespace == virtualCluster.VirtualCluster.Namespace {
			defaultIndex = len(questionOptionsUnformatted)
		}

		matchedVClusters = append(matchedVClusters, virtualCluster)
		vClusterName := virtualCluster.VirtualCluster.Name
		if virtualCluster.VirtualCluster.Annotations != nil && virtualCluster.VirtualCluster.Annotations["loft.sh/display-name"] != "" {
			vClusterName = virtualCluster.VirtualCluster.Annotations["loft.sh/display-name"] + " (" + vClusterName + ")"
		}

		questionOptionsUnformatted = append(questionOptionsUnformatted, []string{vClusterName, virtualCluster.VirtualCluster.Namespace, virtualCluster.Cluster})
	}

	questionOptions := formatOptions("vCluster: %s | Space: %s | Cluster: %s", questionOptionsUnformatted)
	if len(questionOptions) == 0 {
		if virtualClusterName == "" {
			return "", "", "", fmt.Errorf("couldn't find any virtual cluster")
		} else if spaceName != "" {
			return "", "", "", fmt.Errorf("couldn't find virtualcluster %s in space %s in cluster %s", ansi.Color(virtualClusterName, "white+b"), ansi.Color(spaceName, "white+b"), ansi.Color(clusterName, "white+b"))
		} else if clusterName != "" {
			return "", "", "", fmt.Errorf("couldn't find virtualcluster %s in space %s in cluster %s", ansi.Color(virtualClusterName, "white+b"), ansi.Color(spaceName, "white+b"), ansi.Color(clusterName, "white+b"))
		}

		return "", "", "", fmt.Errorf("couldn't find virtual cluster %s", ansi.Color(virtualClusterName, "white+b"))
	} else if len(questionOptions) == 1 {
		return matchedVClusters[0].VirtualCluster.Name, matchedVClusters[0].VirtualCluster.Namespace, matchedVClusters[0].Cluster, nil
	}

	selectedSpace, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a virtual cluster to use",
		DefaultValue: questionOptions[defaultIndex],
		Options:      questionOptions,
	})
	if err != nil {
		return "", "", "", err
	}

	for idx, s := range questionOptions {
		if s == selectedSpace {
			clusterName = matchedVClusters[idx].Cluster
			virtualClusterName = matchedVClusters[idx].VirtualCluster.Name
			spaceName = matchedVClusters[idx].VirtualCluster.Namespace
			break
		}
	}

	return virtualClusterName, spaceName, clusterName, nil
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

func getProjectSpaceInstances(ctx context.Context, managementClient kube.Interface, project *managementv1.Project) ([]*SpaceInstanceProject, error) {
	spaceInstanceList := &managementv1.SpaceInstanceList{}
	err := managementClient.Loft().ManagementV1().RESTClient().
		Get().
		Resource("spaceinstances").
		Namespace(projectutil.ProjectNamespace(project.Name)).
		VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
		Param("extended", "true").
		Do(ctx).
		Into(spaceInstanceList)
	if err != nil {
		return nil, err
	}

	var spaces []*SpaceInstanceProject
	for _, spaceInstance := range spaceInstanceList.Items {
		if !spaceInstance.Status.CanUse {
			continue
		}

		s := spaceInstance
		spaces = append(spaces, &SpaceInstanceProject{
			SpaceInstance: &s,
			Project:       project,
		})
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

func getProjectVirtualClusterInstances(ctx context.Context, managementClient kube.Interface, project *managementv1.Project) ([]*VirtualClusterInstanceProject, error) {
	virtualClusterInstanceList := &managementv1.VirtualClusterInstanceList{}
	err := managementClient.Loft().ManagementV1().RESTClient().
		Get().
		Resource("virtualclusterinstances").
		Namespace(projectutil.ProjectNamespace(project.Name)).
		VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
		Param("extended", "true").
		Do(ctx).
		Into(virtualClusterInstanceList)
	if err != nil {
		return nil, err
	}

	var virtualClusters []*VirtualClusterInstanceProject
	for _, virtualClusterInstance := range virtualClusterInstanceList.Items {
		if !virtualClusterInstance.Status.CanUse {
			continue
		}

		v := virtualClusterInstance
		virtualClusters = append(virtualClusters, &VirtualClusterInstanceProject{
			VirtualCluster: &v,
			Project:        project,
		})
	}
	return virtualClusters, nil
}
