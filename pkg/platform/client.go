package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/api/v4/pkg/client/clientset_generated/clientset/scheme"
	loftclient "github.com/loft-sh/loftctl/v4/pkg/client"
	"github.com/loft-sh/loftctl/v4/pkg/client/naming"
	"github.com/loft-sh/loftctl/v4/pkg/kube"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var Self *managementv1.Self

var selfOnce sync.Once

type Client interface {
	loftclient.Client

	Self() *managementv1.Self
	ApplyPlatformSecret(ctx context.Context, kubeClient kubernetes.Interface, name, namespace, project string) error
	ListVClusters(ctx context.Context, virtualClusterName, projectName string) ([]VirtualClusterInstanceProject, error)
}

type VirtualClusterInstanceProject struct {
	VirtualCluster *managementv1.VirtualClusterInstance
	Project        *managementv1.Project
}

var ErrConfigNotFound = errors.New("couldn't find vCluster platform config")

func CreatePlatformClient() (Client, error) {
	configPath, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(configPath)
	if err != nil {
		return nil, fmt.Errorf("%w: please make sure to run 'vcluster login' to connect to an existing instance or 'vcluster platform start' to deploy a new instance", ErrConfigNotFound)
	}

	platformClient, err := loftclient.NewClientFromPath(configPath)
	if err != nil {
		return nil, err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return nil, fmt.Errorf("error creating pro client: %w", err)
	}

	self, err := managementClient.Loft().ManagementV1().Selves().Create(context.TODO(), &managementv1.Self{}, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "the server rejected our request for an unknown reason") {
			return nil, fmt.Errorf("vCluster platform instance is not reachable at %s, please make sure you are correctly logged in via 'vcluster login'", platformClient.Config().Host)
		}

		return nil, fmt.Errorf("get self: %w", err)
	} else if self.Status.User == nil && self.Status.Team == nil {
		return nil, fmt.Errorf("no user or team name returned for vCluster platform credentials")
	}

	selfOnce.Do(func() {
		Self = self
	})
	return &client{
		Client: platformClient,

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

// ListVClusters lists all virtual clusters across all projects if virtualClusterName and projectName are empty.
// The list can be narrowed down by the given virtual cluster name and project name.
func (c *client) ListVClusters(ctx context.Context, virtualClusterName, projectName string) ([]VirtualClusterInstanceProject, error) {
	managementClient, err := c.Management()
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

	// gather virtual cluster instances in those projects
	virtualClusters := []VirtualClusterInstanceProject{}
	for _, p := range projects {
		if virtualClusterName != "" {
			virtualClusterInstance, err := getProjectVirtualClusterInstance(ctx, managementClient, p, virtualClusterName)
			if err != nil {
				continue
			}

			virtualClusters = append(virtualClusters, virtualClusterInstance)
		} else {
			virtualClusters, err = getProjectVirtualClusterInstances(ctx, managementClient, p)
			if err != nil {
				continue
			}
		}
	}

	return virtualClusters, nil
}

func getProjectVirtualClusterInstance(ctx context.Context, managementClient kube.Interface, project *managementv1.Project, virtualClusterName string) (VirtualClusterInstanceProject, error) {
	virtualClusterInstance := &managementv1.VirtualClusterInstance{}
	err := managementClient.Loft().ManagementV1().RESTClient().
		Get().
		Resource("virtualclusterinstances").
		Namespace(naming.ProjectNamespace(project.Name)).
		Name(virtualClusterName).
		VersionedParams(&metav1.GetOptions{}, scheme.ParameterCodec).
		Param("extended", "true").
		Do(ctx).
		Into(virtualClusterInstance)
	if err != nil {
		return VirtualClusterInstanceProject{}, err
	}

	if !virtualClusterInstance.Status.CanUse {
		return VirtualClusterInstanceProject{}, fmt.Errorf("no use access")
	}

	return VirtualClusterInstanceProject{
		VirtualCluster: virtualClusterInstance,
		Project:        project,
	}, nil
}

func getProjectVirtualClusterInstances(ctx context.Context, managementClient kube.Interface, project *managementv1.Project) ([]VirtualClusterInstanceProject, error) {
	virtualClusterInstanceList := &managementv1.VirtualClusterInstanceList{}
	err := managementClient.Loft().ManagementV1().RESTClient().
		Get().
		Resource("virtualclusterinstances").
		Namespace(naming.ProjectNamespace(project.Name)).
		VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
		Param("extended", "true").
		Do(ctx).
		Into(virtualClusterInstanceList)
	if err != nil {
		return nil, err
	}

	var virtualClusters []VirtualClusterInstanceProject
	for _, virtualClusterInstance := range virtualClusterInstanceList.Items {
		if !virtualClusterInstance.Status.CanUse {
			continue
		}

		v := virtualClusterInstance
		virtualClusters = append(virtualClusters, VirtualClusterInstanceProject{
			VirtualCluster: &v,
			Project:        project,
		})
	}
	return virtualClusters, nil
}
