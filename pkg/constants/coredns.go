package constants

// Please refer to https://github.com/coredns/deployment/blob/master/kubernetes/CoreDNS-k8s_version.md

var CoreDNSVersionMap = map[string]string{
	"1.32": "coredns/coredns:1.11.3",
	"1.31": "coredns/coredns:1.11.3",
	"1.30": "coredns/coredns:1.11.3",
	"1.33": "coredns/coredns:1.12.0",
}

var (
	CoreDNSLabelKey   = "k8s-app"
	CoreDNSLabelValue = "vcluster-kube-dns"
)
