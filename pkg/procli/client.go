package procli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	loftclient "github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var Self *managementv1.Self

var selfOnce sync.Once

type Client interface {
	loftclient.Client

	Self() *managementv1.Self
}

type VirtualClusterInstanceProject struct {
	VirtualCluster *managementv1.VirtualClusterInstance
	Project        *managementv1.Project
}

var ErrConfigNotFound = errors.New("couldn't find vCluster.Pro config")

func CreateProClient() (Client, error) {
	configPath, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(configPath)
	if err != nil {
		return nil, fmt.Errorf("%w: please make sure to run 'vcluster login' to connect to an existing instance or 'vcluster pro start' to deploy a new instance", ErrConfigNotFound)
	}

	proClient, err := loftclient.NewClientFromPath(configPath)
	if err != nil {
		return nil, err
	}

	managementClient, err := proClient.Management()
	if err != nil {
		return nil, fmt.Errorf("error creating pro client: %w", err)
	}

	self, err := managementClient.Loft().ManagementV1().Selves().Create(context.TODO(), &managementv1.Self{}, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "the server rejected our request for an unknown reason") {
			return nil, fmt.Errorf("vCluster.Pro instance is not reachable at %s, please make sure you are correctly logged in via 'vcluster login'", proClient.Config().Host)
		}

		return nil, fmt.Errorf("get self: %w", err)
	} else if self.Status.User == nil && self.Status.Team == nil {
		return nil, fmt.Errorf("no user or team name returned for vCluster.Pro credentials")
	}

	selfOnce.Do(func() {
		Self = self
	})
	return &client{
		Client: proClient,

		self: self,
	}, nil
}

type client struct {
	loftclient.Client

	self *managementv1.Self
}

func (c *client) Self() *managementv1.Self {
	return c.self.DeepCopy()
}

func ListVClusters(ctx context.Context, baseClient Client, virtualClusterName, projectName string) ([]VirtualClusterInstanceProject, error) {
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
