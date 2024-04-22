package cli

import "fmt"

type ActivateOptions struct {
	Manager string

	ClusterName string
	Project     string
	ImportName  string
}

func ActivateHelm() error {
	return fmt.Errorf("this function is currently not implemented")
}
