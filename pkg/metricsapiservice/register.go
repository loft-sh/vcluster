package metricsapiservice

import (
	"context"
	"fmt"
	"math"
	"os"
	"path"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/setup/options"
	"github.com/loft-sh/vcluster/pkg/util/applier"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/metrics/pkg/apis/metrics"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

var (
	ErrNoMetricsManifests = fmt.Errorf("no metrics server service manifests found")
)

const (
	ManifestRelativePath = "metrics-server/service.yaml"
	MetricsVersion       = "v1beta1"
	MetricsAPIService    = MetricsVersion + "." + metrics.GroupName // "v1beta1.metrics.k8s.io"
)

func checkExistingAPIService(ctx context.Context, client client.Client) bool {
	var exists bool
	_ = applyOperation(ctx, func(ctx context.Context) (bool, error) {
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
			ObjectMeta: v1.ObjectMeta{
				Name: MetricsAPIService,
			},
		})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			klog.Errorf("error creating api service %v", err)
			return false, nil
		}

		return true, nil
	}
}

func createOperation(ctrlCtx *options.ControllerContext) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		if !ctrlCtx.Options.SingleBinaryDistro {
			// create aux metrics server service in vcluster
			manifestPath := path.Join(constants.ContainerManifestsFolder, ManifestRelativePath)
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				klog.Errorf("error, no metrics server manifest file found %v", ErrNoMetricsManifests)
				return false, nil
			}

			err := applier.ApplyManifestFile(ctrlCtx.Context, ctrlCtx.VirtualManager.GetConfig(), manifestPath)
			if err != nil {
				klog.Errorf("error applying metrics server manifest %v", err)
				return false, nil
			}
		}

		spec := apiregistrationv1.APIServiceSpec{
			Service: &apiregistrationv1.ServiceReference{
				Name:      "metrics-server",
				Namespace: "kube-system",
				Port:      pointer.Int32(443),
			},
			Group:                 metrics.GroupName,
			GroupPriorityMinimum:  100,
			Version:               MetricsVersion,
			VersionPriority:       100,
			InsecureSkipTLSVerify: true,
		}

		apiService := &apiregistrationv1.APIService{
			ObjectMeta: v1.ObjectMeta{
				Name: MetricsAPIService,
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
