package k8s

import "os"

// Thin wrappers so the test file can manipulate the environment without
// importing os directly (keeping the surface area of the tests minimal and
// easy to grep for env accesses).

func lookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func setEnv(key, value string) error {
	return os.Setenv(key, value)
}

func unsetEnv(key string) error {
	return os.Unsetenv(key)
}
