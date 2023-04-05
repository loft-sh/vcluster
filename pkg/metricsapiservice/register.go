package metricsapiservice

import (
	"context"
	"math"
	"time"

	vclustercontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/metrics/pkg/apis/metrics"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	apiregclientv1 "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
)

const (
	MetricsVersion    = "v1beta1"
	MetricsAPIService = MetricsVersion + "." + metrics.GroupName // "v1beta1.metrics.k8s.io"

	KubernetesSvc = "kubernetes"
)

func checkExistingAPIService(ctx context.Context, kubeAggClient *apiregclientv1.ApiregistrationV1Client) bool {
	var exists bool

	_ = applyOperation(ctx, func() (bool, error) {
		_, err := kubeAggClient.APIServices().Get(ctx, MetricsAPIService, v1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		exists = true
		return true, nil
	})

	return exists
}

func applyOperation(ctx context.Context, operationFunc wait.ConditionFunc) error {
	return wait.ExponentialBackoffWithContext(ctx, wait.Backoff{
		Duration: time.Second,
		Factor:   1.5,
		Cap:      time.Minute,
		Steps:    math.MaxInt32,
	}, operationFunc)
}

func deleteOperation(ctx context.Context, kubeAggClient *apiregclientv1.ApiregistrationV1Client) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeAggClient.APIServices().Delete(ctx, MetricsAPIService, v1.DeleteOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		return true, nil
	}
}

func createOperation(ctx context.Context, kubeAggClient *apiregclientv1.ApiregistrationV1Client) wait.ConditionFunc {
	return func() (bool, error) {
		_, err := kubeAggClient.APIServices().Create(ctx, &apiregistrationv1.APIService{
			ObjectMeta: v1.ObjectMeta{
				Name: MetricsAPIService,
			},
			Spec: apiregistrationv1.APIServiceSpec{
				Group:                metrics.GroupName,
				GroupPriorityMinimum: 100,
				Version:              MetricsVersion,
				VersionPriority:      100,
			},
		}, v1.CreateOptions{})
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return true, nil
			}

			return false, err
		}

		return true, nil
	}
}

func RegisterOrDeregisterAPIService(ctx context.Context, options *vclustercontext.VirtualClusterOptions, vConfig *rest.Config) error {
	kubeAggClient, err := apiregclientv1.NewForConfig(vConfig)
	if err != nil {
		return err
	}

	exists := checkExistingAPIService(ctx, kubeAggClient)
	if options.ProxyMetricsServer && !exists {
		// register apiservice
		return applyOperation(ctx, createOperation(ctx, kubeAggClient))
	}

	if !options.ProxyMetricsServer && exists {
		// delete apiservice
		return applyOperation(ctx, deleteOperation(ctx, kubeAggClient))
	}

	return nil
}
