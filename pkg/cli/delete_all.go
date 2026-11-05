package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/platform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// deleteResult records the outcome of deleting one virtual cluster with --all.
type deleteResult struct {
	Name string
	// Location is the namespace (helm) or project (platform) of the virtual
	// cluster. It is empty for the docker driver.
	Location string
	Err      error
}

// helmDeleteTargets is the outcome of selecting which helm virtual clusters
// --all is going to delete.
type helmDeleteTargets struct {
	// Targets belong to the kube context the command operates on.
	Targets []find.VCluster
	// SkippedParent were listed from the parent context because the current
	// context is itself a virtual cluster. They are reported but never deleted.
	SkippedParent []find.VCluster
	// InKubeSystem counts the targets that run in the kube-system namespace.
	InKubeSystem int
}

// selectHelmTargets splits the listed virtual clusters into the ones that
// belong to kubeContext and the ones that were listed from a parent context.
func selectHelmTargets(vClusters []find.VCluster, kubeContext string) helmDeleteTargets {
	result := helmDeleteTargets{}
	for _, vCluster := range vClusters {
		if vCluster.Context != kubeContext {
			result.SkippedParent = append(result.SkippedParent, vCluster)
			continue
		}

		if vCluster.Namespace == metav1.NamespaceSystem {
			result.InKubeSystem++
		}
		result.Targets = append(result.Targets, vCluster)
	}

	return result
}

// DeleteAllHelm deletes every helm driven virtual cluster in the current kube
// context, or only the ones in globalFlags.Namespace when it is set.
func DeleteAllHelm(ctx context.Context, platformClient platform.Client, options *DeleteOptions, globalFlags *flags.GlobalFlags, log log.Logger) error {
	kubeContext := globalFlags.Context
	if kubeContext == "" {
		var err error
		kubeContext, _, err = find.CurrentContext()
		if err != nil {
			return err
		}
	}

	namespace := metav1.NamespaceAll
	if globalFlags.Namespace != "" {
		namespace = globalFlags.Namespace
	}

	vClusters, err := find.ListVClusters(ctx, kubeContext, "", namespace, log)
	if err != nil {
		return err
	}
	targets := selectHelmTargets(vClusters, kubeContext)

	// warn first when the command runs inside a virtual cluster: only the nested
	// virtual clusters are deleted, the ones in the parent context are skipped.
	if nestedName, _, parentContext := find.VClusterFromContext(kubeContext); nestedName != "" {
		log.Warnf("You are connected to virtual cluster %s (context %s). Only virtual clusters running inside it will be deleted.", nestedName, kubeContext)
		if len(targets.SkippedParent) > 0 {
			log.Warnf("Skipping %d virtual cluster(s) in the parent context %s. Run 'vcluster delete --all --context %s' to delete those.", len(targets.SkippedParent), parentContext, parentContext)
		}
	}

	// warn second about kube-system, and refuse to combine it with --delete-namespace
	if targets.InKubeSystem > 0 {
		if options.DeleteNamespace {
			return fmt.Errorf("refusing to delete %d virtual cluster(s) in namespace %s together with --delete-namespace, delete those virtual clusters individually", targets.InKubeSystem, metav1.NamespaceSystem)
		}
		log.Warnf("%d of the virtual clusters below run in namespace %s", targets.InKubeSystem, metav1.NamespaceSystem)
	}

	if len(targets.Targets) == 0 {
		if namespace == metav1.NamespaceAll {
			log.Infof("No virtual clusters found in context %s", kubeContext)
		} else {
			log.Infof("No virtual clusters found in namespace %s of context %s", namespace, kubeContext)
		}
		return nil
	}

	rows := make([][]string, 0, len(targets.Targets))
	for _, vCluster := range targets.Targets {
		rows = append(rows, []string{vCluster.Name, vCluster.Namespace})
	}
	log.Infof("The following virtual clusters in context %s will be deleted:", kubeContext)
	table.PrintTable(log, []string{"NAME", "NAMESPACE"}, rows)

	confirmed, err := confirmDeleteAll(len(targets.Targets), options, log)
	if err != nil {
		return err
	} else if !confirmed {
		log.Info("delete cancelled")
		return nil
	}

	results := make([]deleteResult, 0, len(targets.Targets))
	for i := range targets.Targets {
		vCluster := &targets.Targets[i]
		log.Infof("Deleting virtual cluster %s in namespace %s (%d/%d)...", vCluster.Name, vCluster.Namespace, i+1, len(targets.Targets))

		// every deletion gets its own copy of the options
		perVClusterOptions := *options
		err := deleteHelmVCluster(ctx, platformClient, &perVClusterOptions, globalFlags, vCluster, log)
		results = append(results, deleteResult{Name: vCluster.Name, Location: vCluster.Namespace, Err: err})
	}

	return reportDeleteResults(results, "NAMESPACE", log)
}

