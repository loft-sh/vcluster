package namespaces

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Unsafe annotations based on the docs here:
// https://kubernetes.io/docs/reference/labels-annotations-taints/
var excludedAnnotations = []string{
	"scheduler.alpha.kubernetes.io/node-selector",
	"scheduler.alpha.kubernetes.io/defaultTolerations",
}

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &namespaceSyncer{
		Translator:                 translator.NewClusterTranslator(ctx, "namespace", &corev1.Namespace{}, NewNamespaceTranslator(), excludedAnnotations...),
		workloadServiceAccountName: ctx.Options.ServiceAccount,
	}, nil
}

type namespaceSyncer struct {
	translator.Translator
	workloadServiceAccountName string
}

var _ syncer.Syncer = &namespaceSyncer{}

func (s *namespaceSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	newNamespace := s.translate(vObj.(*corev1.Namespace))
	ctx.Log.Infof("create physical namespace %s", newNamespace.Name)
	err := ctx.PhysicalClient.Create(ctx.Context, newNamespace)
	if err != nil {
		ctx.Log.Infof("error syncing %s to physical cluster: %v", vObj.GetName(), err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, s.EnsureWorkloadServiceAccount(ctx, newNamespace.Name)
}

func (s *namespaceSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	updated := s.translateUpdate(pObj.(*corev1.Namespace), vObj.(*corev1.Namespace))
	if updated != nil {
		ctx.Log.Infof("updating physical namespace %s, because virtual namespace has changed", updated.Name)
		translator.PrintChanges(pObj, updated, ctx.Log)
		err := ctx.PhysicalClient.Update(ctx.Context, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, s.EnsureWorkloadServiceAccount(ctx, updated.Name)
}

func (s *namespaceSyncer) EnsureWorkloadServiceAccount(ctx *synccontext.SyncContext, pNamespace string) error {
	svc := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pNamespace,
			Name:      s.workloadServiceAccountName,
		},
	}
	_, err := controllerutil.CreateOrPatch(ctx.Context, ctx.PhysicalClient, svc, func() error {

		return nil
	})
	return err
}

func NewNamespaceTranslator() translate.PhysicalNameTranslator {
	return func(vName string, _ client.Object) string {
		return translate.Default.PhysicalNamespace(vName)
	}
}
