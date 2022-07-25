package manifests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/util/compress"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InitObjectStatus string

const (
	InitChartsKey          = "charts"
	InitManifestSuffix     = "-init-manifests"
	LastAppliedManifestKey = "vcluster.loft.sh/last-applied-init-manifests"
	LastAppliedChartConfig = "vcluster.loft.sh/last-applied-chart-config"

	InitManifestsStatusKey = "vcluster.loft.sh/init-status"
	InitChartStatusKey     = "vcluster.loft.sh/init-chart-status"

	DefaultTimeOut = 120 * time.Second

	HelmWorkDir = "/tmp"

	StatusFailed  InitObjectStatus = "Failed"
	StatusSuccess InitObjectStatus = "Success"

	ChartPullError = "ChartPullFailed"
	InstallError   = "InstallFailed"
	UpgradeError   = "UpgradeFailed"
	RollBackError  = "RollbackFailed"
)

type InitManifestsConfigMapReconciler struct {
	Log loghelper.Logger

	LocalClient    client.Client
	VirtualManager ctrl.Manager

	HelmClient helm.Client
}

func (r *InitManifestsConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO: implement better filteration through predicates
	if req.Name == translate.Suffix+InitManifestSuffix {
		r.Log.Infof("process init manifests")
		cm, err := r.getConfigMap(ctx, req)
		if err != nil {
			return ctrl.Result{}, err
		}

		if cm == nil {
			return ctrl.Result{}, nil
		}

		result, err := r.ProcessInitManifests(ctx, cm)
		if err != nil {
			return result, err
		}

		if !result.IsZero() {
			// might be a requeue by ProcessInitManifests
			// - we dont explicitly do any requeue this way
			// but keeping this for possible use in future
			return result, nil
		}

		var charts []Chart
		if rawCharts, ok := cm.Data[InitChartsKey]; ok {
			err := yaml.Unmarshal([]byte(rawCharts), &charts)
			if err != nil {
				r.Log.Errorf("error unmarshalling charts: %v", err)
				return ctrl.Result{}, err
			}

			r.Log.Infof("Successfully unmarshalled charts")

			return r.ProcessHelmChart(ctx, charts, req)
		}

	}

	return ctrl.Result{}, nil
}

func (r *InitManifestsConfigMapReconciler) SetupWithManager(hostMgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(hostMgr).
		Named("init_manifests").
		For(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *InitManifestsConfigMapReconciler) getConfigMap(ctx context.Context, req ctrl.Request) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := r.LocalClient.Get(ctx, req.NamespacedName, cm)
	if err != nil {
		if kerrors.IsNotFound(err) {
			r.Log.Errorf("configmap not found %v", err)
			return nil, nil
		}

		return nil, err
	}

	return cm, nil
}

