package namespaces

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
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

const (
	VclusterNameAnnotation      = "vcluster.loft.sh/vcluster-name"
	VclusterNamespaceAnnotation = "vcluster.loft.sh/vcluster-namespace"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	namespaceLabels, err := parseNamespaceLabels(ctx.Options.NamespaceLabels)
	if err != nil {
		return nil, fmt.Errorf("invalid value of the namespace-labels flag: %w", err)
	}

	namespaceLabels[VclusterNameAnnotation] = ctx.Options.Name
	namespaceLabels[VclusterNamespaceAnnotation] = ctx.CurrentNamespace

	return &namespaceSyncer{
		Translator:                 translator.NewClusterTranslator(ctx, "namespace", &corev1.Namespace{}, NamespaceNameTranslator, excludedAnnotations...),
		workloadServiceAccountName: ctx.Options.ServiceAccount,
		namespaceLabels:            namespaceLabels,
	}, nil
}

type namespaceSyncer struct {
	translator.Translator
	workloadServiceAccountName string
	namespaceLabels            map[string]string
}

var _ syncertypes.IndicesRegisterer = &namespaceSyncer{}

func (s *namespaceSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Namespace{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
		return []string{NamespaceNameTranslator(rawObj.GetName(), rawObj)}
	})
}

var _ syncertypes.Syncer = &namespaceSyncer{}

func (s *namespaceSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	newNamespace := s.translate(ctx.Context, vObj.(*corev1.Namespace))
	ctx.Log.Infof("create physical namespace %s", newNamespace.Name)
	err := ctx.PhysicalClient.Create(ctx.Context, newNamespace)
	if err != nil {
		ctx.Log.Infof("error syncing %s to physical cluster: %v", vObj.GetName(), err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, s.EnsureWorkloadServiceAccount(ctx, newNamespace.Name)
}

func (s *namespaceSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	updated := s.translateUpdate(ctx.Context, pObj.(*corev1.Namespace), vObj.(*corev1.Namespace))
	if updated != nil {
		ctx.Log.Infof("updating physical namespace %s, because virtual namespace has changed", updated.Name)
		translator.PrintChanges(pObj, updated, ctx.Log)
		err := ctx.PhysicalClient.Update(ctx.Context, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, s.EnsureWorkloadServiceAccount(ctx, pObj.GetName())
}

func (s *namespaceSyncer) EnsureWorkloadServiceAccount(ctx *synccontext.SyncContext, pNamespace string) error {
	if s.workloadServiceAccountName == "" {
		return nil
	}

	svc := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pNamespace,
			Name:      s.workloadServiceAccountName,
		},
	}
	_, err := controllerutil.CreateOrPatch(ctx.Context, ctx.PhysicalClient, svc, func() error { return nil })
	return err
}

func NamespaceNameTranslator(vName string, _ client.Object) string {
	return translate.Default.PhysicalNamespace(vName)
}

func parseNamespaceLabels(labels []string) (map[string]string, error) {
	out := map[string]string{}
	for _, v := range labels {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("incorrect format, expected: key=value got: %s", v)
		}
		out[parts[0]] = parts[1]
	}
	errs := validation.ValidateLabels(out, field.NewPath("namespace-labels"))
	if len(errs) != 0 {
		return nil, fmt.Errorf("invalid labels: %v", errs)
	}

	return out, nil
}
