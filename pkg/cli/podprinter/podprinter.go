package podprinter

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/util/stringutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PodInfoPrinter struct {
	lastMutex   sync.Mutex
	LastWarning time.Time

	shownEvents []string
}

func (u *PodInfoPrinter) PrintPodInfo(pod *corev1.Pod, log log.Logger) {
	u.lastMutex.Lock()
	defer u.lastMutex.Unlock()

	if time.Since(u.LastWarning) > time.Second*10 {
		status := find.GetPodStatus(pod)
		if status != "Running" {
			log.Warnf("vcluster is waiting, because vcluster pod %s has status: %s", pod.Name, status)
		}
		u.LastWarning = time.Now()
	}
}

func (u *PodInfoPrinter) PrintPodWarning(ctx context.Context, client *kubernetes.Clientset, pod *corev1.Pod, log log.Logger) {
	u.lastMutex.Lock()
	defer u.lastMutex.Unlock()

	if time.Since(u.LastWarning) > time.Second*10 {
		status := find.GetPodStatus(pod)
		u.shownEvents = displayWarnings(ctx, relevantObjectsFromPod(pod), pod.Namespace, client, u.shownEvents, log)
		log.Warnf("Pod %s has critical status: %s. vcluster will continue waiting, but this operation might timeout", pod.Name, status)
		u.LastWarning = time.Now()
	}
}

type relevantObject struct {
	Kind string
	Name string
	UID  string
}

func displayWarnings(ctx context.Context, relevantObjects []relevantObject, namespace string, client *kubernetes.Clientset, filter []string, log log.Logger) []string {
	events, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Debugf("Error retrieving pod events: %v", err)
		return nil
	}

	sort.Slice(events.Items, func(i, j int) bool {
		return events.Items[i].CreationTimestamp.Unix() > events.Items[j].CreationTimestamp.Unix()
	})
	for _, event := range events.Items {
		if event.Type != "Warning" {
			continue
		} else if stringutil.Contains(filter, event.Name) {
			continue
		} else if !eventMatches(&event, relevantObjects) {
			continue
		}

		log.Warnf("%s %s: %s (%s)", event.InvolvedObject.Kind, event.InvolvedObject.Name, event.Message, event.Reason)
		filter = append(filter, event.Name)
	}

	return filter
}

func relevantObjectsFromPod(pod *corev1.Pod) []relevantObject {
	// search for persistent volume claims
	objects := []relevantObject{
		{
			Kind: "Pod",
			Name: pod.Name,
			UID:  string(pod.UID),
		},
	}
	for _, v := range pod.Spec.Volumes {
		if v.PersistentVolumeClaim != nil {
			objects = append(objects, relevantObject{
				Kind: "PersistentVolumeClaim",
				Name: v.PersistentVolumeClaim.ClaimName,
			})
		}
	}
	return objects
}

func eventMatches(event *corev1.Event, objects []relevantObject) bool {
	for _, o := range objects {
		if o.Name != "" && event.InvolvedObject.Name != o.Name {
			continue
		} else if o.Kind != "" && event.InvolvedObject.Kind != o.Kind {
			continue
		} else if o.UID != "" && string(event.InvolvedObject.UID) != o.UID {
			continue
		}

		return true
	}

	return false
}
