package ingresses

import (
	"encoding/json"
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/resources"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/stringutil"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

const (
	AlbConditionAnnotation = "alb.ingress.kubernetes.io/conditions"
	AlbActionsAnnotation   = "alb.ingress.kubernetes.io/actions"
	ConditionSuffix        = "/conditions."
	ActionsSuffix          = "/actions."
)

func (s *ingressSyncer) translate(ctx *synccontext.SyncContext, vIngress *networkingv1.Ingress) (*networkingv1.Ingress, error) {
	newIngress := translate.HostMetadata(vIngress, s.VirtualToHost(ctx, types.NamespacedName{Name: vIngress.Name, Namespace: vIngress.Namespace}, vIngress), s.excludedAnnotations...)
	newIngress.Spec = *translateSpec(ctx, vIngress.Namespace, &vIngress.Spec)
	newIngress.Annotations = updateAnnotations(ctx, newIngress.Annotations, vIngress.Namespace)
	return newIngress, nil
}

func (s *ingressSyncer) translateUpdate(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*networkingv1.Ingress]) {
	event.Host.Spec = *translateSpec(ctx, event.Virtual.Namespace, &event.Virtual.Spec)

	// bi-directional sync of labels & annotations
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdateFunction(
		event,
		func(key string, value interface{}) (string, interface{}) {
			// we need to ignore the rewritten annotations here
			if resources.TranslateAnnotations[key] {
				return "", nil
			}
			if strings.HasPrefix(key, AlbActionsAnnotation) || strings.HasPrefix(key, AlbConditionAnnotation) {
				return "", nil
			}
			if stringutil.Contains(s.excludedAnnotations, key) {
				return "", nil
			}
			return key, value
		},
		func(key string, value interface{}) (string, interface{}) {
			// we ignore delete values
			strValue, ok := value.(string)
			if !ok {
				return key, value
			}
			// we ignore excluded annotations
			if stringutil.Contains(s.excludedAnnotations, key) {
				return "", nil
			}

			// translate the annotation
			translatedAnnotations := updateAnnotations(ctx, map[string]string{
				key: strValue,
			}, event.Virtual.Namespace)
			for k, v := range translatedAnnotations {
				return k, v
			}
			return "", nil
		},
	)
}

func translateSpec(ctx *synccontext.SyncContext, namespace string, vIngressSpec *networkingv1.IngressSpec) *networkingv1.IngressSpec {
	retSpec := vIngressSpec.DeepCopy()
	if retSpec.DefaultBackend != nil {
		if retSpec.DefaultBackend.Service != nil && retSpec.DefaultBackend.Service.Name != "" {
			retSpec.DefaultBackend.Service.Name = mappings.VirtualToHostName(ctx, retSpec.DefaultBackend.Service.Name, namespace, mappings.Services())
		}
		if retSpec.DefaultBackend.Resource != nil {
			retSpec.DefaultBackend.Resource.Name = translate.Default.HostName(ctx, retSpec.DefaultBackend.Resource.Name, namespace).Name
		}
	}

	for i, rule := range retSpec.Rules {
		if rule.HTTP != nil {
			for j, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil && path.Backend.Service.Name != "" {
					retSpec.Rules[i].HTTP.Paths[j].Backend.Service.Name = mappings.VirtualToHostName(ctx, retSpec.Rules[i].HTTP.Paths[j].Backend.Service.Name, namespace, mappings.Services())
				}
				if path.Backend.Resource != nil {
					retSpec.Rules[i].HTTP.Paths[j].Backend.Resource.Name = translate.Default.HostName(ctx, retSpec.Rules[i].HTTP.Paths[j].Backend.Resource.Name, namespace).Name
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

func processAlbAnnotations(ctx *synccontext.SyncContext, namespace string, k string, v string) (string, string) {
	if strings.HasPrefix(k, AlbActionsAnnotation) {
		// change k
		action := getActionOrConditionValue(k, ActionsSuffix)
		if !strings.Contains(k, "x-"+namespace+"-x") {
			k = strings.Replace(k, action, translate.Default.HostName(ctx, action, namespace).Name, 1)
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
								targetGroup["serviceName"] = translate.Default.HostName(ctx, svcName, namespace).Name
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
			k = strings.Replace(k, condition, translate.Default.HostName(ctx, condition, namespace).Name, 1)
		}
	}
	return k, v
}

func updateAnnotations(ctx *synccontext.SyncContext, vAnnotations map[string]string, vNamespace string) map[string]string {
	retAnnotations := map[string]string{}
	for k, v := range vAnnotations {
		k, v = processAlbAnnotations(ctx, vNamespace, k, v)
		retAnnotations[k] = v
	}

	retAnnotations, _ = resources.TranslateIngressAnnotations(ctx, retAnnotations, vNamespace)
	return retAnnotations
}
