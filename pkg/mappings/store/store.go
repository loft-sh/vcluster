package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const GarbageCollectionInterval = time.Minute * 3

type VerifyMapping func(mapping synccontext.NameMapping) bool

func NewStore(ctx context.Context, cachedVirtualClient, cachedHostClient client.Client, backend Backend) (synccontext.MappingsStore, error) {
	return NewStoreWithVerifyMapping(ctx, cachedVirtualClient, cachedHostClient, backend, nil)
}

func NewStoreWithVerifyMapping(ctx context.Context, cachedVirtualClient, cachedHostClient client.Client, backend Backend, verifyMapping VerifyMapping) (synccontext.MappingsStore, error) {
	store := &Store{
		backend: backend,

		verifyMapping: verifyMapping,

		sender: uuid.NewString(),

		cachedVirtualClient: cachedVirtualClient,
		cachedHostClient:    cachedHostClient,

		mappings: &TypedSyncMap[synccontext.NameMapping, *Mapping]{},

		hostToVirtualName: &TypedSyncMap[synccontext.Object, lookupName]{},
		virtualToHostName: &TypedSyncMap[synccontext.Object, lookupName]{},

		watches: &TypedSyncMap[schema.GroupVersionKind, []*watcher]{},
	}

	// retrieve initial mappings from backend
	err := store.start(ctx)
	if err != nil {
		return nil, fmt.Errorf("start store: %w", err)
	}

	return store, nil
}

type TypedSyncMap[K comparable, V any] struct {
	m sync.Map
}

func (t *TypedSyncMap[K, V]) Store(key K, value V) {
	t.m.Store(key, value)
}

func (t *TypedSyncMap[K, V]) Load(key K) (V, bool) {
	value, ok := t.m.Load(key)
	if ok {
		return value.(V), ok
	}
	var zero V
	return zero, ok
}

func (t *TypedSyncMap[K, V]) Delete(key K) {
	t.m.Delete(key)
}

func (t *TypedSyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	actualInterface, loaded := t.m.LoadOrStore(key, value)
	return actualInterface.(V), loaded
}

