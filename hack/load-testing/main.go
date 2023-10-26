package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/hack/load-testing/tests/throughput"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func printUsage() {
	_, _ = fmt.Fprintln(os.Stderr, "usage: load-testing <TEST> [-namespace STRING]")
	os.Exit(1)
}

func printError(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	os.Exit(1)
}

func main() {
	ctx := context.Background()
	namespace := ""
	flag.StringVar(&namespace, "namespace", "load-testing", "namespace to use")
	flag.Parse()

	test := flag.Arg(0)
	if test == "" {
		test = "throughput"
	}

	// We increase the limits here so that we don't get any problems
	restConfig := ctrl.GetConfigOrDie()
	restConfig.QPS = 9999999
	restConfig.Burst = 9999999
	restConfig.Timeout = 0

	// build client
	kubeClient, err := client.New(restConfig, client.Options{})
	if err != nil {
		printError(err)
		return
	}

	switch test {
	case "throughput":
		err = throughput.TestThroughput(ctx, kubeClient, namespace)
		if err != nil {
			printError(err)
		}
	default:
		printUsage()
		return
	}

	klog.FromContext(ctx).Info("Test succeeded", "test", test)
}
