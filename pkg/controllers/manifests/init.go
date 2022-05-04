package manifests

import (
	"context"
	"fmt"
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
	ConfigMapDataKey        = "initmanifests.yaml"
	DefaultNamespaceIfEmpty = "default"
)

type InitManifestsConfigMapReconciler struct {
	client.Client
	Log loghelper.Logger

	VManager ctrl.Manager
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

	manifests, ok := cm.Data[ConfigMapDataKey]
	if !ok {
		r.Log.Errorf("key %s not found in the configmap", ConfigMapDataKey)
		return ctrl.Result{}, fmt.Errorf("key %s not found in the configmap", ConfigMapDataKey)
	}

	err = initmanifests.ApplyGivenInitManifests(ctx, r.VManager.GetClient(), DefaultNamespaceIfEmpty, manifests)
	if err != nil {
		r.Log.Errorf("error applying init manifests: %v", err)

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