func (r *InitManifestsConfigMapReconciler) ProcessInitManifests(ctx context.Context, cm *corev1.ConfigMap) (ctrl.Result, error) {
	var cmData []string
	for k, v := range cm.Data {
		if k != InitChartsKey {
			cmData = append(cmData, v)
		}
	}

	// make array stable or otherwise order is random
	sort.Strings(cmData)
	manifests := strings.Join(cmData, "\n---\n")
	lastAppliedManifests := ""
	if cm.ObjectMeta.Annotations != nil {
		lastAppliedManifests = cm.ObjectMeta.Annotations[LastAppliedManifestKey]
		if lastAppliedManifests != "" {
			var err error
			lastAppliedManifests, err = compress.Uncompress(lastAppliedManifests)
			if err != nil {
				r.Log.Errorf("error decompressing manifests: %v", err)
			}
		}
	}

	// should skip?
	if manifests == lastAppliedManifests {
		return ctrl.Result{}, nil
	}

	// apply manifests
	err := ApplyGivenInitManifests(ctx, r.VirtualManager.GetClient(), r.VirtualManager.GetConfig(), manifests, lastAppliedManifests)
	if err != nil {
		r.Log.Errorf("error applying init manifests: %v", err)
		_ = r.SetStatus(ctx, nil, types.NamespacedName{
			Name:      cm.Name,
			Namespace: cm.Namespace,
		}, StatusFailed, InstallError, err)
		return ctrl.Result{}, err
	}

	// apply successful, store in an annotation in the configmap itself
	compressedManifests, err := compress.Compress(manifests)
	if err != nil {
		r.Log.Errorf("error compressing manifests: %v", err)
		return ctrl.Result{}, err
	}

	// update annotation
	if cm.ObjectMeta.Annotations == nil {
		cm.ObjectMeta.Annotations = map[string]string{}
	}
	cm.ObjectMeta.Annotations[LastAppliedManifestKey] = compressedManifests
	_ = r.SetStatus(ctx, nil, types.NamespacedName{
		Name:      cm.Name,
		Namespace: cm.Namespace,
	}, StatusSuccess, "", nil)

	err = r.LocalClient.Update(ctx, cm)
	if err != nil {
		r.Log.Errorf("error updating config map with last applied annotation: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *InitManifestsConfigMapReconciler) ProcessHelmChart(ctx context.Context, charts []Chart, req ctrl.Request) (ctrl.Result, error) {
	for _, chart := range charts {
		r.Log.Infof("processing helm chart for %s", chart.Name)

		err := r.pullChartArchive(ctx, chart)
		if err != nil {
			r.SetStatus(ctx, &chart, req.NamespacedName, StatusFailed, ChartPullError, err)
			return ctrl.Result{}, err
		}

		name, _ := r.getChartOrReleaseDetails(ctx, chart)

		exists, err := r.releaseExists(ctx, chart)
		if err != nil {
			return ctrl.Result{}, err
		}

		if exists {
			r.Log.Infof("release for chart %s already exists", name)
			// check if upgrade is needed
			upgradedNeeded, err := r.checkIfUpgradeNeeded(ctx, chart, req)
			if err != nil {
				return ctrl.Result{}, err
			}

			if upgradedNeeded {
				// initiate upgrade
				err = r.initiateUpgrade(ctx, chart, req)
				if err != nil {
					return ctrl.Result{}, err
				}

				// register this configuration
				err = r.registerLastAppliedChartConfig(ctx, chart, req)
				if err != nil {
					r.Log.Errorf("error updating config map with last applied chart annotation: %v", err)
					return ctrl.Result{}, err
				}

				return ctrl.Result{}, nil
			}

			err = r.SetStatus(ctx, &chart, req.NamespacedName, StatusSuccess, "", nil)
			if err != nil {
				r.Log.Errorf("error setting success status for release %s that already exists: %v", name, err)
				return ctrl.Result{}, err
			}

			// continue to process next chart
			continue
		}

		r.Log.Infof("initaiting installation for chart %s", name)
		// initiate install
		err = r.initiateInstall(ctx, chart, req)
		if err != nil {
			r.Log.Errorf("error installing chart %s", name)
			return ctrl.Result{}, err
		}

		// register this configuration
		err = r.registerLastAppliedChartConfig(ctx, chart, req)
		if err != nil {
			r.Log.Errorf("error updating config map with last applied chart annotation: %v", err)
			return ctrl.Result{}, err
		}

		// install only one chart successfully in each reconcile
		// hence reconcile here without error
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *InitManifestsConfigMapReconciler) checkIfUpgradeNeeded(ctx context.Context, chart Chart, req ctrl.Request) (bool, error) {
	name, _ := r.getChartOrReleaseDetails(ctx, chart)

	// compare with last applied config
	lastAppliedConfig, err := r.getLastAppliedChartConfig(ctx, chart, req)
	if err != nil {
		r.Log.Infof("error getting last applied chart config: %v", err)
		return false, err
	}

	if lastAppliedConfig == "" {
		// should register current config as the last applied config?
		r.Log.Infof("release for chart %s exists but no last applied config exists", name)

		err := r.registerLastAppliedChartConfig(ctx, chart, req)
		if err != nil {
			r.Log.Errorf("error updating config map with last applied chart annotation: %v", err)
			return false, err
		}

		r.Log.Infof("updated current config as last applied config for chart: %s", name)
		return false, nil
	}

	currentConfig, err := r.getCompressedConfig(ctx, chart)
	if err != nil {
		r.Log.Errorf("cannot get current config")
		return false, err
	}

	if lastAppliedConfig != currentConfig {
		// initiate upgrade
		r.Log.Infof("current config different than last config, upgrade need for chart %s", name)
		return true, nil
	}

	return false, nil
}

func (r *InitManifestsConfigMapReconciler) getLastAppliedChartConfig(ctx context.Context, chart Chart, req ctrl.Request) (string, error) {
	cm := corev1.ConfigMap{}

	err := r.LocalClient.Get(ctx, req.NamespacedName, &cm)
	if err != nil {
		return "", err
	}

	lastAppliedConfig := cm.ObjectMeta.Annotations[r.getLastAppliedChartConfigKey(ctx, chart)]

	return lastAppliedConfig, nil
}

func (r *InitManifestsConfigMapReconciler) getLastAppliedChartConfigKey(ctx context.Context, chart Chart) string {
	name, namespace := r.getChartOrReleaseDetails(ctx, chart)
	return fmt.Sprintf("%s-%s-%s", LastAppliedChartConfig, name, namespace)
}

func (r *InitManifestsConfigMapReconciler) initiateUpgrade(ctx context.Context, chart Chart, req ctrl.Request) error {
	name, namespace := r.getChartOrReleaseDetails(ctx, chart)

	lastAppliedConfig, err := r.getLastAppliedChartConfig(ctx, chart, req)
	if err != nil {
		r.Log.Errorf("unable to get last applied config: %v", err)
		return err
	}

	// first check if its the chart version that has changed
	// as that would mean we first need to trigger a new archive to be pulled
	lastConfigRaw, err := compress.Uncompress(lastAppliedConfig)
	if err != nil {
		r.Log.Errorf("unable to uncompress last applied config: %v", err)
	}

	lastConfig := make(map[string]string)
	err = json.Unmarshal([]byte(lastConfigRaw), &lastConfig)
	if err != nil {
		r.Log.Errorf("unable to unmarshal last config: %v", err)
	}

	currentChartVersion := chart.Version
	lastVersion := lastConfig["version"]

	if currentChartVersion != lastVersion {
		err := r.pullChartArchive(ctx, chart)
		if err != nil {
			r.Log.Errorf("unable to pull chart archive for chart %s: %v", name, err)
			r.SetStatus(ctx, &chart, req.NamespacedName, StatusFailed, ChartPullError, err)
			return err
		}

		r.Log.Infof("successfully updated the chart bundle for chart %s", name)
	}

	path := r.getTarballPath(ctx, chart)
	values := chart.Values
	if values == "" {
		r.Log.Infof("no value overrides set for chart %s", name)
	}

	r.Log.Infof("upgrading chart %s from path %s", name, path)

	timedOutContext, cancel := context.WithTimeout(ctx, r.parseTimeout(ctx, chart))
	defer cancel()

	err = r.HelmClient.Upgrade(timedOutContext, name, namespace, helm.UpgradeOptions{
		Chart:           name,
		Path:            path,
		CreateNamespace: true,
		Values:          values,
		WorkDir:         HelmWorkDir,
	})

	if err != nil {
		r.Log.Errorf("unable to upgrade chart %s: %v", name, err)
		_ = r.SetStatus(ctx, &chart, req.NamespacedName, StatusFailed, UpgradeError, err)

		rollbackErr := r.rollbackOrUninstall(ctx, name, namespace)
		if rollbackErr != nil {
			r.Log.Errorf("error while rollingback or uninstall chart %s: %v", name, err)

			_ = r.SetStatus(ctx, &chart, req.NamespacedName, StatusFailed, RollBackError, rollbackErr)

			return rollbackErr
		}
		return err
	}

	r.Log.Infof("successfully upgraded chart %s", name)
	_ = r.SetStatus(ctx, &chart, req.NamespacedName, StatusSuccess, "", nil)

	return nil
}

func (r *InitManifestsConfigMapReconciler) registerLastAppliedChartConfig(ctx context.Context, chart Chart, req ctrl.Request) error {
	compressedConfig, err := r.getCompressedConfig(ctx, chart)
	name, _ := r.getChartOrReleaseDetails(ctx, chart)
	if err != nil {
		r.Log.Errorf("unable to get compressed config for chart %s: %v", name, err)
		return err
	}

	// get the latest configmap in case it might be updated
	latestCm, err := r.getConfigMap(ctx, req)
	if err != nil {
		r.Log.Errorf("unable to get latest configmap object before update: %v", err)
		return err
	}

	if latestCm.ObjectMeta.Annotations == nil {
		latestCm.ObjectMeta.Annotations = map[string]string{}
	}

	latestCm.ObjectMeta.Annotations[r.getLastAppliedChartConfigKey(ctx, chart)] = compressedConfig
	err = r.LocalClient.Update(ctx, latestCm)
	if err != nil {
		r.Log.Errorf("error updating config map with last applied chart annotation: %v", err)
		return err
	}

	return nil
}

func (r *InitManifestsConfigMapReconciler) getTarballPath(ctx context.Context, chart Chart) string {
	name, _ := r.getChartOrReleaseDetails(ctx, chart)
	return fmt.Sprintf("%s/%s-%s.tgz", HelmWorkDir, name, chart.Version)
}

func (r *InitManifestsConfigMapReconciler) initiateInstall(ctx context.Context, chart Chart, req ctrl.Request) error {
	// initiate install
	name, namespace := r.getChartOrReleaseDetails(ctx, chart)

	path := r.getTarballPath(ctx, chart)

	values := chart.Values
	if values == "" {
		r.Log.Infof("no value overrides set for chart %s", name)
	}

	r.Log.Infof("installing chart %s from path %s", name, path)

	timedOutContext, cancel := context.WithTimeout(ctx, r.parseTimeout(ctx, chart))
	defer cancel()

	err := r.HelmClient.Install(timedOutContext, name, namespace, helm.UpgradeOptions{
		Chart:           name,
		Path:            path,
		CreateNamespace: true,
		Values:          values,
		WorkDir:         HelmWorkDir,
	})

	if err != nil {
		r.Log.Errorf("unable to install chart %s: %v", name, err)
		_ = r.SetStatus(ctx, &chart, req.NamespacedName, StatusFailed, InstallError, err)

		rollbackErr := r.rollbackOrUninstall(ctx, name, namespace)
		if rollbackErr != nil {
			r.Log.Errorf("error while rollingback or uninstall chart %s: %v", name, rollbackErr)

			_ = r.SetStatus(ctx, &chart, req.NamespacedName, StatusFailed, RollBackError, rollbackErr)
			return rollbackErr
		}
		return err
	}

	r.Log.Infof("successfully installed chart %s", name)
	_ = r.SetStatus(ctx, &chart, req.NamespacedName, StatusSuccess, "", nil)

	return nil
}

func (r *InitManifestsConfigMapReconciler) getCompressedConfig(ctx context.Context, chart Chart) (string, error) {
	rawData, err := json.Marshal(chart)
	if err != nil {
		return "", err
	}

	return compress.Compress(string(rawData))
}

func (r *InitManifestsConfigMapReconciler) releaseExists(ctx context.Context, chart Chart) (bool, error) {
	name, namespace := r.getChartOrReleaseDetails(ctx, chart)

	ok, err := r.HelmClient.Exists(name, namespace)
	if err != nil {
		r.Log.Errorf("error checking for release existence")
		return false, err
	}

	if ok {
		return true, nil
	}

	return false, nil
}

func (r *InitManifestsConfigMapReconciler) pullChartArchive(ctx context.Context, chart Chart) error {
	name, _ := r.getChartOrReleaseDetails(ctx, chart)

	tarball := r.getTarballPath(ctx, chart)
	// check if tarball exists
	_, err := os.Stat(tarball)
	if err != nil {
		if os.IsNotExist(err) {
			helmErr := r.HelmClient.Pull(ctx, name, helm.UpgradeOptions{
				Chart:   name,
				Repo:    chart.Repo,
				Version: chart.Version,
				WorkDir: HelmWorkDir,
			})

			if helmErr != nil {
				r.Log.Errorf("unable to pull chart %s: %v", name, helmErr)
				return helmErr
			}

			r.Log.Infof("successfully pulled chart %s", name)
		}

		return err
	}

	return nil
}

func (r *InitManifestsConfigMapReconciler) parseTimeout(ctx context.Context, chart Chart) time.Duration {
	t := chart.Timeout

	timeout, err := time.ParseDuration(t)
	if err != nil {
		r.Log.Errorf("unable to parse timeout: %v", err)
		r.Log.Errorf("falling back to default timeout")
		timeout = DefaultTimeOut
	}

	return timeout
}

func (r *InitManifestsConfigMapReconciler) rollbackOrUninstall(ctx context.Context, chartName, namespace string) error {
	output, err := r.HelmClient.Status(ctx, chartName, namespace)
	if err != nil {
		r.Log.Errorf("error getting helm status: %v", err)
		return err
	}

	if strings.Contains(string(output), "pending-install") {
		r.Log.Errorf("release stuck in pending state, proceeding to uninstall")
		err := r.HelmClient.Delete(chartName, namespace)
		if err != nil {
			r.Log.Errorf("unable to delete pending release: %v", err)
			return err
		}

		return nil
	}

	r.Log.Infof("rolling back release %s", chartName)
	err = r.HelmClient.Rollback(ctx, chartName, namespace)
	if err != nil {
		r.Log.Errorf("error rolling back release %s", chartName)
		return err
	}

	return nil
}

func (r *InitManifestsConfigMapReconciler) getChartOrReleaseDetails(ctx context.Context, chart Chart) (string, string) {
	name := chart.Name
	namespace := corev1.NamespaceDefault

	if chart.ReleaseName != "" {
		name = chart.ReleaseName
	}

	if chart.ReleaseNamespace != "" {
		namespace = chart.ReleaseNamespace
	}

	return name, namespace
}

func (r *InitManifestsConfigMapReconciler) SetStatus(ctx context.Context, chart *Chart, obj types.NamespacedName, objStatus InitObjectStatus, reason string, errMsg error) error {
	cm := &corev1.ConfigMap{}
	statusKey := InitManifestsStatusKey

	var c Chart
	if chart != nil {
		statusKey = InitChartStatusKey
		c = *chart
	}

	klog.Infof("Setting init resource status")
	err := r.LocalClient.Get(ctx, obj, cm)
	if err != nil {
		klog.Errorf("error setting init resource status %v", err)
		return err
	}

	if cm.Annotations == nil {
		cm.Annotations = map[string]string{}
	}

	rawStatus := cm.Annotations[statusKey]

	var statuses []ChartStatus
	err = yaml.Unmarshal([]byte(rawStatus), &statuses)
	if err != nil {
		r.Log.Errorf("error unmarshalling rawStatus: %v", err)
	}

	name, ns := r.getChartOrReleaseDetails(ctx, c)
	chartStatusNamespacedName := fmt.Sprintf("%s-%s", name, ns)

	var newStatus ChartStatus
	if objStatus == StatusSuccess {
		newStatus = ChartStatus{
			Name:  chartStatusNamespacedName,
			Phase: string(objStatus),
		}
	} else {
		newStatus = ChartStatus{
			Name:    chartStatusNamespacedName,
			Message: errMsg.Error(),
			Reason:  reason,
		}
	}

	found := false
	// find the correct chart in the array and update it
	for i, status := range statuses {
		if status.Name == chartStatusNamespacedName {
			statuses[i] = newStatus
			found = true
			break
		}
	}

	if !found {
		statuses = append(statuses, newStatus)
	}

	v, err := json.Marshal(statuses)
	if err != nil {
		klog.Errorf("error marshalling init resource status data")
		return err
	}

	cm.Annotations[statusKey] = string(v)

	// update configmap
	err = r.LocalClient.Update(ctx, cm)
	if err != nil {
		klog.Errorf("error updating init resource status")
		return err
	}

	klog.Infof("Successfully updated init resource status")

	return nil
}
