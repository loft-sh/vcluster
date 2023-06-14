package types

import (
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
)

type FilterOptions struct {
	LocalScheme *runtime.Scheme
}

type Filter func(http.Handler, FilterOptions) http.Handler

type OriginalUserKeyType int

const (
	OriginalUserKey OriginalUserKeyType = iota
)
