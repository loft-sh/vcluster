package util

import (
	"encoding/json"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const AlbConditionAnnotation = "alb.ingress.kubernetes.io/conditions"
const AlbActionsAnnotation = "alb.ingress.kubernetes.io/actions"
const ConditionSuffix = "/conditions."
const ActionsSuffix = "/actions."

func getActionOrConditionValue(annotation, actionOrCondition string) string {
	i := strings.Index(annotation, actionOrCondition)
	if i > -1 {
		return annotation[i+len(actionOrCondition):]
	}
	return ""
}

func ProcessAlbAnnotations(namespace string, k string, v string) (string, string) {
	if strings.HasPrefix(k, AlbActionsAnnotation) {
		type actionPayload struct {
			Type          string `json:"type,omitempty"`
			ForwardConfig struct {
				TargetGroups                []map[string]interface{} `json:"targetGroups,omitempty"`
				TargetGroupStickinessConfig map[string]interface{}   `json:"targetGroupStickinessConfig,omitempty"`
			} `json:"forwardConfig,omitempty"`
		}
		// change k
		action := getActionOrConditionValue(k, ActionsSuffix)
		if !strings.Contains(k, "x-"+namespace+"-x") {
			k = strings.Replace(k, action, translate.Default.PhysicalName(action, namespace), 1)
		}
		// change v
		var payload *actionPayload
		err := json.Unmarshal([]byte(v), &payload)
		if err != nil {
			klog.Errorf("Could not unmarshal payload: %v", err)
		} else {
			for _, targetGroup := range payload.ForwardConfig.TargetGroups {
				if targetGroup["serviceName"] != nil {
					switch svcName := targetGroup["serviceName"].(type) {
					case string:
						if svcName != "" {
							if !strings.Contains(svcName, "x-"+namespace+"-x") {
								targetGroup["serviceName"] = translate.Default.PhysicalName(svcName, namespace)
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
			k = strings.Replace(k, condition, translate.Default.PhysicalName(condition, namespace), 1)
		}
	}
	return k, v
}

func UpdateAnnotations(vObj client.Object) client.Object {
	annotations := vObj.GetAnnotations()
	for k, v := range annotations {
		delete(annotations, k)
		k, v = ProcessAlbAnnotations(vObj.GetNamespace(), k, v)
		annotations[k] = v
	}
	vObj.SetAnnotations(annotations)
	return vObj
}
