package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/hack/load-testing/tests/events"
	"github.com/loft-sh/vcluster/hack/load-testing/tests/secrets"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func printUsage() {
	_, _ = fmt.Fprintln(os.Stderr, "usage: load-testing <TEST> [-amount INT] [-namespace STRING]")
	os.Exit(1)
}

func printError(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	os.Exit(1)
}

func main() {
	ctx := context.Background()
	amount := flag.Int64("amount", 100, "amount to create")

	namespace := ""
	flag.StringVar(&namespace, "namespace", "load-testing", "namespace to use")
	flag.Parse()

	test := flag.Arg(0)
	if test == "" {
		printUsage()
		return
	}

	// We increase the limits here so that we don't get any problems
	restConfig := ctrl.GetConfigOrDie()
	restConfig.QPS = 1000
	restConfig.Burst = 2000
	restConfig.Timeout = 0

	// build client
	kubeClient, err := client.New(restConfig, client.Options{})
	if err != nil {
		printError(err)
		return
	}

	switch test {
	case "secrets":
		err = secrets.TestSecrets(ctx, kubeClient, *amount, namespace)
		if err != nil {
			printError(err)
		}
	case "events":
		err = events.TestEvents(ctx, kubeClient, restConfig, *amount, namespace)
		if err != nil {
			printError(err)
		}
	default:
		printUsage()
		return
	}

	klog.FromContext(ctx).Info("Test succeeded", "test", test)
}
