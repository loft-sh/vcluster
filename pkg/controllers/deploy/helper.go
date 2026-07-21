package deploy

import (
	"context"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/pkg/util/applier"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyGivenInitManifests(ctx context.Context, vClient client.Client, vConfig *rest.Config, rawManifests, lastAppliedManifests string) error {
	lastAppliedObjects, err := populateLastAppliedMap(lastAppliedManifests, corev1.NamespaceDefault)
	if err != nil {
		klog.Errorf("unable to parse objects from last applied manifests: %v", err)
		return errors.Wrap(err, "unable to parse last applied manifests")
	}

	objs, err := ManifestStringToUnstructuredArray(rawManifests, corev1.NamespaceDefault)
	if err != nil {
		klog.Errorf("unable to parse objects: %v", err)
		return errors.Wrap(err, "unable to parse objects")
	}
	if len(lastAppliedObjects) == 0 && len(objs) == 0 {
		return nil
	}

	// create string from objs
	processedManifests := ""
	for _, obj := range objs {
		out, err := yaml.Marshal(obj)
		if err != nil {
			return errors.Wrap(err, "marshal object")
		}

		processedManifests += "\n---\n" + string(out)
	}

	// apply objects
	if len(objs) > 0 {
		klog.Infof("got %d objs to be applied", len(objs))
		err = applier.ApplyManifest(ctx, vConfig, []byte(processedManifests))
		if err != nil {
			klog.Errorf("error applying manifests: %v", err)
			return errors.Wrap(err, "apply manifests")
		}
	}

	// register applied objects in a map
	currentlyApplyingObjects := make(UnstructuredMap)
	for _, obj := range objs {
		currentlyApplyingObjects[UnstructuredToKObject(*obj)] = obj.DeepCopy()
	}

	// delete objects that are no longer part of the current manifests
	// but exists in the last applied map and hence in the vcluster
	for key, value := range lastAppliedObjects {
		if _, ok := currentlyApplyingObjects[key]; !ok {
			// this object is not in the list of objects to be applied
			// but was in last applied configuration, hence proceed for its deletion
			klog.Infof("delete non existing init object: %v", key)
			err = vClient.Delete(ctx, value)
			if err != nil && !kerrors.IsNotFound(err) {
				klog.Errorf("unable to delete old object %s: %v", value.GetName(), err)
				return errors.Wrapf(err, "unable to delete old object: %v", value.GetName())
			}
		}
	}

	return nil
}

func populateLastAppliedMap(manifests, defaultNamespace string) (UnstructuredMap, error) {
	m := make(UnstructuredMap)
	if manifests == "" {
		return m, nil
	}

	objs, err := ManifestStringToUnstructuredArray(manifests, defaultNamespace)
	if err != nil {
		return nil, err
	}

	for _, obj := range objs {
		m[UnstructuredToKObject(*obj)] = obj.DeepCopy()
	}
	return m, nil
}
