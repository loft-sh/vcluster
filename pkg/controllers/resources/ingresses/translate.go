package ingresses

import (
	"encoding/json"
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AlbConditionAnnotation = "alb.ingress.kubernetes.io/conditions"
	AlbActionsAnnotation   = "alb.ingress.kubernetes.io/actions"
	ConditionSuffix        = "/conditions."
	ActionsSuffix          = "/actions."
)

func (s *ingressSyncer) translate(ctx *synccontext.SyncContext, vIngress *networkingv1.Ingress) (*networkingv1.Ingress, error) {
	newIngress := s.TranslateMetadata(ctx, vIngress).(*networkingv1.Ingress)
	newIngress.Spec = *translateSpec(ctx, vIngress.Namespace, &vIngress.Spec)
	newIngress.Annotations, _ = translateIngressAnnotations(ctx, newIngress.Annotations, vIngress.Namespace)
	return newIngress, nil
}

func (s *ingressSyncer) TranslateMetadata(ctx *synccontext.SyncContext, vObj client.Object) client.Object {
	ingress := vObj.(*networkingv1.Ingress).DeepCopy()
	updateAnnotations(ingress)
	return translate.HostMetadata(ctx, vObj, s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj))
}

func (s *ingressSyncer) TranslateMetadataUpdate(ctx *synccontext.SyncContext, vObj client.Object, pObj client.Object) (annotations map[string]string, labels map[string]string) {
	vIngress := vObj.(*networkingv1.Ingress).DeepCopy()
	updateAnnotations(vIngress)
	return translate.HostAnnotations(vIngress, pObj), translate.HostLabels(ctx, vIngress, pObj)
}

func (s *ingressSyncer) translateUpdate(ctx *synccontext.SyncContext, pObj, vObj *networkingv1.Ingress) {
	pObj.Spec = *translateSpec(ctx, vObj.Namespace, &vObj.Spec)

	var translatedAnnotations map[string]string
	translatedAnnotations, pObj.Labels = s.TranslateMetadataUpdate(ctx, vObj, pObj)
	translatedAnnotations, _ = translateIngressAnnotations(ctx, translatedAnnotations, vObj.Namespace)
	pObj.Annotations = translatedAnnotations
}

func translateSpec(ctx *synccontext.SyncContext, namespace string, vIngressSpec *networkingv1.IngressSpec) *networkingv1.IngressSpec {
	retSpec := vIngressSpec.DeepCopy()
	if retSpec.DefaultBackend != nil {
		if retSpec.DefaultBackend.Service != nil && retSpec.DefaultBackend.Service.Name != "" {
			retSpec.DefaultBackend.Service.Name = mappings.VirtualToHostName(ctx, retSpec.DefaultBackend.Service.Name, namespace, mappings.Services())
		}
		if retSpec.DefaultBackend.Resource != nil {
			retSpec.DefaultBackend.Resource.Name = translate.Default.HostName(retSpec.DefaultBackend.Resource.Name, namespace)
		}
	}

	for i, rule := range retSpec.Rules {
		if rule.HTTP != nil {
			for j, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil && path.Backend.Service.Name != "" {
					retSpec.Rules[i].HTTP.Paths[j].Backend.Service.Name = mappings.VirtualToHostName(ctx, retSpec.Rules[i].HTTP.Paths[j].Backend.Service.Name, namespace, mappings.Services())
				}
				if path.Backend.Resource != nil {
					retSpec.Rules[i].HTTP.Paths[j].Backend.Resource.Name = translate.Default.HostName(retSpec.Rules[i].HTTP.Paths[j].Backend.Resource.Name, namespace)
				}
			}
		}
	}

	for i, tls := range retSpec.TLS {
		if tls.SecretName != "" {
			retSpec.TLS[i].SecretName = mappings.VirtualToHostName(ctx, retSpec.TLS[i].SecretName, namespace, mappings.Secrets())
		}
	}

	return retSpec
}

func getActionOrConditionValue(annotation, actionOrCondition string) string {
	i := strings.Index(annotation, actionOrCondition)
	if i > -1 {
		return annotation[i+len(actionOrCondition):]
	}
	return ""
}

// ref https://github.com/kubernetes-sigs/aws-load-balancer-controller/blob/main/pkg/ingress/config_types.go
type actionPayload struct {
	TargetGroupARN      *string                `json:"targetGroupARN,omitempty"`
	FixedResponseConfig map[string]interface{} `json:"fixedResponseConfig,omitempty"`
	RedirectConfig      map[string]interface{} `json:"redirectConfig,omitempty"`
	Type                string                 `json:"type,omitempty"`
	ForwardConfig       struct {
		TargetGroupStickinessConfig map[string]interface{}   `json:"targetGroupStickinessConfig,omitempty"`
		TargetGroups                []map[string]interface{} `json:"targetGroups,omitempty"`
	} `json:"forwardConfig,omitempty"`
}

func processAlbAnnotations(namespace string, k string, v string) (string, string) {
	if strings.HasPrefix(k, AlbActionsAnnotation) {
		// change k
		action := getActionOrConditionValue(k, ActionsSuffix)
		if !strings.Contains(k, "x-"+namespace+"-x") {
			k = strings.Replace(k, action, translate.Default.HostName(action, namespace), 1)
		}
		// change v
		var payload *actionPayload
		err := json.Unmarshal([]byte(v), &payload)
		if err != nil {
			klog.Errorf("Could not unmarshal payload: %v", err)
		} else if payload != nil {
			for _, targetGroup := range payload.ForwardConfig.TargetGroups {
				if targetGroup["serviceName"] != nil {
					switch svcName := targetGroup["serviceName"].(type) {
					case string:
						if svcName != "" {
							if !strings.Contains(svcName, "x-"+namespace+"-x") {
								targetGroup["serviceName"] = translate.Default.HostName(svcName, namespace)
							} else {
								targetGroup["serviceName"] = svcName
							}
						}
					}
				}
			}
			marshalledPayload, err := json.Marshal(payload)
			if err != nil {
				klog.Errorf("Could not marshal payload: %v", err)
			}
			v = string(marshalledPayload)
		}
	}
	if strings.HasPrefix(k, AlbConditionAnnotation) {
		condition := getActionOrConditionValue(k, ConditionSuffix)
		if !strings.Contains(k, "x-"+namespace+"-x") {
			k = strings.Replace(k, condition, translate.Default.HostName(condition, namespace), 1)
		}
	}
	return k, v
}

func updateAnnotations(ingress *networkingv1.Ingress) {
	for k, v := range ingress.Annotations {
		delete(ingress.Annotations, k)
		k, v = processAlbAnnotations(ingress.Namespace, k, v)
		ingress.Annotations[k] = v
	}
}
