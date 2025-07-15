package resources

import (
	"errors"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ForceSyncedEventNamespace = "default"

var AcceptedKinds = map[schema.GroupVersionKind]bool{
	corev1.SchemeGroupVersion.WithKind("Pod"):       true,
	corev1.SchemeGroupVersion.WithKind("Service"):   true,
	corev1.SchemeGroupVersion.WithKind("Endpoint"):  true,
	corev1.SchemeGroupVersion.WithKind("Secret"):    true,
	corev1.SchemeGroupVersion.WithKind("ConfigMap"): true,
}

func CreateEventsMapper(_ *synccontext.RegisterContext) (synccontext.Mapper, error) {
	return &eventMapper{}, nil
}

type eventMapper struct{}

func (s *eventMapper) Migrate(_ *synccontext.RegisterContext, _ synccontext.Mapper) error {
	return nil
}

func (s *eventMapper) GroupVersionKind() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Event")
}

func (s *eventMapper) VirtualToHost(_ *synccontext.SyncContext, _ types.NamespacedName, _ client.Object) types.NamespacedName {
	// we ignore virtual events here, we only react on host events and sync them to the virtual cluster
	return types.NamespacedName{}
}

func (s *eventMapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	involvedObject, err := GetInvolvedObject(ctx, pObj)
	if err != nil {
		err = IgnoreAcceptableErrors(err)
		if err != nil {
			klog.Infof("Error retrieving involved object for %s/%s: %v", req.Namespace, req.Name, err)
		} else if pObj.GetAnnotations()[constants.SyncResourceAnnotation] == "true" {
			return types.NamespacedName{
				Namespace: ForceSyncedEventNamespace,
				Name:      pObj.GetName(),
			}
		}

		return types.NamespacedName{}
	} else if involvedObject == nil {
		return types.NamespacedName{}
	}

	pEvent, ok := pObj.(*corev1.Event)
	if !ok {
		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Namespace: involvedObject.GetNamespace(),
		Name:      HostEventNameToVirtual(pEvent.GetName(), pEvent.InvolvedObject.Name, involvedObject.GetName()),
	}
}

func (s *eventMapper) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
	return s.HostToVirtual(ctx, types.NamespacedName{Namespace: pObj.GetNamespace(), Name: pObj.GetName()}, pObj).Name != "", nil
}

func HostEventNameToVirtual(hostName string, hostInvolvedObjectName, virtualInvolvedObjectName string) string {
	// replace name of object
	if strings.HasPrefix(hostName, hostInvolvedObjectName) {
		hostName = strings.Replace(hostName, hostInvolvedObjectName, virtualInvolvedObjectName, 1)
	}

	return hostName
}

var (
	ErrNilPhysicalObject = errors.New("events: nil pObject")
	ErrKindNotAccepted   = errors.New("events: kind not accpted")
	ErrNotFound          = errors.New("events: not found")
)

func IgnoreAcceptableErrors(err error) error {
	if errors.Is(err, ErrNilPhysicalObject) ||
		errors.Is(err, ErrKindNotAccepted) ||
		errors.Is(err, ErrNotFound) {
		return nil
	}

	return err
}

// GetInvolvedObject returns the related object from the vCLuster.
// Alternatively returns a ErrNilPhysicalObject, ErrKindNotAccepted or ErrNotFound.
func GetInvolvedObject(ctx *synccontext.SyncContext, pObj client.Object) (metav1.Object, error) {
	if pObj == nil {
		return nil, ErrNilPhysicalObject
	}

	pEvent, ok := pObj.(*corev1.Event)
	if !ok {
		return nil, errors.New("object is not of type event")
	}

	// check if the involved object is accepted
	gvk := pEvent.InvolvedObject.GroupVersionKind()
	if !AcceptedKinds[gvk] {
		return nil, ErrKindNotAccepted
	}

	// create new virtual object
	vInvolvedObj, err := ctx.VirtualClient.Scheme().New(gvk)
	if err != nil {
		return nil, err
	}

	// get mapper
	mapper, err := ctx.Mappings.ByGVK(gvk)
	if err != nil {
		return nil, err
	}

	// get involved object
	vName := mapper.HostToVirtual(ctx, types.NamespacedName{
		Namespace: pEvent.Namespace,
		Name:      pEvent.InvolvedObject.Name,
	}, nil)
	if vName.Name == "" {
		return nil, ErrNotFound
	}

	// get virtual object
	err = ctx.VirtualClient.Get(ctx, vName, vInvolvedObj.(client.Object))
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, err
		}

		return nil, fmt.Errorf("%w: %w", ErrNotFound, err)
	}

	// we found the related object
	m, err := meta.Accessor(vInvolvedObj)
	if err != nil {
		return nil, err
	}

	return m, nil
}
