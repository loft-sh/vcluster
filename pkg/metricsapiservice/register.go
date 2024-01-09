package metricsapiservice

import (
	"context"
	"math"
	"time"

	"github.com/loft-sh/vcluster/pkg/setup/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/metrics/pkg/apis/metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	ManifestRelativePath  = "metrics-server/service.yaml"
	MetricsVersion        = "v1beta1"
	MetricsAPIServiceName = MetricsVersion + "." + metrics.GroupName // "v1beta1.metrics.k8s.io"

	AuxVirtualSvcName      = "metrics-server"
	AuxVirtualSvcNamespace = "kube-system"
)

func checkExistingAPIService(ctx context.Context, client client.Client) bool {
	var exists bool
	_ = applyOperation(ctx, func(ctx context.Context) (bool, error) {
		err := client.Get(ctx, types.NamespacedName{Name: MetricsAPIServiceName}, &apiregistrationv1.APIService{})
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

func applyOperation(ctx context.Context, operationFunc wait.ConditionWithContextFunc) error {
	return wait.ExponentialBackoffWithContext(ctx, wait.Backoff{
		Duration: time.Second,
		Factor:   1.5,
		Cap:      time.Minute,
		Steps:    math.MaxInt32,
	}, operationFunc)
}

func deleteOperation(ctrlCtx *options.ControllerContext) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		err := ctrlCtx.VirtualManager.GetClient().Delete(ctx, &apiregistrationv1.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: MetricsAPIServiceName,
			},
		})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			klog.Errorf("error deleting api service %v", err)
			return false, nil
		}

		return true, nil
	}
}

func createOperation(ctrlCtx *options.ControllerContext) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		spec := apiregistrationv1.APIServiceSpec{
			Group:                metrics.GroupName,
			GroupPriorityMinimum: 100,
			Version:              MetricsVersion,
			VersionPriority:      100,
		}

		apiService := &apiregistrationv1.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: MetricsAPIServiceName,
			},
		}

		_, err := controllerutil.CreateOrUpdate(ctx, ctrlCtx.VirtualManager.GetClient(), apiService, func() error {
			apiService.Spec = spec
			return nil
		})
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return true, nil
			}

			klog.Errorf("error creating api service %v", err)
			return false, nil
		}

		return true, nil
	}
}

func RegisterOrDeregisterAPIService(ctx *options.ControllerContext) error {
	// check if the api service should get created
	exists := checkExistingAPIService(ctx.Context, ctx.VirtualManager.GetClient())
	if ctx.Options.ProxyMetricsServer {
		return applyOperation(ctx.Context, createOperation(ctx))
	} else if !ctx.Options.ProxyMetricsServer && exists {
		return applyOperation(ctx.Context, deleteOperation(ctx))
	}

	return nil
}
