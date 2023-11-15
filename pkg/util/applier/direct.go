package applier

/*
Originally sourced from https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern/tree/0867fae819470ae478f2a90df9d943f5b7ee0b4f/pkg/patterns/declarative/pkg/applier/direct.go
*/

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/apply"
	cmdDelete "k8s.io/kubectl/pkg/cmd/delete"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type DirectApplier struct {
}

var _ Applier = &DirectApplier{}

func NewDirectApplier() *DirectApplier {
	return &DirectApplier{}
}

func (d *DirectApplier) Apply(_ context.Context, opt Options) error {
	ioStreams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	ioReader := strings.NewReader(opt.Manifest)

	restClientGetter := &staticRESTClientGetter{
		RESTMapper: opt.RESTMapper,
		RESTConfig: opt.RESTConfig,
	}

	factory := cmdutil.NewFactory(restClientGetter)
	res := resource.NewBuilder(restClientGetter).Unstructured().Stream(ioReader, "manifests").Do()
	infos, resErr := res.Infos()

	// Populate the namespace on any namespace-scoped objects
	if opt.Namespace != "" {
		visitor := resource.SetNamespace(opt.Namespace)
		for _, info := range infos {
			if err := info.Visit(visitor); err != nil {
				return fmt.Errorf("error from SetNamespace: %w", err)
			}
		}
	}

	flags := apply.NewApplyFlags(ioStreams)
	flags.AddFlags(&cobra.Command{})
	applyOpts, err := newOptions(factory, flags, opt.Namespace)
	if err != nil {
		return err
	}

	applyOpts.SetObjects(infos)
	applyOpts.DeleteOptions = &cmdDelete.DeleteOptions{
		IOStreams: ioStreams,
	}

	err = applyOpts.Run()
	if err != nil {
		return err
	}

	return resErr
}

func newOptions(factory cmdutil.Factory, flags *apply.ApplyFlags, namespace string) (*apply.ApplyOptions, error) {
	dynamicClient, err := factory.DynamicClient()
	if err != nil {
		return nil, err
	}

	// allow for a success message operation to be specified at print time
	toPrinter := func(operation string) (printers.ResourcePrinter, error) {
		flags.PrintFlags.NamePrintFlags.Operation = operation
		cmdutil.PrintFlagsWithDryRunStrategy(flags.PrintFlags, cmdutil.DryRunNone)
		return flags.PrintFlags.ToPrinter()
	}

	recorder := genericclioptions.NoopRecorder{}
	deleteOptions := &cmdDelete.DeleteOptions{
		DynamicClient: dynamicClient,
		IOStreams:     flags.IOStreams,
	}

	builder := factory.NewBuilder()
	mapper, err := factory.ToRESTMapper()
	if err != nil {
		return nil, err
	}

	o := &apply.ApplyOptions{
		PrintFlags: flags.PrintFlags,

		DeleteOptions:   deleteOptions,
		ToPrinter:       toPrinter,
		ServerSideApply: false,
		ForceConflicts:  false,
		FieldManager:    "vcluster",
		Selector:        "",
		DryRunStrategy:  cmdutil.DryRunNone,
		Prune:           false,
		PruneResources:  nil,
		All:             flags.All,
		Overwrite:       flags.Overwrite,
		OpenAPIPatch:    flags.OpenAPIPatch,

		Recorder:         recorder,
		Namespace:        namespace,
		EnforceNamespace: true,
		Builder:          builder,
		Mapper:           mapper,
		DynamicClient:    dynamicClient,

		IOStreams:         flags.IOStreams,
		VisitedUids:       sets.New[types.UID](),
		VisitedNamespaces: sets.New[string](),
	}

	o.PostProcessorFn = o.PrintAndPrunePostProcessor()
	return o, nil
}

// staticRESTClientGetter returns a fixed RESTClient
type staticRESTClientGetter struct {
	RESTConfig      *rest.Config
	DiscoveryClient discovery.CachedDiscoveryInterface
	RESTMapper      meta.RESTMapper
}

var _ resource.RESTClientGetter = &staticRESTClientGetter{}

func (s *staticRESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	if s.RESTConfig == nil {
		return nil, fmt.Errorf("RESTConfig not set")
	}
	return s.RESTConfig, nil
}
func (s *staticRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	if s.DiscoveryClient == nil {
		return nil, fmt.Errorf("DiscoveryClient not set")
	}
	return s.DiscoveryClient, nil
}
func (s *staticRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	if s.RESTMapper == nil {
		return nil, fmt.Errorf("RESTMapper not set")
	}
	return s.RESTMapper, nil
}
func (s *staticRESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return nil
}
