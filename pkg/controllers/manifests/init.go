package manifests

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/util/compress"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"sort"
	"strings"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InitManifestSuffix     = "-init-manifests"
	LastAppliedManifestKey = "vcluster.loft.sh/last-applied-init-manifests"
)

type InitManifestsConfigMapReconciler struct {
	Log loghelper.Logger

	LocalClient    client.Client
	VirtualManager ctrl.Manager
}

func (r *InitManifestsConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO: implement better filteration through predicates
	if req.Name != translate.Suffix+InitManifestSuffix {
		return ctrl.Result{}, nil
	}

	cm := &corev1.ConfigMap{}
	err := r.LocalClient.Get(ctx, req.NamespacedName, cm)
	if err != nil {
		if kerrors.IsNotFound(err) {
			r.Log.Errorf("configmap not found %v", err)
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	var cmData []string
	for _, v := range cm.Data {
		cmData = append(cmData, v)
	}

	// make array stable or otherwise order is random
	sort.Strings(cmData)
	manifests := strings.Join(cmData, "\n---\n")
	lastAppliedManifests := ""
	if cm.ObjectMeta.Annotations != nil {
		lastAppliedManifests = cm.ObjectMeta.Annotations[LastAppliedManifestKey]
		if lastAppliedManifests != "" {
			lastAppliedManifests, err = compress.Uncompress(lastAppliedManifests)
			if err != nil {
				r.Log.Errorf("error decompressing manifests: %v", err)
			}
		}
	}

	// should skip?
	if manifests == lastAppliedManifests {
		return ctrl.Result{}, nil
	}

	// apply manifests
	err = ApplyGivenInitManifests(ctx, r.VirtualManager.GetClient(), r.VirtualManager.GetConfig(), manifests, lastAppliedManifests)
	if err != nil {
		r.Log.Errorf("error applying init manifests: %v", err)
		return ctrl.Result{}, err
	}

	// apply successful, store in an annotation in the configmap itself
	compressedManifests, err := compress.Compress(manifests)
	if err != nil {
		r.Log.Errorf("error compressing manifests: %v", err)
		return ctrl.Result{}, err
	}

	// update annotation
	if cm.ObjectMeta.Annotations == nil {
		cm.ObjectMeta.Annotations = map[string]string{}
	}
	cm.ObjectMeta.Annotations[LastAppliedManifestKey] = compressedManifests
	err = r.LocalClient.Update(ctx, cm, &client.UpdateOptions{})
	if err != nil {
		r.Log.Errorf("error updating config map with last applied annotation: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *InitManifestsConfigMapReconciler) SetupWithManager(hostMgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(hostMgr).
		Named("init_manifests").
		For(&corev1.ConfigMap{}).
		Complete(r)
}
