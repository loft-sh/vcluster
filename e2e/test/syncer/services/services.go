package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/e2e/framework"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
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
		ns = fmt.Sprintf("e2e-syncer-services-%d", iteration)
		// execute cleanup in case previous e2e test were terminated prematurely
		err := f.DeleteTestNamespace(ns, true)
		framework.ExpectNoError(err)

		// create test namespace
		_, err = f.VclusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.AfterEach(func() {
		// delete test namespace
		err := f.DeleteTestNamespace(ns, false)
		framework.ExpectNoError(err)
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

		_, err = f.VclusterClient.RESTClient().Post().AbsPath("/api/v1/namespaces/" + ns + "/services").Body(body).DoRaw(f.Context)
		framework.ExpectNoError(err)

		_, err = f.VclusterClient.CoreV1().Services(ns).Get(f.Context, service.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		_, err = f.HostClient.CoreV1().Services(f.VclusterNamespace).Get(f.Context, translate.ObjectPhysicalName(&service), metav1.GetOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("Services should complete a service status lifecycle", func() {
		svcResource := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
		svcClient := f.VclusterClient.CoreV1().Services(ns)
		testSvcName := "test-service-" + utilrand.String(5)
		testSvcLabels := map[string]string{"test-service-static": "true"}
		testSvcLabelsFlat := "test-service-static=true"

		svcList, err := f.VclusterClient.CoreV1().Services("").List(f.Context, metav1.ListOptions{LabelSelector: testSvcLabelsFlat})
		framework.ExpectNoError(err, "failed to list Services")

		w := &cache.ListWatch{
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = testSvcLabelsFlat
				return f.VclusterClient.CoreV1().Services(ns).Watch(f.Context, options)
			},
		}

		ginkgo.By("creating a Service")
		testService := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testSvcName,
				Labels: testSvcLabels,
			},
			Spec: corev1.ServiceSpec{
				Type: "ClusterIP",
				Ports: []corev1.ServicePort{{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       int32(80),
					TargetPort: intstr.FromInt(80),
				}},
			},
		}

		_, err = f.VclusterClient.CoreV1().Services(ns).Create(f.Context, testService, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("watching for the Service to be added")
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
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
		DynamicClient, err := dynamic.NewForConfig(f.VclusterConfig)
		framework.ExpectNoError(err, "Failed to initialize the client", err)
		svcStatusUnstructured, err := DynamicClient.Resource(svcResource).Namespace(ns).Get(context.TODO(), testSvcName, metav1.GetOptions{}, "status")
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
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Minute)
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
			statusToUpdate, err = svcClient.Get(context.TODO(), testSvcName, metav1.GetOptions{})
			framework.ExpectNoError(err, "Unable to retrieve service %s", testSvcName)

			statusToUpdate.Status.Conditions = append(statusToUpdate.Status.Conditions, metav1.Condition{
				Type:    "StatusUpdate",
				Status:  metav1.ConditionTrue,
				Reason:  "E2E",
				Message: "Set from e2e test",
			})

			updatedStatus, err = svcClient.UpdateStatus(context.TODO(), statusToUpdate, metav1.UpdateOptions{})
			return err
		})
		framework.ExpectNoError(err, "\n\n Failed to UpdateStatus. %v\n\n", err)
		f.Log.Infof("updatedStatus.Conditions: %#v", updatedStatus.Status.Conditions)

		ginkgo.By("watching for the Service to be updated")
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Minute)
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
					} else {
						f.Log.Infof("Observed Service %v in namespace %v with annotations: %v & Conditions: %v", svc.ObjectMeta.Name, svc.ObjectMeta.Namespace, svc.Annotations, svc.Status.LoadBalancer)
						return false, nil
					}
				}
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

		_, err = svcClient.Patch(context.TODO(), testSvcName, types.StrategicMergePatchType, []byte(servicePatchPayload), metav1.PatchOptions{})
		framework.ExpectNoError(err, "failed to patch service. %v", err)

		ginkgo.By("watching for the Service to be patched")
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Minute)
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
		err = f.VclusterClient.CoreV1().Services(ns).Delete(f.Context, testSvcName, metav1.DeleteOptions{})
		framework.ExpectNoError(err, "failed to delete the Service. %v", err)

		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Minute)
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

	ginkgo.It("should have Endpoints and EndpointSlices pointing to API Server", func() {
		namespace := "default"
		name := "kubernetes"
		// verify "kubernetes.default" service exist
		_, err := f.VclusterClient.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		framework.ExpectNoError(err, "error obtaining API server \"kubernetes\" Service resource on \"default\" namespace")

		// verify Endpoints for the API servers exist
		endpoints, err := f.VclusterClient.CoreV1().Endpoints(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		framework.ExpectNoError(err, "error obtaining API server \"kubernetes\" Endpoint resource on \"default\" namespace")
		if len(endpoints.Subsets) == 0 {
			f.Log.Failf("Expected at least 1 subset in endpoints, got %d: %#v", len(endpoints.Subsets), endpoints.Subsets)
			ginkgo.Fail("Expected at least 1 subset in endpoints")
		}
		// verify EndpointSlices for the API servers exist
		endpointSliceList, err := f.VclusterClient.DiscoveryV1().EndpointSlices(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: "kubernetes.io/service-name=" + name,
		})
		framework.ExpectNoError(err, "error obtaining API server \"kubernetes\" EndpointSlice resource on \"default\" namespace")
		if len(endpointSliceList.Items) == 0 {
			f.Log.Failf("Expected at least 1 EndpointSlice, got %d: %#v", len(endpoints.Subsets), endpoints.Subsets)
			ginkgo.Fail("Expected at least 1 subset in endpoints")
		}

		if !endpointSlicesEqual(f, endpoints, endpointSliceList) {
			f.Log.Failf("Expected EndpointSlice to have same addresses and port as Endpoints, got %#v: %#v", endpoints, endpointSliceList)
			ginkgo.Fail("Expected EndpointSlice to have same addresses and port as Endpoints")
		}

	})
})

