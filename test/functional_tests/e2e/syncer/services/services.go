package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/translate"

	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/client-go/util/retry"
)

var _ = ginkgo.Describe("Services are created as expected", func() {
	var (
		f         *framework.Framework
		iteration int
		ns        string
	)

	ginkgo.JustBeforeEach(func() {
		// use default framework
		f = framework.DefaultFramework
		iteration++
		ns = fmt.Sprintf("e2e-syncer-services-%d-%s", iteration, random.String(5))

		// create test namespace
		_, err := f.VClusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.AfterEach(func() {
		// delete test namespace
		err := f.DeleteTestNamespace(ns, false)
		framework.ExpectNoError(err)
	})

	ginkgo.It("Test LoadBalancer node ports & cluster ip", func() {
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myservice-loadbalancer",
				Namespace: ns,
			},
			Spec: corev1.ServiceSpec{
				Type:                  "LoadBalancer",
				ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeLocal,
				Selector:              map[string]string{"doesnt": "matter"},
				Ports: []corev1.ServicePort{
					{
						Port: 80,
					},
				},
			},
		}

		vService, err := f.VClusterClient.CoreV1().Services(ns).Create(f.Context, service, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		err = f.WaitForService(vService.Name, vService.Namespace)
		framework.ExpectNoError(err)

		// get physical service
		pServiceName := translate.Default.HostName(nil, vService.Name, vService.Namespace)
		pService, err := f.HostClient.CoreV1().Services(pServiceName.Namespace).Get(f.Context, pServiceName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		// check node ports are the same
		framework.ExpectEqual(vService.Spec.ClusterIP, pService.Spec.ClusterIP)
		framework.ExpectEqual(vService.Spec.HealthCheckNodePort, pService.Spec.HealthCheckNodePort)
		for i := range vService.Spec.Ports {
			framework.ExpectEqual(vService.Spec.Ports[i].NodePort, pService.Spec.Ports[i].NodePort)
		}
	})

	ginkgo.It("Test Service gets created when no Kind is present in body", func() {
		service := corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myservice",
				Namespace: ns,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"doesnt": "matter"},
				Ports: []corev1.ServicePort{
					{Port: 80},
				},
			},
		}
		body, err := json.Marshal(service)
		framework.ExpectNoError(err)

		_, err = f.VClusterClient.RESTClient().Post().AbsPath("/api/v1/namespaces/" + ns + "/services").Body(body).DoRaw(f.Context)
		framework.ExpectNoError(err)

		err = f.WaitForService(service.Name, service.Namespace)
		framework.ExpectNoError(err)

		_, err = f.VClusterClient.CoreV1().Services(ns).Get(f.Context, service.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		pServiceName := translate.Default.HostName(nil, service.Name, service.Namespace)
		_, err = f.HostClient.CoreV1().Services(pServiceName.Namespace).Get(f.Context, pServiceName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("Services should complete a service status lifecycle", func() {
		svcResource := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
		svcClient := f.VClusterClient.CoreV1().Services(ns)
		testSvcName := "test-service-" + utilrand.String(5)
		testSvcLabels := map[string]string{"test-service-static": "true"}
		testSvcLabelsFlat := "test-service-static=true"
		ctx := f.Context

		svcList, err := f.VClusterClient.CoreV1().Services("").List(f.Context, metav1.ListOptions{LabelSelector: testSvcLabelsFlat})
		framework.ExpectNoError(err, "failed to list Services")

		w := &cache.ListWatch{
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = testSvcLabelsFlat
				return f.VClusterClient.CoreV1().Services(ns).Watch(f.Context, options)
			},
		}

		ginkgo.By("creating a Service")
		testService := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testSvcName,
				Labels: testSvcLabels,
			},
			Spec: corev1.ServiceSpec{
				Type: "LoadBalancer",
				Ports: []corev1.ServicePort{{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       int32(80),
					TargetPort: intstr.FromInt32(80),
				}},
			},
		}

		_, err = f.VClusterClient.CoreV1().Services(ns).Create(f.Context, testService, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("watching for the Service to be added")
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		_, err = watchtools.Until(ctx, svcList.ResourceVersion, w, func(event watch.Event) (bool, error) {
			if svc, ok := event.Object.(*corev1.Service); ok {
				found := svc.ObjectMeta.Name == testService.ObjectMeta.Name &&
					svc.ObjectMeta.Namespace == ns &&
					svc.Labels["test-service-static"] == "true"
				if !found {
					f.Log.Infof("observed Service %v in namespace %v with labels: %v & ports %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Labels, svc.Spec.Ports)
					return false, nil
				}
				f.Log.Infof("Found Service %v in namespace %v with labels: %v & ports %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Labels, svc.Spec.Ports)
				return found, nil
			}
			f.Log.Infof("Observed event: %+v", event.Object)
			return false, nil
		})
		framework.ExpectNoError(err, "Failed to locate Service %v in namespace %v", testService.ObjectMeta.Name, ns)
		f.Log.Infof("Service %s created", testSvcName)

		ginkgo.By("Getting /status")
		DynamicClient, err := dynamic.NewForConfig(f.VClusterConfig)
		framework.ExpectNoError(err, "Failed to initialize the client", err)
		svcStatusUnstructured, err := DynamicClient.Resource(svcResource).Namespace(ns).Get(ctx, testSvcName, metav1.GetOptions{}, "status")
		framework.ExpectNoError(err, "Failed to fetch ServiceStatus of Service %s in namespace %s", testSvcName, ns)
		svcStatusBytes, err := json.Marshal(svcStatusUnstructured)
		framework.ExpectNoError(err, "Failed to marshal unstructured response. %v", err)

		var svcStatus corev1.Service
		err = json.Unmarshal(svcStatusBytes, &svcStatus)
		framework.ExpectNoError(err, "Failed to unmarshal JSON bytes to a Service object type")
		f.Log.Infof("Service %s has LoadBalancer: %v", testSvcName, svcStatus.Status.LoadBalancer)

		ginkgo.By("patching the ServiceStatus")
		lbStatus := corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{{IP: "203.0.113.1"}},
		}
		lbStatusJSON, err := json.Marshal(lbStatus)
		framework.ExpectNoError(err, "Failed to marshal JSON. %v", err)
		_, err = svcClient.Patch(f.Context, testSvcName, types.MergePatchType,
			[]byte(`{"metadata":{"annotations":{"patchedstatus":"true"}},"status":{"loadBalancer":`+string(lbStatusJSON)+`}}`),
			metav1.PatchOptions{}, "status")
		framework.ExpectNoError(err, "Could not patch service status", err)

		ginkgo.By("watching for the Service to be patched")
		ctx, cancel = context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		_, err = watchtools.Until(ctx, svcList.ResourceVersion, w, func(event watch.Event) (bool, error) {
			if svc, ok := event.Object.(*corev1.Service); ok {
				found := svc.ObjectMeta.Name == testService.ObjectMeta.Name &&
					svc.ObjectMeta.Namespace == ns &&
					svc.Annotations["patchedstatus"] == "true"
				if !found {
					f.Log.Infof("observed Service %v in namespace %v with annotations: %v & LoadBalancer: %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Annotations, svc.Status.LoadBalancer)
					return false, nil
				}
				f.Log.Infof("Found Service %v in namespace %v with annotations: %v & LoadBalancer: %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Annotations, svc.Status.LoadBalancer)
				return found, nil
			}
			f.Log.Infof("Observed event: %+v", event.Object)
			return false, nil
		})
		framework.ExpectNoError(err)

		ginkgo.By("updating the ServiceStatus")

		var statusToUpdate, updatedStatus *corev1.Service
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			statusToUpdate, err = svcClient.Get(ctx, testSvcName, metav1.GetOptions{})
			framework.ExpectNoError(err, "Unable to retrieve service %s", testSvcName)

			statusToUpdate.Status.Conditions = append(statusToUpdate.Status.Conditions, metav1.Condition{
				Type:    "StatusUpdate",
				Status:  metav1.ConditionTrue,
				Reason:  "E2E",
				Message: "Set from e2e test",
			})

			updatedStatus, err = svcClient.UpdateStatus(ctx, statusToUpdate, metav1.UpdateOptions{})
			return err
		})
		framework.ExpectNoError(err, "\n\n Failed to UpdateStatus. %v\n\n", err)
		f.Log.Infof("updatedStatus.Conditions: %#v", updatedStatus.Status.Conditions)

		ginkgo.By("watching for the Service to be updated")
		ctx, cancel = context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		_, err = watchtools.Until(ctx, svcList.ResourceVersion, w, func(event watch.Event) (bool, error) {
			if svc, ok := event.Object.(*corev1.Service); ok {
				found := svc.ObjectMeta.Name == testService.ObjectMeta.Name &&
					svc.ObjectMeta.Namespace == ns &&
					svc.Annotations["patchedstatus"] == "true"
				if !found {
					f.Log.Infof("Observed Service %v in namespace %v with annotations: %v & Conditions: %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Annotations, svc.Status.LoadBalancer)
					return false, nil
				}
				for _, cond := range svc.Status.Conditions {
					if cond.Type == "StatusUpdate" &&
						cond.Reason == "E2E" &&
						cond.Message == "Set from e2e test" {
						f.Log.Infof("Found Service %v in namespace %v with annotations: %v & Conditions: %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Annotations, svc.Status.Conditions)
						return found, nil
					}
				}
				f.Log.Infof("Observed Service %v in namespace %v with annotations: %v & Conditions: %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Annotations, svc.Status.LoadBalancer)
				return false, nil
			}
			f.Log.Infof("Observed event: %+v", event.Object)
			return false, nil
		})
		framework.ExpectNoError(err, "failed to locate Service %v in namespace %v", testService.ObjectMeta.Name, ns)
		f.Log.Infof("Service %s has service status updated", testSvcName)

		ginkgo.By("patching the service")
		servicePatchPayload, err := json.Marshal(corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"test-service": "patched",
				},
			},
		})

		_, err = svcClient.Patch(ctx, testSvcName, types.StrategicMergePatchType, servicePatchPayload, metav1.PatchOptions{})
		framework.ExpectNoError(err, "failed to patch service. %v", err)

		ginkgo.By("watching for the Service to be patched")
		ctx, cancel = context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		_, err = watchtools.Until(ctx, svcList.ResourceVersion, w, func(event watch.Event) (bool, error) {
			if svc, ok := event.Object.(*corev1.Service); ok {
				found := svc.ObjectMeta.Name == testService.ObjectMeta.Name &&
					svc.ObjectMeta.Namespace == ns &&
					svc.Labels["test-service"] == "patched"
				if !found {
					f.Log.Infof("observed Service %v in namespace %v with labels: %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Labels)
					return false, nil
				}
				f.Log.Infof("Found Service %v in namespace %v with labels: %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Labels)
				return found, nil
			}
			f.Log.Infof("Observed event: %+v", event.Object)
			return false, nil
		})
		framework.ExpectNoError(err, "failed to locate Service %v in namespace %v", testService.ObjectMeta.Name, ns)
		f.Log.Infof("Service %s patched", testSvcName)

		// Delete service
		err = f.VClusterClient.CoreV1().Services(ns).Delete(f.Context, testSvcName, metav1.DeleteOptions{})
		framework.ExpectNoError(err, "failed to delete the Service. %v", err)

		ctx, cancel = context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		_, err = watchtools.Until(ctx, svcList.ResourceVersion, w, func(event watch.Event) (bool, error) {
			switch event.Type {
			case watch.Deleted:
				if svc, ok := event.Object.(*corev1.Service); ok {
					found := svc.ObjectMeta.Name == testService.ObjectMeta.Name &&
						svc.ObjectMeta.Namespace == ns &&
						svc.Labels["test-service-static"] == "true"
					if !found {
						f.Log.Infof("observed Service %v in namespace %v with labels: %v & annotations: %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Labels, svc.Annotations)
						return false, nil
					}
					f.Log.Infof("Found Service %v in namespace %v with labels: %v & annotations: %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Labels, svc.Annotations)
					return found, nil
				}
			default:
				f.Log.Infof("Observed event: %+v", event.Type)
			}
			return false, nil
		})
		framework.ExpectNoError(err, "failed to delete Service %v in namespace %v", testService.ObjectMeta.Name, ns)
		f.Log.Infof("Service %s deleted", testSvcName)
	})
})
