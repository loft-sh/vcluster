package applier

/*
Originally sourced from https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern/tree/0867fae819470ae478f2a90df9d943f5b7ee0b4f/pkg/patterns/declarative/pkg/applier/direct.go
*/

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
)

type Applier interface {
	Apply(ctx context.Context, options applierOptions) error
}

type applierOptions struct {
	Manifest string

	RESTConfig *rest.Config
	RESTMapper meta.RESTMapper
	Namespace  string
	ExtraArgs  []string
}

type ApplyOptions func(*applierOptions)

func WithNamespace(ns string) ApplyOptions {
	return func(opts *applierOptions) {
		opts.Namespace = ns
	}
}

func WithExtraArgs(args []string) ApplyOptions {
	return func(opts *applierOptions) {
		opts.ExtraArgs = args
	}
}
