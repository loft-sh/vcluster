package toleration

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func ParseToleration(st string) (corev1.Toleration, error) {
	var toleration corev1.Toleration
	var key string
	var value string
	var effect corev1.TaintEffect
	var operator corev1.TolerationOperator

	if st == "*" {
		toleration.Operator = corev1.TolerationOpExists
		return toleration, nil
	}
	partsCl := strings.Split(st, ":")
	partsEq := strings.Split(st, "=")
	switch len(partsCl) {
	case 1:
		switch len(partsEq) {
		case 1:
			key = partsEq[0]
		case 2:
			key = partsEq[0]
			value = partsEq[1]
			if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
				return toleration, fmt.Errorf("invalid toleration spec: %v, %s", st, strings.Join(errs, "; "))
			}
		default:
			return toleration, fmt.Errorf("invalid toleration spec: %v", st)
		}
	case 2:
		effect = corev1.TaintEffect(partsCl[1])
		operator = corev1.TolerationOpExists
		partsKV := strings.Split(partsCl[0], "=")
		if len(partsKV) > 2 {
			return toleration, fmt.Errorf("invalid toleration spec: %v", st)
		}
		key = partsKV[0]
		if len(partsKV) == 2 {
			operator = corev1.TolerationOpEqual
			value = partsKV[1]
			if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
				return toleration, fmt.Errorf("invalid toleration spec: %v, %s", st, strings.Join(errs, "; "))
			}
		}
	default:
		return toleration, fmt.Errorf("invalid toleration spec: %v", st)
	}

	toleration.Key = key
	toleration.Value = value
	toleration.Effect = effect
	toleration.Operator = operator
	return toleration, nil
}
