package manifests

import (
	"context"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/loft-sh/vcluster/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyGivenInitManifests(ctx context.Context,
	vClient client.Client,
	defaultNamespace,
	rawManifests,
	lastAppliedManifest string) error {
	currentlyApplyingObjects := make(util.UnstructuredMap)

	lastAppliedObjects, err := populateLastAppliedMap(lastAppliedManifest, defaultNamespace)
	if err != nil {
		klog.Errorf("unable to parse objects from last applied manifests: %v", err)
		return errors.Wrap(err, "unable to parse last applied manifests")
	}

	objs, err := util.ManifestStringToUnstructureArray(rawManifests, defaultNamespace)
	if err != nil {
		klog.Errorf("unable to parse objects: %v", err)
		return errors.Wrap(err, "unable to parse objects")
	}

	klog.Infof("got %d objs to be applied", len(objs))

	for _, obj := range objs {
		err = vClient.Create(ctx, obj, &client.CreateOptions{})
		if err != nil {
			klog.Errorf("unable to create object %s: %v", obj.GetName(), err)

			// this allows us to continue and register in applied objects
			// map in case the object is already existing
			// otherwise for all other error cases we return error
			if !kerrors.IsAlreadyExists(err) {
				return err
			}

			// we only reach here in case the object already exists
			// hence we make sure we propagate any updates to the
			// particular manifest
			err = vClient.Update(ctx, obj, &client.UpdateOptions{})
			if err != nil {
				klog.Infof("error updating an already existing object %s: %v", obj.GetName(), err)
				return err
			}

			klog.Infof("successfully updated already existing object [%s:%s:%s]", obj.GetAPIVersion(), obj.GetKind(), obj.GetName())
		}

		klog.Infof("successfully created init manifest object %s", obj.GetName())
		// register applied objects in a map
		currentlyApplyingObjects[util.UnstructuredToKObject(*obj)] = obj.DeepCopy()
	}

	// delete objects no longer in the passed in objects
	// but exists in the last applied map and hence in the vcluster
	objectsToDelete := []*unstructured.Unstructured{}

	for kobj, uobj := range lastAppliedObjects {
		klog.Infof("looking up kobj %v in currentlyApplyingObjects", kobj)
		if _, ok := currentlyApplyingObjects[kobj]; !ok {
			// this object is not in the list of objects to be applied
			// but was in last applied configuration, hence proceed for its deletion
			klog.Infof("found an old object to delete: %v", kobj)
			objectsToDelete = append(objectsToDelete, uobj)
		}
	}

	for _, obj := range objectsToDelete {
		err = vClient.Delete(ctx, obj)
		if err != nil {
			klog.Errorf("unable to delete old object %s: %v", obj.GetName(), err)
			return errors.Wrapf(err, "unable to delete old object: %v", obj)
		}

		delete(lastAppliedObjects, util.UnstructuredToKObject(*obj))
	}

	klog.Infof("successfully applied all init manifest objects")
	return nil
}

func populateLastAppliedMap(manifests, defaultNamespace string) (util.UnstructuredMap, error) {
	m := make(util.UnstructuredMap)

	objs, err := util.ManifestStringToUnstructureArray(manifests, defaultNamespace)
	if err != nil {
		return nil, err
	}

	for _, obj := range objs {
		m[util.UnstructuredToKObject(*obj)] = obj.DeepCopy()
	}

	return m, nil
}
