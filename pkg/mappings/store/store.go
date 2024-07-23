package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const GarbageCollectionInterval = time.Minute * 5

func NewStore(ctx context.Context, cachedVirtualClient, cachedHostClient client.Client, backend Backend) (synccontext.MappingsStore, error) {
	store := &Store{
		backend: backend,

		cachedVirtualClient: cachedVirtualClient,
		cachedHostClient:    cachedHostClient,

		mappings: make(map[synccontext.NameMapping]*Mapping),
		
		hostToVirtualName:         make(map[synccontext.Object]lookupName),
		virtualToHostName:         make(map[synccontext.Object]lookupName),
		hostToVirtualLabel:        make(map[string]lookupLabel),
		virtualToHostLabel:        make(map[string]lookupLabel),
		hostToVirtualLabelCluster: make(map[string]lookupLabel),
		virtualToHostLabelCluster: make(map[string]lookupLabel),
	}

	// retrieve initial mappings from backend
	err := store.start(ctx)
	if err != nil {
		return nil, fmt.Errorf("start store: %w", err)
	}

	return store, nil
}

type Store struct {
	m sync.RWMutex

	cachedVirtualClient client.Client
	cachedHostClient    client.Client

	backend  Backend
	mappings map[synccontext.NameMapping]*Mapping

	// maps Object -> Object
	hostToVirtualName map[synccontext.Object]lookupName
	virtualToHostName map[synccontext.Object]lookupName

	// maps Label -> Label
	hostToVirtualLabel map[string]lookupLabel
	virtualToHostLabel map[string]lookupLabel

	// maps Label -> Label
	hostToVirtualLabelCluster map[string]lookupLabel
	virtualToHostLabelCluster map[string]lookupLabel
}

type lookupName struct {
	Object synccontext.Object

	Mappings []*Mapping
}

type lookupLabel struct {
	Label string

	Mappings []*Mapping
}

func (s *Store) dumpMappings(ctx context.Context) {
	s.m.RLock()
	defer s.m.RUnlock()

	klog.FromContext(ctx).Info("Dump mappings")
	for mapping, refMapping := range s.mappings {
		klog.FromContext(ctx).Info("Mapping", "host", mapping.Host().String(), "virtual", mapping.Virtual().String(), "references", len(refMapping.References), "labels", len(refMapping.Labels), "labelsCluster", len(refMapping.LabelsCluster))
	}
}

func (s *Store) StartGarbageCollection(ctx context.Context) {
	go func() {
		wait.Until(func() {
			s.garbageCollectMappings(ctx)
		}, GarbageCollectionInterval, ctx.Done())
	}()
}

func (s *Store) garbageCollectMappings(ctx context.Context) {
	s.m.Lock()
	defer s.m.Unlock()

	for _, mapping := range s.mappings {
		err := s.garbageCollectMapping(ctx, mapping)
		if err != nil {
			klog.FromContext(ctx).Error(err, "Garbage collect mapping", "mapping", mapping.String())
		}
	}
}

func (s *Store) garbageCollectMapping(ctx context.Context, mapping *Mapping) error {
	// build the object we can query
	obj, err := scheme.Scheme.New(mapping.GroupVersionKind)
	if err != nil {
		if !runtime.IsNotRegisteredError(err) {
			return fmt.Errorf("create object: %w", err)
		}

		unstructuredObj := &unstructured.Unstructured{}
		unstructuredObj.SetKind(mapping.GroupVersionKind.Kind)
		unstructuredObj.SetAPIVersion(mapping.GroupVersionKind.GroupVersion().String())

		obj = unstructuredObj
	}

	// check if virtual object exists
	virtualExists := true
	err = s.cachedVirtualClient.Get(ctx, types.NamespacedName{Name: mapping.VirtualName.Name, Namespace: mapping.VirtualName.Namespace}, obj.DeepCopyObject().(client.Object))
	if err != nil {
		if !kerrors.IsNotFound(err) {
			// TODO: filter out other allowed errors here could be Forbidden, Type not found etc.
			klog.FromContext(ctx).Info("Error retrieving virtual object", "virtualObject", mapping.Virtual().String())
		}

		virtualExists = false
	}

	// check if host object exists
	hostExists := true
	err = s.cachedVirtualClient.Get(ctx, types.NamespacedName{Name: mapping.HostName.Name, Namespace: mapping.HostName.Namespace}, obj.DeepCopyObject().(client.Object))
	if err != nil {
		if !kerrors.IsNotFound(err) {
			// TODO: filter out other allowed errors here could be Forbidden, Type not found etc.
			klog.FromContext(ctx).Info("Error retrieving host object", "hostObject", mapping.Host().String())
		}

		hostExists = false
	}

	// remove mapping if both objects are not found anymore
	if virtualExists || hostExists {
		return nil
	}

	// remove mapping from backend
	err = s.backend.Delete(ctx, mapping)
	if err != nil {
		return fmt.Errorf("remove mapping from backend: %w", err)
	}

	klog.FromContext(ctx).Info("Remove mapping as both virtual and host were not found", "mapping", mapping.String())
	s.removeMapping(mapping)
	return nil
}

