package manifests

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"os"
	"path"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/applier"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InitManifestRelativePath = "init/initmanifests.yaml"
	DEFAULT_NAMESPACE        = "default"
)

var (
	LAST_APPLIED_MANIFEST_HASH string

	lastAppliedObjects util.UnstructuredMap
	ErrorEmptyManifest = errors.New("empty manifest file")
)

func init() {
	lastAppliedObjects = make(util.UnstructuredMap)
}

func ApplyInitManifests(inClusterConfig *rest.Config, namespace string) error {
	vars := make(map[string]interface{})
	output, err := processManifestTemplate(vars)
	if err != nil {
		return err
	}

	return applier.ApplyManifest(inClusterConfig,
		output,
		applier.WithNamespace(namespace))
}

func ApplyGivenInitManifests(ctx context.Context, vClient client.Client, defaultNamespace, rawManifests string) error {
	currentlyApplyingObjects := make(util.UnstructuredMap)

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

	lastAppliedObjects = currentlyApplyingObjects
	klog.Infof("successfully applied all init manifest objects")
	return nil
}

func processManifestTemplate(vars map[string]interface{}) ([]byte, error) {
	manifestInputPath := path.Join(constants.ContainerManifestsFolder, InitManifestRelativePath)
	// check if the file exists, it won't in case init.manifests is null
	_, err := os.Stat(manifestInputPath)
	if err != nil {
		return nil, err
	}

	ok, err := checkIfEmpty(manifestInputPath)
	if err != nil {
		return nil, err
	}
	if ok {
		// file is empty
		updateLastApplied(manifestInputPath, []byte("---"))
		return nil, ErrorEmptyManifest
	}

	manifestTemplate, err := template.ParseFiles(manifestInputPath)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %v", manifestInputPath, err)
	}

	buf := new(bytes.Buffer)
	err = manifestTemplate.Execute(buf, vars)
	if err != nil {
		return nil, fmt.Errorf("manifestTemplate.Execute failed for manifest %s: %v", manifestInputPath, err)
	}

	// update last applied manifest and hash
	updateLastApplied(manifestInputPath, buf.Bytes())

	return buf.Bytes(), nil
}

func checkIfEmpty(path string) (bool, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	if bytes.Equal(contents, []byte("---")) {
		return true, nil
	}

	return false, nil
}

func ChangeDetected(c client.Client, namespace string) (bool, error) {
	manifestInputPath := path.Join(constants.ContainerManifestsFolder, InitManifestRelativePath)

	// check if the file exists, it won't in case init.manifests is null
	_, err := os.Stat(manifestInputPath)
	if err != nil {
		return false, err
	}

	content, err := os.ReadFile(manifestInputPath)
	if err != nil {
		return false, err
	}

	currentHash := hexHasher(content)
	klog.Info("current", string(content), LAST_APPLIED_MANIFEST_HASH)
	if currentHash != LAST_APPLIED_MANIFEST_HASH {
		return true, nil
	}

	return false, nil
}

func updateLastApplied(path string, lastAppliedManifest []byte) {
	content, _ := os.ReadFile(path)
	LAST_APPLIED_MANIFEST_HASH = hexHasher(content)

	// LAST_APPLIED_MANIFESTS = hex.EncodeToString(lastAppliedManifest[:])
}

func hexHasher(input []byte) string {
	hash := md5.Sum(input)
	return hex.EncodeToString(hash[:])
}