func (t *TypedSyncMap[K, V]) Range(f func(key K, value V) bool) {
	t.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

type Store struct {
	//m sync.RWMutex

	verifyMapping VerifyMapping

	sender string

	cachedVirtualClient client.Client
	cachedHostClient    client.Client

	backend Backend

	mappings *TypedSyncMap[synccontext.NameMapping, *Mapping]

	// maps Object -> Object
	hostToVirtualName *TypedSyncMap[synccontext.Object, lookupName]
	virtualToHostName *TypedSyncMap[synccontext.Object, lookupName]

	watches *TypedSyncMap[schema.GroupVersionKind, []*watcher]
}

type lookupName struct {
	Object synccontext.Object

	Mappings []*Mapping
}

func (s *Store) Watch(gvk schema.GroupVersionKind, addQueueFn synccontext.AddQueueFunc) source.Source {
	w := &watcher{
		addQueueFn: addQueueFn,
	}

	lw, _ := s.watches.Load(gvk)
	s.watches.Store(gvk, append(lw, w))
	return w
}

func (s *Store) StartGarbageCollection(ctx context.Context) {
	go func() {
		wait.Until(func() {
			s.garbageCollectMappings(ctx)
		}, GarbageCollectionInterval, ctx.Done())
	}()
}

func (s *Store) garbageCollectMappings(ctx context.Context) {
	startTime := time.Now()
	klog.FromContext(ctx).V(1).Info("Start mappings garbage collection")
	defer func() {
		klog.FromContext(ctx).V(1).Info("Garbage collection done", "took", time.Since(startTime).String())
	}()

	s.mappings.Range(func(key synccontext.NameMapping, mapping *Mapping) bool {
		err := s.garbageCollectMapping(ctx, mapping)
		if err != nil {
			klog.FromContext(ctx).Error(err, "Garbage collect mapping", "mapping", mapping.String())
		}

		return true
	})
}

func (s *Store) garbageCollectMapping(ctx context.Context, mapping *Mapping) error {
	// check if object exists
	exists, err := s.objectExists(ctx, mapping.NameMapping)
	if err != nil {
		return err
	} else if exists {
		return nil
	}

	// delete the mapping
	err = s.deleteMapping(ctx, mapping)
	if err != nil {
		return err
	}

	klog.FromContext(ctx).Info("Remove mapping as both virtual and host were not found", "mapping", mapping.String())
	return nil
}

func (s *Store) deleteMapping(ctx context.Context, mapping *Mapping) error {
	// set sender
	mapping.Sender = s.sender

	// remove mapping from backend
	err := s.backend.Delete(ctx, mapping)
	if err != nil {
		return fmt.Errorf("remove mapping from backend: %w", err)
	}

	s.removeMapping(mapping)
	return nil
}

func (s *Store) objectExists(ctx context.Context, nameMapping synccontext.NameMapping) (bool, error) {
	// build the object we can query
	obj, err := scheme.Scheme.New(nameMapping.GroupVersionKind)
	if err != nil {
		if !runtime.IsNotRegisteredError(err) {
			return false, fmt.Errorf("create object: %w", err)
		}

		obj = &unstructured.Unstructured{}
	}

	// set kind & apiVersion if unstructured
	uObject, ok := obj.(*unstructured.Unstructured)
	if ok {
		uObject.SetGroupVersionKind(nameMapping.GroupVersionKind)
	}

	// check if virtual object exists
	err = s.cachedVirtualClient.Get(ctx, nameMapping.VirtualName, obj.DeepCopyObject().(client.Object))
	if err == nil {
		return true, nil
	} else if !kerrors.IsNotFound(err) {
		// TODO: filter out other allowed errors here could be Forbidden, Type not found etc.
		klog.FromContext(ctx).Info("Error retrieving virtual object", "virtualObject", nameMapping.Virtual().String())
	}

	// check if host object exists
	err = s.cachedHostClient.Get(ctx, nameMapping.HostName, obj.DeepCopyObject().(client.Object))
	if err == nil {
		return true, nil
	} else if !kerrors.IsNotFound(err) {
		// TODO: filter out other allowed errors here could be Forbidden, Type not found etc.
		klog.FromContext(ctx).Info("Error retrieving host object", "hostObject", nameMapping.Host().String())
	}

	return false, nil
}

func (s *Store) start(ctx context.Context) error {
	mappings, err := s.backend.List(ctx)
	if err != nil {
		return fmt.Errorf("list mappings: %w", err)
	}

	for _, mapping := range mappings {
		// verify mapping if needed
		if s.verifyMapping != nil && !s.verifyMapping(mapping.NameMapping) {
			continue
		}

		oldMapping, ok := s.mappings.Load(mapping.NameMapping)
		if ok {
			s.removeMapping(oldMapping)
		}

		klog.FromContext(ctx).V(1).Info("Add mapping", "mapping", mapping.String())
		s.addMapping(mapping)
	}

	go func() {
		w := s.backend.Watch(ctx)
		for {
			select {
			case watchEvent, ok := <-w:
				if !ok {
					klog.FromContext(ctx).Info("mapping store watch has ended, restarting...")
					w = s.backend.Watch(ctx)
					continue
				}

				s.handleEvent(ctx, watchEvent)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (s *Store) handleEvent(ctx context.Context, watchEvent BackendWatchResponse) {
	if watchEvent.Err != nil {
		klog.FromContext(ctx).Error(watchEvent.Err, "watch err in mappings store")
		return
	}

	for _, event := range watchEvent.Events {
		// ignore events sent by us
		if event.Mapping.Sender == s.sender {
			continue
		}

		// verify mapping if needed
		if s.verifyMapping != nil && !s.verifyMapping(event.Mapping.NameMapping) {
			continue
		}

		klog.FromContext(ctx).V(1).Info("mapping store received event", "type", event.Type, "mapping", event.Mapping.String())

		// remove mapping in any case
		oldMapping, ok := s.mappings.Load(event.Mapping.NameMapping)
		if ok {
			s.removeMapping(oldMapping)
		}

		// re-add mapping if its an update
		if event.Type == BackendWatchEventTypeUpdate {
			s.addMapping(event.Mapping)
		}
	}
}

func (s *Store) HasHostObject(ctx context.Context, pObj synccontext.Object) bool {
	_, ok := s.HostToVirtualName(ctx, pObj)
	return ok
}

func (s *Store) HostToVirtualName(_ context.Context, pObj synccontext.Object) (types.NamespacedName, bool) {
	vObjLookup, ok := s.hostToVirtualName.Load(pObj)
	return vObjLookup.Object.NamespacedName, ok
}

func (s *Store) HasVirtualObject(ctx context.Context, vObj synccontext.Object) bool {
	_, ok := s.VirtualToHostName(ctx, vObj)
	return ok
}

func (s *Store) VirtualToHostName(_ context.Context, vObj synccontext.Object) (types.NamespacedName, bool) {
	pObjLookup, ok := s.virtualToHostName.Load(vObj)
	return pObjLookup.Object.NamespacedName, ok
}

func (s *Store) DeleteReferenceAndSave(ctx context.Context, nameMapping, belongsTo synccontext.NameMapping) error {
	err := s.DeleteReference(ctx, nameMapping, belongsTo)
	if err != nil {
		return err
	}

	return s.SaveMapping(ctx, belongsTo)
}

func (s *Store) DeleteReference(ctx context.Context, nameMapping, belongsTo synccontext.NameMapping) error {
	// we don't record incomplete mappings
	if nameMapping.Host().Empty() || nameMapping.Virtual().Empty() {
		return nil
	}

	// check if there is already a mapping
	mapping, ok := s.findMapping(belongsTo)
	if !ok {
		return nil
	}

	// check if reference already exists
	newReferences := make([]synccontext.NameMapping, 0, len(mapping.References)-1)
	for _, reference := range mapping.References {
		if reference.Equals(nameMapping) {
			continue
		}

		newReferences = append(newReferences, reference)
	}

	// check if we found the reference
	if len(newReferences) == len(mapping.References) {
		return nil
	}

	// signal mapping was changed
	mapping.References = newReferences
	mapping.changed = true
	klog.FromContext(ctx).Info("Delete mapping reference", "host", nameMapping.Host().String(), "virtual", nameMapping.Virtual().String(), "owner", mapping.Virtual().String())

	// add to lookup maps
	s.removeNameFromMaps(mapping, nameMapping.Virtual(), nameMapping.Host())
	load, _ := s.watches.Load(nameMapping.GroupVersionKind)
	dispatchAll(load, nameMapping)
	return nil
}

func (s *Store) AddReferenceAndSave(ctx context.Context, nameMapping, belongsTo synccontext.NameMapping) error {
	err := s.AddReference(ctx, nameMapping, belongsTo)
	if err != nil {
		return err
	}

	return s.SaveMapping(ctx, belongsTo)
}

func (s *Store) AddReference(ctx context.Context, nameMapping, belongsTo synccontext.NameMapping) error {
	// we don't record incomplete mappings
	if nameMapping.Host().Empty() || nameMapping.Virtual().Empty() {
		return nil
	}

	// check if there is already a conflicting mapping
	err := s.checkNameConflict(nameMapping)
	if err != nil {
		return err
	}

	// verify mapping if needed
	if s.verifyMapping != nil && !s.verifyMapping(nameMapping) {
		return nil
	}

	// check if there is already a mapping
	mapping, ok := s.findMapping(belongsTo)
	if !ok {
		s.createMapping(ctx, nameMapping, belongsTo)
		return nil
	}

	// check if we need to add mapping
	if mapping.NameMapping.Equals(nameMapping) {
		return nil
	}

	// check if reference already exists
	for _, reference := range mapping.References {
		if reference.Equals(nameMapping) {
			return nil
		}
	}

	// add mapping
	mapping.changed = true
	klog.FromContext(ctx).Info("Add mapping reference", "host", nameMapping.Host().String(), "virtual", nameMapping.Virtual().String(), "owner", mapping.Virtual().String())
	mapping.References = append(mapping.References, nameMapping)

	// add to lookup maps
	s.addNameToMaps(mapping, nameMapping.Virtual(), nameMapping.Host())
	lw, _ := s.watches.Load(nameMapping.GroupVersionKind)
	dispatchAll(lw, nameMapping)
	return nil
}

func (s *Store) SaveMapping(ctx context.Context, nameMapping synccontext.NameMapping) error {
	// we ignore empty mappings here
	if nameMapping.Empty() {
		return nil
	}

	// check if there is already a mapping
	mapping, ok := s.findMapping(nameMapping)
	if !ok {
		return nil
	} else if !mapping.changed {
		return nil
	}

	// set sender
	mapping.Sender = s.sender

	// save mapping
	klog.FromContext(ctx).Info("Save object mappings in store", "mapping", mapping.String())
	err := s.backend.Save(ctx, mapping)
	if err != nil {
		return fmt.Errorf("save mapping %s: %w", mapping.NameMapping.String(), err)
	}

	mapping.changed = false
	return nil
}

func (s *Store) DeleteMapping(ctx context.Context, nameMapping synccontext.NameMapping) error {
	// we ignore empty mappings here
	if nameMapping.Empty() {
		return nil
	}

	// check if there is already a mapping
	mapping, ok := s.findMapping(nameMapping)
	if !ok {
		return nil
	}

	// delete the mapping
	err := s.deleteMapping(ctx, mapping)
	if err != nil {
		return err
	}

	klog.FromContext(ctx).Info("Remove object mappings in store", "mapping", mapping.String())
	return nil
}

func (s *Store) ReferencesTo(ctx context.Context, vObj synccontext.Object) []synccontext.NameMapping {
	retReferences := s.referencesTo(vObj)
	klog.FromContext(ctx).V(1).Info("Found references for object", "object", vObj.String(), "references", len(retReferences))
	return retReferences
}

func (s *Store) referencesTo(vObj synccontext.Object) []synccontext.NameMapping {
	if vObj.Empty() {
		return nil
	}

	hostNameLookup, ok := s.virtualToHostName.Load(vObj)
	if !ok {
		return nil
	}

	// loop over references and exclude owner mapping
	nameMapping := synccontext.NameMapping{
		GroupVersionKind: vObj.GroupVersionKind,
		VirtualName:      vObj.NamespacedName,
		HostName:         hostNameLookup.Object.NamespacedName,
	}
	retReferences := []synccontext.NameMapping{}
	for _, reference := range hostNameLookup.Mappings {
		if reference.Equals(nameMapping) {
			continue
		}

		retReferences = append(retReferences, reference.NameMapping)
	}

	return retReferences
}

func (s *Store) findMapping(mapping synccontext.NameMapping) (*Mapping, bool) {
	// check if the mapping is empty
	if mapping.Empty() {
		return nil, false
	}

	// get objects
	vObj, pObj := mapping.Virtual(), mapping.Host()
	if vObj.Empty() {
		// try to find by pObj
		vObjLookup, ok := s.hostToVirtualName.Load(pObj)
		if !ok {
			return nil, false
		}

		vObj = vObjLookup.Object
	} else if pObj.Empty() {
		// try to find by vObj
		pObjLookup, ok := s.virtualToHostName.Load(vObj)
		if !ok {
			return nil, false
		}

		pObj = pObjLookup.Object
	}

	// just check for the mapping
	retMapping, ok := s.mappings.Load(synccontext.NameMapping{
		GroupVersionKind: mapping.GroupVersionKind,
		VirtualName:      vObj.NamespacedName,
		HostName:         pObj.NamespacedName,
	})
	return retMapping, ok
}

func (s *Store) createMapping(ctx context.Context, nameMapping, belongsTo synccontext.NameMapping) {
	// check if we should add a new mapping
	if belongsTo.Empty() {
		return
	}

	// check what object is empty
	pObj, vObj := belongsTo.Host(), belongsTo.Virtual()
	if vObj.Empty() || pObj.Empty() {
		// check if the name mapping matches
		if nameMapping.GroupVersionKind.String() != belongsTo.GroupVersionKind.String() {
			klog.FromContext(ctx).Info("Cannot create name mapping, because owner mapping is incomplete and does not match group version kind", "owner", belongsTo.String(), "nameMapping", nameMapping.String())
			return
		}

		// try to find missing virtual or host object
		if vObj.Empty() && pObj.Equals(nameMapping.Host()) {
			vObj = nameMapping.Virtual()
		} else if pObj.Empty() && vObj.Equals(nameMapping.Virtual()) {
			pObj = nameMapping.Host()
		} else {
			return
		}
	}

	// create new mapping
	newMapping := &Mapping{
		NameMapping: synccontext.NameMapping{
			GroupVersionKind: belongsTo.GroupVersionKind,
			VirtualName:      vObj.NamespacedName,
			HostName:         pObj.NamespacedName,
		},

		changed: true,
	}

	// add to lookup maps
	klog.FromContext(ctx).Info("Create name mapping", "host", newMapping.NameMapping.Host().String(), "virtual", newMapping.NameMapping.Virtual().String(), "nameMapping", nameMapping.String(), "belongsTo", belongsTo.String())
	s.addMapping(newMapping)
}

func (s *Store) checkNameConflict(nameMapping synccontext.NameMapping) error {
	// check if the mapping is conflicting
	vName, ok := s.hostToVirtualName.Load(nameMapping.Host())
	if ok && !vName.Object.Equals(nameMapping.Virtual()) {
		return fmt.Errorf("there is already another name mapping %s -> %s that conflicts with %s -> %s", nameMapping.Host().String(), vName.Object.String(), nameMapping.Host().String(), nameMapping.Virtual().String())
	}

	// check the other way around
	pName, ok := s.virtualToHostName.Load(nameMapping.Virtual())
	if ok && !pName.Object.Equals(nameMapping.Host()) {
		return fmt.Errorf("there is already another name mapping %s -> %s that conflicts with %s -> %s", nameMapping.Virtual().String(), pName.Object.String(), nameMapping.Virtual().String(), nameMapping.Host().String())
	}

	return nil
}

func (s *Store) removeNameFromMaps(mapping *Mapping, vObj, pObj synccontext.Object) {
	removeMappingFromNameMap(s.hostToVirtualName, mapping, pObj)
	removeMappingFromNameMap(s.virtualToHostName, mapping, vObj)
}

func (s *Store) addNameToMaps(mapping *Mapping, vObj, pObj synccontext.Object) {
	addMappingToNameMap(s.hostToVirtualName, mapping, pObj, vObj)
	addMappingToNameMap(s.virtualToHostName, mapping, vObj, pObj)
}

func (s *Store) addMapping(mapping *Mapping) {
	s.mappings.Store(mapping.NameMapping, mapping)
	s.addNameToMaps(mapping, mapping.Virtual(), mapping.Host())
	lnm, _ := s.watches.Load(mapping.GroupVersionKind)
	dispatchAll(lnm, mapping.NameMapping)

	// add references
	for _, reference := range mapping.References {
		s.addNameToMaps(mapping, reference.Virtual(), reference.Host())
		lr, _ := s.watches.Load(reference.GroupVersionKind)
		dispatchAll(lr, reference)
	}
}

func (s *Store) removeMapping(mapping *Mapping) {
	s.mappings.Delete(mapping.NameMapping)

	s.removeNameFromMaps(mapping, mapping.Virtual(), mapping.Host())
	lmw, _ := s.watches.Load(mapping.GroupVersionKind)
	dispatchAll(lmw, mapping.NameMapping)

	// delete references
	for _, reference := range mapping.References {
		s.removeNameFromMaps(mapping, reference.Virtual(), reference.Host())
		lr, _ := s.watches.Load(reference.GroupVersionKind)
		dispatchAll(lr, reference)
	}
}
