package metricsapiservice

import (
	"context"
	"math"
	"time"

	vclustercontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/metrics/pkg/apis/metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	MetricsVersion    = "v1beta1"
	MetricsAPIService = MetricsVersion + "." + metrics.GroupName // "v1beta1.metrics.k8s.io"
)

func checkExistingAPIService(ctx context.Context, client client.Client) bool {
	var exists bool
	_ = applyOperation(ctx, func() (bool, error) {
		err := client.Get(ctx, types.NamespacedName{Name: MetricsAPIService}, &apiregistrationv1.APIService{})
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

func deleteOperation(ctx context.Context, client client.Client) wait.ConditionFunc {
	return func() (bool, error) {
		err := client.Delete(ctx, &apiregistrationv1.APIService{
			ObjectMeta: v1.ObjectMeta{
				Name: MetricsAPIService,
			},
		})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		return true, nil
	}
}

func createOperation(ctx context.Context, client client.Client) wait.ConditionFunc {
	return func() (bool, error) {
		spec := apiregistrationv1.APIServiceSpec{
			Group:                metrics.GroupName,
			GroupPriorityMinimum: 100,
			Version:              MetricsVersion,
			VersionPriority:      100,
		}

		apiService := &apiregistrationv1.APIService{
			ObjectMeta: v1.ObjectMeta{
				Name: MetricsAPIService,
			},
		}

		_, err := controllerutil.CreateOrUpdate(ctx, client, apiService, func() error {
			apiService.Spec = spec
			return nil
		})
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return true, nil
			}

			klog.Errorf("error creating api service %v", err)
			return false, err
		}

		return true, nil
	}
}

func RegisterOrDeregisterAPIService(ctx context.Context, options *vclustercontext.VirtualClusterOptions, client client.Client) error {
	// check if the api service should get created
	exists := checkExistingAPIService(ctx, client)
	if options.ProxyMetricsServer {
		return applyOperation(ctx, createOperation(ctx, client))
	} else if !options.ProxyMetricsServer && exists {
		return applyOperation(ctx, deleteOperation(ctx, client))
	}

	return nil
}
