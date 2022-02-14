package podsecurity

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/pod-security-admission/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodSecurityReconciler struct {
	client.Client
	PodSecurityStandard string
	Log                 loghelper.Logger
}

func (r *PodSecurityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	client := r.Client
	ns := &corev1.Namespace{}
	err := client.Get(ctx, req.NamespacedName, ns)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: time.Second}, err
	}

	labels := ns.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	if v, ok := labels[api.EnforceLevelLabel]; !ok || v != r.PodSecurityStandard {
		labels[api.EnforceLevelLabel] = r.PodSecurityStandard
		labels[api.EnforceVersionLabel] = api.VersionLatest
		labels[api.WarnLevelLabel] = r.PodSecurityStandard
		labels[api.WarnVersionLabel] = api.VersionLatest
		ns.SetLabels(labels)
		err = client.Update(ctx, ns)
		if err != nil {
			return ctrl.Result{RequeueAfter: time.Second}, err
		}
		r.Log.Infof(`enforcing pod security standard "%s" on namespace "%s"`, r.PodSecurityStandard, ns.Name)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager adds the controller to the manager
func (r *PodSecurityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("pod_security").
		For(&corev1.Namespace{}).
		Complete(r)
}
