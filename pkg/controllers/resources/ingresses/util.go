package ingresses

import (
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
)

func SecretNamesFromIngress(ingress *networkingv1beta1.Ingress) []string {
	secrets := []string{}
	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName != "" {
			secrets = append(secrets, ingress.Namespace+"/"+tls.SecretName)
		}
	}
	return translate.UniqueSlice(secrets)
}