func (s *Store) start(ctx context.Context) error {
	go func() {
		wait.Until(func() {
			s.dumpMappings(ctx)
		}, time.Second*10, ctx.Done())
	}()

	s.m.Lock()
	defer s.m.Unlock()

	mappings, err := s.backend.List(ctx)
	if err != nil {
		return fmt.Errorf("list mappings: %w", err)
	}

	for _, mapping := range mappings {
		oldMapping, ok := s.mappings[mapping.NameMapping]
		if ok {
			s.removeMapping(oldMapping)
		}

		s.addMapping(mapping)
	}

	return nil
}

func (s *Store) HostToVirtualLabel(ctx context.Context, pLabel string) (string, bool) {
	s.m.RLock()
	defer s.m.RUnlock()

	vObjLookup, ok := s.hostToVirtualLabel[pLabel]
	if ok {
		klog.FromContext(ctx).V(1).Info("Found host label mapping in store", "fromHost", pLabel, "toVirtual", vObjLookup.Label)
	}
	return vObjLookup.Label, ok
}

func (s *Store) VirtualToHostLabel(ctx context.Context, vLabel string) (string, bool) {
	s.m.RLock()
	defer s.m.RUnlock()

	pObjLookup, ok := s.virtualToHostLabel[vLabel]
	if ok {
		klog.FromContext(ctx).V(1).Info("Found virtual label mapping in store", "fromHost", vLabel, "toVirtual", pObjLookup.Label)
	}
	return pObjLookup.Label, ok
}

func (s *Store) HostToVirtualLabelCluster(ctx context.Context, pLabel string) (string, bool) {
	s.m.RLock()
	defer s.m.RUnlock()

	vObjLookup, ok := s.hostToVirtualLabelCluster[pLabel]
	if ok {
		klog.FromContext(ctx).V(1).Info("Found host cluster-scoped label mapping in store", "fromHost", pLabel, "toVirtual", vObjLookup.Label)
	}
	return vObjLookup.Label, ok
}

func (s *Store) VirtualToHostLabelCluster(ctx context.Context, vLabel string) (string, bool) {
	s.m.RLock()
	defer s.m.RUnlock()

	pObjLookup, ok := s.virtualToHostLabelCluster[vLabel]
	if ok {
		klog.FromContext(ctx).V(1).Info("Found virtual cluster-scoped label mapping in store", "fromHost", vLabel, "toVirtual", pObjLookup.Label)
	}
	return pObjLookup.Label, ok
}

func (s *Store) HostToVirtualName(ctx context.Context, pObj synccontext.Object) (types.NamespacedName, bool) {
	s.m.RLock()
	defer s.m.RUnlock()

	vObjLookup, ok := s.hostToVirtualName[pObj]
	if ok {
		klog.FromContext(ctx).V(1).Info("Found host name mapping in store", "fromHost", pObj.NamespacedName.String(), "toVirtual", vObjLookup.Object.NamespacedName.String())
	}
	return vObjLookup.Object.NamespacedName, ok
}

