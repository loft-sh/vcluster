package translate

import (
	"fmt"
	"strings"
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
)

var (
	v127 = utilversion.MustParseSemantic("1.27.0") // pre-ExpandedDNSConfig GA -> legacy limits (6/256)
	v128 = utilversion.MustParseSemantic("1.28.0") // ExpandedDNSConfig GA-locked  -> expanded limits (32/2048)
)

func vTeamPod() *corev1.Pod {
	p := &corev1.Pod{}
	p.Namespace = "team"
	return p
}

// distinctSearches returns n distinct search domains. Distinctness matters: the merge
// runs deleteDuplicates, so repeated domains would collapse and never exercise the cap.
func distinctSearches(n int) []string {
	s := make([]string, n)
	for i := range s {
		s[i] = fmt.Sprintf("d%d.team.svc.example.com", i)
	}
	return s
}

func TestTranslateDNSNameserverCap(t *testing.T) {
	// 3 distinct user nameservers + the prepended cluster IP = 4 -> must cap to 3, cluster first.
	pPod := &corev1.Pod{Spec: corev1.PodSpec{DNSConfig: &corev1.PodDNSConfig{
		Nameservers: []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"},
	}}}
	translateDNSClusterFirstConfig(pPod, vTeamPod(), "cluster.local", "10.0.0.10", v128)

	ns := pPod.Spec.DNSConfig.Nameservers
	assert.Assert(t, len(ns) <= maxDNSNameservers, "nameservers = %d, exceeds host max %d: %v", len(ns), maxDNSNameservers, ns)
	assert.Equal(t, ns[0], "10.0.0.10", "cluster nameserver must be kept first")
}

func TestTranslateDNSSearchCountCapExpanded(t *testing.T) {
	// 40 distinct search domains + 3 cluster = 43 -> must cap to 32 on an expanded (>=1.28)
	// host, with the cluster search domain kept first.
	pPod := &corev1.Pod{Spec: corev1.PodSpec{DNSConfig: &corev1.PodDNSConfig{Searches: distinctSearches(40)}}}
	translateDNSClusterFirstConfig(pPod, vTeamPod(), "cluster.local", "10.0.0.10", v128)

	s := pPod.Spec.DNSConfig.Searches
	assert.Assert(t, len(s) <= maxDNSSearchPathsExpanded, "searches = %d, exceeds expanded max %d", len(s), maxDNSSearchPathsExpanded)
	assert.Equal(t, s[0], "team.svc.cluster.local", "cluster search domain must be kept first")
}

func TestTranslateDNSSearchCountCapLegacy(t *testing.T) {
	// On a pre-1.28 host the validated limit is 6, not 32: 43 domains must cap to 6. Fails if
	// the code hardcodes the expanded limit instead of honoring the host version.
	pPod := &corev1.Pod{Spec: corev1.PodSpec{DNSConfig: &corev1.PodDNSConfig{Searches: distinctSearches(40)}}}
	translateDNSClusterFirstConfig(pPod, vTeamPod(), "cluster.local", "10.0.0.10", v127)

	got := len(pPod.Spec.DNSConfig.Searches)
	assert.Assert(t, got <= maxDNSSearchPathsLegacy, "searches = %d on a pre-1.28 host, exceeds legacy max %d", got, maxDNSSearchPathsLegacy)
}

func TestTranslateDNSSearchCharLengthCap(t *testing.T) {
	// Few but long distinct domains: the count (12+3=15) is under the expanded limit (32), but
	// the joined length exceeds 2048 chars, so the character-length cap must trigger. Fails if
	// only the count is capped.
	long := make([]string, 12)
	for i := range long {
		long[i] = fmt.Sprintf("%s-%d.%s.%s.example.com",
			strings.Repeat("a", 55), i, strings.Repeat("b", 60), strings.Repeat("c", 60))
	}
	pPod := &corev1.Pod{Spec: corev1.PodSpec{DNSConfig: &corev1.PodDNSConfig{Searches: long}}}
	translateDNSClusterFirstConfig(pPod, vTeamPod(), "cluster.local", "10.0.0.10", v128)

	s := pPod.Spec.DNSConfig.Searches
	joined := len(strings.Join(s, " "))
	assert.Assert(t, joined <= maxDNSSearchListCharsExpanded, "joined search length = %d, exceeds host limit of %d chars", joined, maxDNSSearchListCharsExpanded)
	assert.Assert(t, len(s) < 15, "expected the character-length cap to drop some domains, kept all %d", len(s))
}
