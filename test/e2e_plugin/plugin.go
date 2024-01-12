package e2eplugin

import (
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pollingInterval     = time.Second * 2
	pollingDurationLong = time.Second * 60
)

var _ = ginkgo.Describe("plugin", func() {
	f := framework.DefaultFramework

	ginkgo.It("test legacy vCluster plugin", func() {
		// check if deployment is there
		gomega.Eventually(func() bool {
			_, err := f.VclusterClient.AppsV1().Deployments("default").Get(f.Context, "mydeployment", metav1.GetOptions{})
			return err == nil
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.BeTrue())
	})
})