func (s *Store) VirtualToHostName(ctx context.Context, vObj synccontext.Object) (types.NamespacedName, bool) {
	s.m.RLock()
	defer s.m.RUnlock()

	pObjLookup, ok := s.virtualToHostName[vObj]
	if ok {
		klog.FromContext(ctx).V(1).Info("Found virtual name mapping in store", "fromVirtual", vObj.NamespacedName.String(), "toHost", pObjLookup.Object.NamespacedName.String())
	}
	return pObjLookup.Object.NamespacedName, ok
}

func (s *Store) RecordReference(ctx context.Context, nameMapping, belongsTo synccontext.NameMapping) error {
	// we don't record incomplete mappings
	if nameMapping.Host().Empty() || nameMapping.Virtual().Empty() {
		return nil
	}

	s.m.Lock()
	defer s.m.Unlock()

	// check if there is already a conflicting mapping
	err := s.checkNameConflict(nameMapping)
	if err != nil {
		return err
	}

	// check if there is already a mapping
	mapping, ok := s.findMapping(belongsTo)
	if !ok {
		return s.createMapping(ctx, nameMapping, belongsTo)
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
	klog.FromContext(ctx).Info("Add name mapping", "host", nameMapping.Host().String(), "virtual", nameMapping.Virtual().String(), "owner", mapping.Virtual().String())
	mapping.References = append(mapping.References, nameMapping)

	// add to lookup maps
	s.addNameToMaps(mapping, nameMapping.Virtual(), nameMapping.Host())
	return nil
}

func (s *Store) RecordLabel(ctx context.Context, labelMapping synccontext.LabelMapping, belongsTo synccontext.NameMapping) error {
	// we don't record incomplete mappings
	if labelMapping.Host == "" || labelMapping.Virtual == "" {
		return nil
	}

	s.m.Lock()
	defer s.m.Unlock()

	// check if there is already a conflicting mapping
	err := s.checkLabelConflict(labelMapping)
	if err != nil {
		return err
	}

	// check if there is already a mapping
	mapping, ok := s.findMapping(belongsTo)
	if !ok {
		return nil
	}

	// check if reference already exists
	for _, label := range mapping.Labels {
		if label.Equals(labelMapping) {
			return nil
		}
	}

	// add mapping
	mapping.changed = true
	klog.FromContext(ctx).Info("Add label mapping", "host", labelMapping.Host, "virtual", labelMapping.Virtual, "owner", mapping.Virtual().String())
	mapping.Labels = append(mapping.Labels, labelMapping)

	// add to lookup maps
	s.addLabelToMaps(mapping, labelMapping.Virtual, labelMapping.Host)
	return nil
}

func (s *Store) RecordLabelCluster(ctx context.Context, labelMapping synccontext.LabelMapping, belongsTo synccontext.NameMapping) error {
	// check if we have the owning object in the context
	belongsTo, ok := synccontext.MappingFrom(ctx)
	if !ok {
		return nil
	}

	// we don't record incomplete mappings
	if labelMapping.Host == "" || labelMapping.Virtual == "" {
		return nil
	}

	s.m.Lock()
	defer s.m.Unlock()

	// check if there is already a conflicting mapping
	err := s.checkLabelClusterConflict(labelMapping)
	if err != nil {
		return err
	}

	// check if there is already a mapping
	mapping, ok := s.findMapping(belongsTo)
	if !ok {
		return nil
	}

	// check if reference already exists
	for _, label := range mapping.LabelsCluster {
		if label.Equals(labelMapping) {
			return nil
		}
	}

	// add mapping
	mapping.changed = true
	klog.FromContext(ctx).Info("Add cluster-scoped label mapping", "host", labelMapping.Host, "virtual", labelMapping.Virtual, "owner", mapping.Virtual().String())
	mapping.LabelsCluster = append(mapping.LabelsCluster, labelMapping)

	// add to lookup maps
	s.addLabelClusterToMaps(mapping, labelMapping.Virtual, labelMapping.Host)
	return nil
}

func (s *Store) SaveMapping(ctx context.Context, nameMapping synccontext.NameMapping) error {
	// we ignore empty mappings here
	if nameMapping.Empty() {
		return nil
	}

	s.m.Lock()
	defer s.m.Unlock()

	// check if there is already a mapping
	mapping, ok := s.findMapping(nameMapping)
	if !ok {
		klog.FromContext(ctx).V(1).Info("couldn't find mapping", "mapping", nameMapping.String())
		return nil
	} else if !mapping.changed {
		return nil
	}

	// save mapping
	klog.FromContext(ctx).Info("Save mapping in store", "mapping", mapping.String())
	err := s.backend.Save(ctx, mapping)
	if err != nil {
		return fmt.Errorf("save mapping %s: %w", mapping.NameMapping.String(), err)
	}

	mapping.changed = false
	return nil
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
		vObjLookup, ok := s.hostToVirtualName[pObj]
		if !ok {
			return nil, false
		}

		vObj = vObjLookup.Object
	} else if pObj.Empty() {
		// try to find by vObj
		pObjLookup, ok := s.virtualToHostName[vObj]
		if !ok {
			return nil, false
		}

		pObj = pObjLookup.Object
	}

	// just check for the mapping
	retMapping, ok := s.mappings[synccontext.NameMapping{
		GroupVersionKind: mapping.GroupVersionKind,
		VirtualName:      vObj.NamespacedName,
		HostName:         pObj.NamespacedName,
	}]
	return retMapping, ok
}

