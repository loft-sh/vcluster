package pro

import (
	"context"
	"errors"
	"fmt"
	"os"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VirtualClusterInstanceProject struct {
	VirtualCluster *managementv1.VirtualClusterInstance
	Project        *managementv1.Project
}

var ErrConfigNotFound = errors.New("couldn't find vCluster.Pro config")

func CreateProClient() (client.Client, error) {
	configPath, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(configPath)
	if err != nil {
		return nil, fmt.Errorf("%w: please make sure to run 'vcluster login' to connect to an existing instance or 'vcluster pro start' to deploy a new instance", ErrConfigNotFound)
	}

	proClient, err := client.NewClientFromPath(configPath)
	if err != nil {
		return nil, err
	}

	return proClient, nil
}

func ListVClusters(ctx context.Context, baseClient client.Client, virtualClusterName, projectName string) ([]VirtualClusterInstanceProject, error) {
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, err
	}

	// gather projects and virtual cluster instances to access
	projects := []*managementv1.Project{}
	if projectName != "" {
		project, err := managementClient.Loft().ManagementV1().Projects().Get(ctx, projectName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, fmt.Errorf("couldn't find or access project %s", projectName)
			}

			return nil, err
		}

		projects = append(projects, project)
	} else {
		projectsList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
		if err != nil || len(projectsList.Items) == 0 {
			return nil, err
		}

		for _, p := range projectsList.Items {
			proj := p
			projects = append(projects, &proj)
		}
	}

	// gather space instances in those projects
	virtualClusters := []VirtualClusterInstanceProject{}
	for _, p := range projects {
		if virtualClusterName != "" {
			virtualClusterInstance, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(naming.ProjectNamespace(p.Name)).Get(ctx, virtualClusterName, metav1.GetOptions{})
			if err != nil {
				continue
			}

			virtualClusters = append(virtualClusters, VirtualClusterInstanceProject{
				VirtualCluster: virtualClusterInstance,
				Project:        p,
			})
		} else {
			virtualClusterInstanceList, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(naming.ProjectNamespace(p.Name)).List(ctx, metav1.ListOptions{})
			if err != nil {
				continue
			}

			for _, virtualClusterInstance := range virtualClusterInstanceList.Items {
				s := virtualClusterInstance
				virtualClusters = append(virtualClusters, VirtualClusterInstanceProject{
					VirtualCluster: &s,
					Project:        p,
				})
			}
		}
	}

	// filter out virtual clusters we cannot access
	newVirtualClusters := []VirtualClusterInstanceProject{}
	for _, virtualCluster := range virtualClusters {
		canAccess, err := helper.CanAccessVirtualClusterInstance(managementClient, virtualCluster.VirtualCluster.Namespace, virtualCluster.VirtualCluster.Name)
		if err != nil {
			return nil, err
		} else if !canAccess {
			continue
		}

		newVirtualClusters = append(newVirtualClusters, virtualCluster)
	}

	return newVirtualClusters, nil
}
