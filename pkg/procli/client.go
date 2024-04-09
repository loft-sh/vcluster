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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var Self *managementv1.Self

var selfOnce sync.Once

type Client interface {
	loftclient.Client

	Self() *managementv1.Self
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