// endpointSlicesEqual compare if the Endpoint and the EndpointSliceList contains the same endpoints values
// as in addresses and ports, considering Ready and Unready addresses
func endpointSlicesEqual(f *framework.Framework, endpoints *v1.Endpoints, endpointSliceList *discoveryv1.EndpointSliceList) bool {
	// get the apiserver endpoint addresses
	epAddresses := sets.NewString()
	epPorts := sets.NewInt32()
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			epAddresses.Insert(addr.IP)
		}
		for _, addr := range subset.NotReadyAddresses {
			epAddresses.Insert(addr.IP)
		}
		for _, port := range subset.Ports {
			epPorts.Insert(port.Port)
		}
	}
	f.Log.Infof("Endpoints addresses: %v , ports: %v", epAddresses.List(), epPorts.List())

	addrType := discoveryv1.AddressTypeIPv4

	// get the apiserver addresses from the endpoint slice list
	sliceAddresses := sets.NewString()
	slicePorts := sets.NewInt32()
	for _, slice := range endpointSliceList.Items {
		if slice.AddressType != addrType {
			f.Log.Infof("Skipping slice %s: wanted %s family, got %s", slice.Name, addrType, slice.AddressType)
			continue
		}
		for _, s := range slice.Endpoints {
			sliceAddresses.Insert(s.Addresses...)
		}
		for _, ports := range slice.Ports {
			if ports.Port != nil {
				slicePorts.Insert(*ports.Port)
			}
		}
	}

	f.Log.Infof("EndpointSlices addresses: %v , ports: %v", sliceAddresses.List(), slicePorts.List())
	if sliceAddresses.Equal(epAddresses) && slicePorts.Equal(epPorts) {
		return true
	}
	return false
}
