package manifests

import (
	"context"
	"strings"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	initmanifests "github.com/loft-sh/vcluster/pkg/manifests"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InitManifestSuffix      = "-init-manifests"
	DefaultNamespaceIfEmpty = corev1.NamespaceDefault
	LastAppliedManifestKey  = "vcluster.loft.sh/last-applied-init-manifests"
)

type InitManifestsConfigMapReconciler struct {
	client.Client
	Log loghelper.Logger

	VirtualManager ctrl.Manager
}

func (r *InitManifestsConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var cm corev1.ConfigMap

	err := r.Client.Get(ctx, req.NamespacedName, &cm)
	if err != nil {
		if kerrors.IsNotFound(err) {
			r.Log.Errorf("configmap not found %v", err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	// TODO: implement better filteration through predicates
	if !strings.Contains(cm.ObjectMeta.Name, InitManifestSuffix) {
		// skip
		return ctrl.Result{}, nil
	}

	var cmData []string
	for _, v := range cm.Data {
		cmData = append(cmData, v)
	}

	manifests := strings.Join(cmData, "\n---\n")

	lastAppliedManifests := cm.ObjectMeta.Annotations[LastAppliedManifestKey]
	err = initmanifests.ApplyGivenInitManifests(ctx, r.VirtualManager.GetClient(), DefaultNamespaceIfEmpty, manifests, lastAppliedManifests)
	if err != nil {
		r.Log.Errorf("error applying init manifests: %v", err)

		return ctrl.Result{}, err
	}
	// apply successful, store in an annotation in the configmap itself
	cm.ObjectMeta.Annotations[LastAppliedManifestKey] = manifests
	err = r.Client.Update(ctx, &cm, &client.UpdateOptions{})
	if err != nil {
		r.Log.Errorf("error updating config map with last applied annotation: %v", err)
		return ctrl.Result{}, err
	}

	r.Log.Infof("init configuration manifests applied successfully")
	return ctrl.Result{}, nil
}

func (r *InitManifestsConfigMapReconciler) SetupWithManager(hostMgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(hostMgr).
		Named("init_manifests").
		For(&corev1.ConfigMap{}).
		Complete(r)
}