func (s *Store) createMapping(ctx context.Context, nameMapping, belongsTo synccontext.NameMapping) error {
	// check if we should add a new mapping
	if belongsTo.Empty() {
		return nil
	}

	// check what object is empty
	pObj, vObj := belongsTo.Host(), belongsTo.Virtual()
	if vObj.Empty() || pObj.Empty() {
		// check if the name mapping matches
		if nameMapping.GroupVersionKind.String() != belongsTo.GroupVersionKind.String() {
			return nil
		}

		// try to find missing virtual or host object
		if vObj.Empty() && pObj.Equals(nameMapping.Host()) {
			vObj = nameMapping.Virtual()
		} else if pObj.Empty() && vObj.Equals(nameMapping.Virtual()) {
			pObj = nameMapping.Host()
		} else {
			return nil
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
	klog.FromContext(ctx).Info("Create name mapping", "host", newMapping.NameMapping.Host().String(), "virtual", newMapping.NameMapping.Virtual().String())
	s.addMapping(newMapping)
	return nil
}

func (s *Store) checkNameConflict(nameMapping synccontext.NameMapping) error {
	// check if the mapping is conflicting
	vName, ok := s.hostToVirtualName[nameMapping.Host()]
	if ok && !vName.Object.Equals(nameMapping.Virtual()) {
		return fmt.Errorf("there is already another name mapping %s -> %s that conflicts with %s -> %s", nameMapping.Host().String(), vName.Object.String(), nameMapping.Host().String(), nameMapping.Virtual().String())
	}

	// check the other way around
	pName, ok := s.virtualToHostName[nameMapping.Virtual()]
	if ok && !pName.Object.Equals(nameMapping.Host()) {
		return fmt.Errorf("there is already another name mapping %s -> %s that conflicts with %s -> %s", nameMapping.Virtual().String(), pName.Object.String(), nameMapping.Virtual().String(), nameMapping.Host().String())
	}

	return nil
}

func (s *Store) checkLabelConflict(labelMapping synccontext.LabelMapping) error {
	// check if the mapping is conflicting
	vLabel, ok := s.hostToVirtualLabel[labelMapping.Host]
	if ok && vLabel.Label != labelMapping.Virtual {
		return fmt.Errorf("there is already another label mapping %s -> %s that conflicts with %s -> %s", labelMapping.Host, vLabel.Label, labelMapping.Host, labelMapping.Virtual)
	}

	// check the other way around
	pLabel, ok := s.hostToVirtualLabel[labelMapping.Virtual]
	if ok && pLabel.Label != labelMapping.Host {
		return fmt.Errorf("there is already another label mapping %s -> %s that conflicts with %s -> %s", labelMapping.Virtual, pLabel.Label, labelMapping.Virtual, labelMapping.Host)
	}

	return nil
}

func (s *Store) checkLabelClusterConflict(labelMapping synccontext.LabelMapping) error {
	// check if the mapping is conflicting
	vLabel, ok := s.hostToVirtualLabelCluster[labelMapping.Host]
	if ok && vLabel.Label != labelMapping.Virtual {
		return fmt.Errorf("there is already another cluster-scoped label mapping %s -> %s that conflicts with %s -> %s", labelMapping.Host, vLabel.Label, labelMapping.Host, labelMapping.Virtual)
	}

	// check the other way around
	pLabel, ok := s.hostToVirtualLabelCluster[labelMapping.Virtual]
	if ok && pLabel.Label != labelMapping.Host {
		return fmt.Errorf("there is already another cluster-scoped label mapping %s -> %s that conflicts with %s -> %s", labelMapping.Virtual, pLabel.Label, labelMapping.Virtual, labelMapping.Host)
	}

	return nil
}

func (s *Store) removeLabelFromMaps(mapping *Mapping, vLabel, pLabel string) {
	removeMappingFromLabelMap(s.hostToVirtualLabel, mapping, pLabel)
	removeMappingFromLabelMap(s.virtualToHostLabel, mapping, vLabel)
}

func (s *Store) removeLabelClusterFromMaps(mapping *Mapping, vLabel, pLabel string) {
	removeMappingFromLabelMap(s.hostToVirtualLabelCluster, mapping, pLabel)
	removeMappingFromLabelMap(s.virtualToHostLabelCluster, mapping, vLabel)
}

func (s *Store) removeNameFromMaps(mapping *Mapping, vObj, pObj synccontext.Object) {
	removeMappingFromNameMap(s.hostToVirtualName, mapping, pObj)
	removeMappingFromNameMap(s.virtualToHostName, mapping, vObj)
}

func (s *Store) addLabelToMaps(mapping *Mapping, vLabel, pLabel string) {
	addMappingToLabelMap(s.hostToVirtualLabel, mapping, pLabel, vLabel)
	addMappingToLabelMap(s.virtualToHostLabel, mapping, vLabel, pLabel)
}

func (s *Store) addLabelClusterToMaps(mapping *Mapping, vLabel, pLabel string) {
	addMappingToLabelMap(s.hostToVirtualLabelCluster, mapping, pLabel, vLabel)
	addMappingToLabelMap(s.virtualToHostLabelCluster, mapping, vLabel, pLabel)
}

func (s *Store) addNameToMaps(mapping *Mapping, vObj, pObj synccontext.Object) {
	addMappingToNameMap(s.hostToVirtualName, mapping, pObj, vObj)
	addMappingToNameMap(s.virtualToHostName, mapping, vObj, pObj)
}

func (s *Store) addMapping(mapping *Mapping) {
	s.mappings[mapping.NameMapping] = mapping
	s.addNameToMaps(mapping, mapping.Virtual(), mapping.Host())

	// add references
	for _, reference := range mapping.References {
		s.addNameToMaps(mapping, reference.Virtual(), reference.Host())
	}

	// add labels
	for _, label := range mapping.Labels {
		s.addLabelToMaps(mapping, label.Virtual, label.Host)
	}

	// add labels cluster
	for _, label := range mapping.LabelsCluster {
		s.addLabelClusterToMaps(mapping, label.Virtual, label.Host)
	}
}

func (s *Store) removeMapping(mapping *Mapping) {
	delete(s.mappings, mapping.NameMapping)

	// delete references
	for _, reference := range mapping.References {
		s.removeNameFromMaps(mapping, reference.Virtual(), reference.Host())
	}

	// delete labels
	for _, label := range mapping.Labels {
		s.removeLabelFromMaps(mapping, label.Virtual, label.Host)
	}

	// delete labels cluster
	for _, label := range mapping.LabelsCluster {
		s.removeLabelClusterFromMaps(mapping, label.Virtual, label.Host)
	}
}
