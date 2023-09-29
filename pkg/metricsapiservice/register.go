package metricsapiservice

import (
	"context"
	"math"
	"time"

	"github.com/loft-sh/vcluster/pkg/setup/options"
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

const (
	MetricsVersion    = "v1beta1"
	MetricsAPIService = MetricsVersion + "." + metrics.GroupName // "v1beta1.metrics.k8s.io"
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

func deleteOperation(_ context.Context, client client.Client) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		err := client.Delete(ctx, &apiregistrationv1.APIService{
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

func createOperation(_ context.Context, client client.Client) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
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
			// CABundle: []byte(`-----BEGIN CERTIFICATE-----\nMIIC/jCCAeagAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl\ncm5ldGVzMB4XDTIyMDgxNjAzMDgwMFoXDTMyMDgxMzAzMDgwMFowFTETMBEGA1UE\nAxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMf1\nzJTnoUVt+OH4GG2qLFN4JlLoupTa3xFr+yhnJxf0LqVMBF6JKN/0khKEClFlO1lp\nXtWZHbz2yrQKB/3PZ7mWZiu5zcW4BxMFww6Je/2Ut5Y9KWcQgocdu9lQkhkCyPY9\no5RKVCEnuSB/rfYPD2d97Q4bNDwH6+/DT6vOE1KcNY7nzLynWcSa+xAh/ArG+PZO\noUIZ1kjDbE7NL7IJV1yWTvmVokOV9BDTll4HPctvhMblYMzZbxG6uB1SniJlxzuB\nF1+uBrVV/v5H4c4xyko5WTFwhAMe1aLM4NEpE6xCEoJmB6Qgrhun1AMukUBuJkqE\nxWykUfLkLK1lgFU2OdsCAwEAAaNZMFcwDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB\n/wQFMAMBAf8wHQYDVR0OBBYEFHwYuwZ+21NrYghdBkY0HG9CkgfmMBUGA1UdEQQO\nMAyCCmt1YmVybmV0ZXMwDQYJKoZIhvcNAQELBQADggEBALgbrX/UUXSLi/uRZ5h7\nKMluBCBFs1ATBfgzlMHqlCdYJTR/Eps3NWBy26+yC0URYIlnDHqtQs14eHPo0iJR\nrff6BTQvyS5jZqkZyvkQWjE8J9xXVJe6vew8yQbM4pgZZIXjRRBjV7Mlr6bzjY74\nxxlI1JnCP75+/3sJnQrZDy6lcg4MsacvojHYdXEgHX8MccEZ6Gt6x2++plsfmtax\nFspo3R7HuP1eM4jlZ24rRj+w2bwyTPZ22wpc6eAljrR2qjlYWHEmTMKQS+MjJk5q\nzg2frjE410c8bqLZpa61Npun/q7gpxIAXlj914DJEiv+9DotjQuFJ59mQFYrU9iA\nfVE=\n-----END CERTIFICATE-----`),
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
			return false, nil
		}

		return true, nil
	}
}

func RegisterOrDeregisterAPIService(ctx context.Context, options *options.VirtualClusterOptions, client client.Client) error {
	// check if the api service should get created
	exists := checkExistingAPIService(ctx, client)
	if options.ProxyMetricsServer {
		return applyOperation(ctx, createOperation(ctx, client))
	} else if !options.ProxyMetricsServer && exists {
		return applyOperation(ctx, deleteOperation(ctx, client))
	}

	return nil
}
