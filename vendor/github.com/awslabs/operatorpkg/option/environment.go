package option

import (
	"os"

	"github.com/samber/lo"
)

func MustGetEnv(name string) string {
	env, exists := os.LookupEnv(name)
	lo.Must0(lo.Validate(exists, "env var %s must exist", name))
	return env
}
