package types

import (
	"net/http"
)

type Filter func(http.Handler) http.Handler

type OriginalUserKeyType int

const (
	OriginalUserKey OriginalUserKeyType = iota
)
