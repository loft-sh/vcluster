package constants

import (
	"fmt"
	"runtime"
	"strings"

	"k8s.io/client-go/pkg/version"
)

const (
	FieldManager = "vcluster"
)

// returns vcluster user agent with a constant name part used by the API server for FieldManager
func GetVclusterUserAgent() string {
	// Based on client-go code - https://github.com/kubernetes/client-go/blob/3e0d99053357a2c37e68b65adbf16437803b1f64/rest/config.go#LL498
	v := version.Get().GitVersion
	if len(v) == 0 {
		v = "unknown"
	}
	v = strings.SplitN(v, "-", 2)[0]

	c := version.Get().GitCommit
	if len(c) == 0 {
		c = "unknown"
	}
	if len(c) > 7 {
		c = c[:7]
	}

	return fmt.Sprintf("%s/%s (%s/%s) kubernetes/%s", FieldManager, v, runtime.GOOS, runtime.GOARCH, c)
}
