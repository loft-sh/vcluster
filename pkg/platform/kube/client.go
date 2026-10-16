package kube

import (
	agentloftclient "github.com/loft-sh/agentapi/v4/pkg/clientset/versioned"
	loftclient "github.com/loft-sh/api/v4/pkg/clientset/versioned"

	"github.com/pkg/errors"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Interface interface {
	kubernetes.Interface
	Loft() loftclient.Interface
	Agent() agentloftclient.Interface
}

func NewForConfig(c *rest.Config) (Interface, error) {
	kubeClient, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, errors.Wrap(err, "create kube client")
	}

	loftClient, err := loftclient.NewForConfig(c)
	if err != nil {
		return nil, errors.Wrap(err, "create loft client")
	}

	agentLoftClient, err := agentloftclient.NewForConfig(c)
	if err != nil {
		return nil, errors.Wrap(err, "create kiosk client")
	}

	return &client{
		Interface:       kubeClient,
		loftClient:      loftClient,
		agentLoftClient: agentLoftClient,
	}, nil
}

type client struct {
	kubernetes.Interface
	loftClient      loftclient.Interface
	agentLoftClient agentloftclient.Interface
}

func (c *client) Loft() loftclient.Interface {
	return c.loftClient
}

func (c *client) Agent() agentloftclient.Interface {
	return c.agentLoftClient
}
