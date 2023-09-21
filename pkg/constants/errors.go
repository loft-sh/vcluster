package constants

import "errors"

var (
	ErrOnlyInPro = errors.New("this command is only available in vcluster pro")
)
