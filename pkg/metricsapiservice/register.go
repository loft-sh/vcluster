package metricsapiservice

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/loft-sh/vcluster/pkg/setup/options"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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
		if !ctrlCtx.Options.SingleBinaryDistro {
			auxVirtualSvc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuxVirtualSvcName,
					Namespace: AuxVirtualSvcNamespace,
				},
			}
			err := ctrlCtx.VirtualManager.GetClient().Delete(ctx, auxVirtualSvc)
			if err != nil {
				if !kerrors.IsNotFound(err) {
					return false, nil
				}
			}

			hostMetricsProxySvc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ctrlCtx.Options.ServiceName + "-metrics-proxy",
					Namespace: ctrlCtx.CurrentNamespace,
				},
			}
			err = ctrlCtx.LocalManager.GetClient().Delete(ctx, hostMetricsProxySvc)
			if err != nil {
				if !kerrors.IsNotFound(err) {
					return false, nil
				}
			}
		}

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

		if !ctrlCtx.Options.SingleBinaryDistro {
			// in this case we register an apiservice with a service reference object
			// this service is created as a special service and the physical-virtual
			// pair makes sure the service discovery happens as expected in even non single
			// binary distros like k8s and eks
			spec.Service = &apiregistrationv1.ServiceReference{
				Name:      AuxVirtualSvcName,
				Namespace: AuxVirtualSvcNamespace,
				Port:      pointer.Int32(443),
			}
			spec.InsecureSkipTLSVerify = true

			// create aux metrics server service in vcluster
			auxVirtualSvc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuxVirtualSvcName,
					Namespace: AuxVirtualSvcNamespace,
					Labels: map[string]string{
						"k8s-app": "metrics-server",
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:       "https",
							Port:       443,
							Protocol:   "TCP",
							TargetPort: intstr.FromInt(8443),
						},
					},
					Selector: map[string]string{
						"k8s-app": "metrics-server",
					},
					SessionAffinity: corev1.ServiceAffinityNone,
					Type:            corev1.ServiceTypeClusterIP,
				},
			}
			err := ctrlCtx.VirtualManager.GetClient().Create(ctx, auxVirtualSvc)
			if err != nil {
				if !kerrors.IsAlreadyExists(err) {
					klog.Errorf("error creating metrics server service inside vcluster %v", err)
					return false, nil
				}
			}

			// create aux metrics service in host cluster
			hostMetricsProxySvc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ctrlCtx.Options.ServiceName + "-metrics-proxy",
					Namespace: ctrlCtx.CurrentNamespace,
					Labels: map[string]string{
						"app":     "vcluster-metrics-proxy",
						"release": ctrlCtx.Options.Name,
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:       "https",
							Port:       443,
							Protocol:   "TCP",
							TargetPort: intstr.FromInt(8443),
						},
						{
							Name:       "kubelet",
							Port:       10250,
							Protocol:   "TCP",
							TargetPort: intstr.FromInt(8443),
						},
					},
					Selector: map[string]string{
						"app":     "vcluster",
						"release": ctrlCtx.Options.Name,
					},
				},
			}
			err = ctrlCtx.LocalManager.GetClient().Create(ctx, hostMetricsProxySvc)
			if err != nil {
				if !kerrors.IsAlreadyExists(err) {
					klog.Errorf("error create host metrics proxy service %v", err)
					return false, nil
				}
			}
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