// DeleteAllPlatform deletes every platform driven virtual cluster the user can
// access, or only the ones in options.Project when it is set.
func DeleteAllPlatform(ctx context.Context, platformClient platform.Client, options *DeleteOptions, log log.Logger) error {
	if platformClient == nil {
		return fmt.Errorf("platform client not set")
	}

	vClusters, err := platform.ListVClusters(ctx, platformClient, "", options.Project, false)
	if err != nil {
		return err
	}

	targets := make([]*platform.VirtualClusterInstanceProject, 0, len(vClusters))
	for _, vCluster := range vClusters {
		if vCluster.VirtualCluster != nil && vCluster.VirtualCluster.Spec.External {
			log.Infof("Skipping virtual cluster %s in project %s because it was created via helm, use '--driver helm' to delete it", vCluster.VirtualCluster.Name, vCluster.Project.Name)
			continue
		}
		targets = append(targets, vCluster)
	}

	if len(targets) == 0 {
		if options.Project == "" {
			log.Info("No virtual clusters found in the platform")
		} else {
			log.Infof("No virtual clusters found in project %s", options.Project)
		}
		return nil
	}

	rows := make([][]string, 0, len(targets))
	for _, vCluster := range targets {
		rows = append(rows, []string{vCluster.VirtualCluster.Name, vCluster.Project.Name})
	}
	log.Info("The following virtual clusters in the platform will be deleted:")
	table.PrintTable(log, []string{"NAME", "PROJECT"}, rows)

	confirmed, err := confirmDeleteAll(len(targets), options, log)
	if err != nil {
		return err
	} else if !confirmed {
		log.Info("delete cancelled")
		return nil
	}

	results := make([]deleteResult, 0, len(targets))
	for i, vCluster := range targets {
		log.Infof("Deleting virtual cluster %s in project %s (%d/%d)...", vCluster.VirtualCluster.Name, vCluster.Project.Name, i+1, len(targets))

		// every deletion gets its own copy of the options with the project pinned
		perVClusterOptions := *options
		perVClusterOptions.Project = vCluster.Project.Name
		err := deletePlatformVCluster(ctx, platformClient, &perVClusterOptions, vCluster, log)
		results = append(results, deleteResult{Name: vCluster.VirtualCluster.Name, Location: vCluster.Project.Name, Err: err})
	}

	return reportDeleteResults(results, "PROJECT", log)
}

// DeleteAllDocker deletes every docker driven virtual cluster on the local
// docker daemon.
func DeleteAllDocker(ctx context.Context, platformClient platform.Client, options *DeleteOptions, globalFlags *flags.GlobalFlags, log log.Logger) error {
	// make sure a docker daemon is reachable, falling back to podman if available
	err := EnsureDockerDaemon(ctx, log)
	if err != nil {
		return err
	}

	vClusters, err := findDockerContainer(ctx, constants.DockerControlPlanePrefix)
	if err != nil {
		return fmt.Errorf("failed to list docker vclusters: %w", err)
	}

	if len(vClusters) == 0 {
		log.Info("No virtual clusters found in docker")
		return nil
	}

	rows := make([][]string, 0, len(vClusters))
	for _, vCluster := range vClusters {
		rows = append(rows, []string{vCluster.Name})
	}
	log.Info("The following virtual clusters in docker will be deleted:")
	table.PrintTable(log, []string{"NAME"}, rows)

	confirmed, err := confirmDeleteAll(len(vClusters), options, log)
	if err != nil {
		return err
	} else if !confirmed {
		log.Info("delete cancelled")
		return nil
	}

	results := make([]deleteResult, 0, len(vClusters))
	for i, vCluster := range vClusters {
		log.Infof("Deleting virtual cluster %s (%d/%d)...", vCluster.Name, i+1, len(vClusters))

		// every deletion gets its own copy of the options
		perVClusterOptions := *options
		err := DeleteDocker(ctx, platformClient, &perVClusterOptions, globalFlags, vCluster.Name, log)
		results = append(results, deleteResult{Name: vCluster.Name, Err: err})
	}

	return reportDeleteResults(results, "", log)
}

// confirmDeleteAll asks the user to confirm the deletion. It returns true
// without asking when --non-interactive is set and refuses to continue when
// there is no terminal to ask on.
func confirmDeleteAll(count int, options *DeleteOptions, log log.Logger) (bool, error) {
	if options.NonInteractive {
		return true, nil
	}

	if !terminal.IsTerminalIn {
		return false, errors.New("refusing to delete all virtual clusters without confirmation, rerun with --non-interactive to skip the prompt")
	}

	answer, err := log.Question(&survey.QuestionOptions{
		Question:     fmt.Sprintf("Delete all %d virtual cluster(s) listed above?", count),
		Options:      []string{util.PositiveResponse, util.NegativeResponse},
		DefaultValue: util.NegativeResponse,
	})
	if err != nil {
		return false, err
	}

	return answer == util.PositiveResponse, nil
}

// reportDeleteResults prints one row per virtual cluster and returns an error
// that wraps every failed deletion, or nil when all of them succeeded.
func reportDeleteResults(results []deleteResult, locationHeader string, log log.Logger) error {
	header := []string{"NAME", "RESULT"}
	if locationHeader != "" {
		header = []string{"NAME", locationHeader, "RESULT"}
	}

	var errs []error
	rows := make([][]string, 0, len(results))
	for _, result := range results {
		outcome := "deleted"
		if result.Err != nil {
			outcome = result.Err.Error()
			if locationHeader != "" {
				errs = append(errs, fmt.Errorf("%s/%s: %w", result.Location, result.Name, result.Err))
			} else {
				errs = append(errs, fmt.Errorf("%s: %w", result.Name, result.Err))
			}
		}

		row := []string{result.Name}
		if locationHeader != "" {
			row = append(row, result.Location)
		}
		rows = append(rows, append(row, outcome))
	}
	table.PrintTable(log, header, rows)

	if len(errs) > 0 {
		return fmt.Errorf("%d of %d virtual cluster(s) could not be deleted: %w", len(errs), len(results), errors.Join(errs...))
	}

	log.Donef("Successfully deleted %d virtual cluster(s)", len(results))
	return nil
}
