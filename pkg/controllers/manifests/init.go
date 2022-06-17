package manifests

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/util/compress"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InitManifestSuffix     = "-init-manifests"
	LastAppliedManifestKey = "vcluster.loft.sh/last-applied-init-manifests"
	LastAppliedChartConfig = "vcluster.loft.sh/last-applied-chart-config"

	DefaultTimeOut = 120 * time.Second

	HelmWorkDir = "/tmp"
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

		return r.ProcessInitManifests(ctx, cm)
	} else if strings.HasPrefix(req.Name, "cm-"+translate.Suffix+"-chart") {
		cm, err := r.getConfigMap(ctx, req)
		if err != nil {
			return ctrl.Result{}, err
		}

		if cm == nil {
			return ctrl.Result{}, nil
		}

		return r.ProcessHelmChart(ctx, cm)
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
	for _, v := range cm.Data {
		cmData = append(cmData, v)
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
	err = r.LocalClient.Update(ctx, cm)
	if err != nil {
		r.Log.Errorf("error updating config map with last applied annotation: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *InitManifestsConfigMapReconciler) ProcessHelmChart(ctx context.Context, cm *corev1.ConfigMap) (ctrl.Result, error) {
	r.Log.Infof("processing helm chart for %s", cm.Name)

	_, ok := cm.Data["bundle"]
	if !ok {
		r.Log.Infof("chart bundle does not exist")
		// get chart bundle?

		err := r.pullChartArchive(ctx, cm)
		if err != nil {
			return ctrl.Result{}, err
		}

		// chart pulled and cached in cm successfully
		// continue processing in next reconcile
		return ctrl.Result{}, nil
	}

	name, _ := r.getChartOrReleaseDetails(ctx, cm)

	exists, err := r.releaseExists(ctx, cm)
	if err != nil {
		return ctrl.Result{}, err
	}

	if exists {
		r.Log.Infof("release for chart %s already exists", name)
		// check if upgrade is needed
		upgradedNeeded, err := r.checkIfUpgradeNeeded(ctx, cm)
		if err != nil {
			return ctrl.Result{}, err
		}

		if upgradedNeeded {
			// initiate upgrade
			err = r.initiateUpgrade(ctx, cm)
			if err != nil {
				return ctrl.Result{}, err
			}

			// register this configuration
			err = r.registerLastAppliedChartConfig(ctx, cm)
			if err != nil {
				r.Log.Errorf("error updating config map with last applied chart annotation: %v", err)
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, nil
	}

	r.Log.Infof("initaiting installation for chart %s", name)
	// initiate install
	err = r.initiateInstall(ctx, cm)
	if err != nil {
		r.Log.Errorf("error installing chart %s", name)
		return ctrl.Result{}, err
	}

	// register this configuration
	err = r.registerLastAppliedChartConfig(ctx, cm)
	if err != nil {
		r.Log.Errorf("error updating config map with last applied chart annotation: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *InitManifestsConfigMapReconciler) checkIfUpgradeNeeded(ctx context.Context, cm *corev1.ConfigMap) (bool, error) {
	name, _ := r.getChartOrReleaseDetails(ctx, cm)

	// compare with last applied config
	lastAppliedConfig, ok := cm.ObjectMeta.Annotations[LastAppliedChartConfig]
	if !ok {
		// should register current config as the last applied config?
		r.Log.Infof("release for chart %s exists but no last applied config exists", name)

		err := r.registerLastAppliedChartConfig(ctx, cm)
		if err != nil {
			r.Log.Errorf("error updating config map with last applied chart annotation: %v", err)
			return false, err
		}

		r.Log.Infof("updated current config as last applied config for chart: %s", name)
		return false, nil
	}

	currentConfig, err := r.getCompressedConfig(ctx, cm)
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

func (r *InitManifestsConfigMapReconciler) initiateUpgrade(ctx context.Context, cm *corev1.ConfigMap) error {
	name, _ := r.getChartOrReleaseDetails(ctx, cm)

	lastAppliedConfig := cm.ObjectMeta.Annotations[LastAppliedChartConfig]
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

	currentChartVersion := cm.Data["version"]
	lastVersion := lastConfig["version"]

	if currentChartVersion != lastVersion {
		err := r.pullChartArchive(ctx, cm)
		if err != nil {
			r.Log.Errorf("unable to pull chart archive for chart %s: %v", name, err)
			return err
		}

		r.Log.Infof("successfully updated the chart bundle for chart %s", name)
	}

	// start chart upgrade
	path, err := r.bundleToChart(ctx, cm)
	if err != nil {
		return err
	}
	defer os.Remove(path)

	//path := filepath.Join(chartTarball, chartName)
	namespace := cm.Data["namespace"]
	values, ok := cm.Data["values"]
	if !ok {
		r.Log.Infof("no value overrides set for chart %s", name)
	}

	r.Log.Infof("upgrading chart %s from path %s", name, path)

	timedOutContext, cancel := context.WithTimeout(ctx, r.parseTimeout(ctx, cm))
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
		rollbackErr := r.rollbackOrUninstall(ctx, name, namespace)
		if rollbackErr != nil {
			r.Log.Errorf("error while rollingback or uninstall chart %s: %v", name, err)
			return rollbackErr
		}
		return err
	}

	r.Log.Infof("successfully upgraded chart %s", name)
	return nil
}

func (r *InitManifestsConfigMapReconciler) registerLastAppliedChartConfig(ctx context.Context, cm *corev1.ConfigMap) error {
	compressedConfig, err := r.getCompressedConfig(ctx, cm)
	if err != nil {
		name, _ := r.getChartOrReleaseDetails(ctx, cm)
		r.Log.Errorf("unable to get compressed config for chart %s: %v", name, err)
		return err
	}

	if cm.ObjectMeta.Annotations == nil {
		cm.ObjectMeta.Annotations = map[string]string{}
	}

	cm.ObjectMeta.Annotations[LastAppliedChartConfig] = compressedConfig
	err = r.LocalClient.Update(ctx, cm)
	if err != nil {
		r.Log.Errorf("error updating config map with last applied chart annotation: %v", err)
		return err
	}

	return nil
}

func (r *InitManifestsConfigMapReconciler) initiateInstall(ctx context.Context, cm *corev1.ConfigMap) error {
	// initiate install
	name, namespace := r.getChartOrReleaseDetails(ctx, cm)
	path, err := r.bundleToChart(ctx, cm)
	if err != nil {
		return err
	}
	defer os.Remove(path)

	values, ok := cm.Data["values"]
	if !ok {
		r.Log.Infof("no value overrides set for chart %s", name)
	}

	r.Log.Infof("installing chart %s from path %s", name, path)

	timedOutContext, cancel := context.WithTimeout(ctx, r.parseTimeout(ctx, cm))
	defer cancel()

	err = r.HelmClient.Install(timedOutContext, name, namespace, helm.UpgradeOptions{
		Chart:           name,
		Path:            path,
		CreateNamespace: true,
		Values:          values,
		WorkDir:         HelmWorkDir,
	})

	if err != nil {
		r.Log.Errorf("unable to install chart %s: %v", name, err)
		rollbackErr := r.rollbackOrUninstall(ctx, name, namespace)
		if rollbackErr != nil {
			r.Log.Errorf("error while rollingback or uninstall chart %s: %v", name, err)
			return rollbackErr
		}
		return err
	}

	r.Log.Infof("successfully installed chart %s", name)

	return nil
}

func (r *InitManifestsConfigMapReconciler) getCompressedConfig(ctx context.Context, cm *corev1.ConfigMap) (string, error) {
	data := cm.DeepCopy().Data

	bundleHash := md5.Sum([]byte(data["bundle"]))
	data["bundleHash"] = string(bundleHash[:])

	delete(data, "bundle")

	rawData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return compress.Compress(string(rawData))
}

func (r *InitManifestsConfigMapReconciler) releaseExists(ctx context.Context, cm *corev1.ConfigMap) (bool, error) {
	name, namespace := r.getChartOrReleaseDetails(ctx, cm)

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

func (r *InitManifestsConfigMapReconciler) pullChartArchive(ctx context.Context, cm *corev1.ConfigMap) error {
	// here we always take name from chart name and not release name
	// as this is used to refer to the upstream repo
	name, ok := cm.Data["name"]
	if !ok {
		err := fmt.Errorf("chart name not found")
		r.Log.Errorf(err.Error())
		return err
	}

	repo, ok := cm.Data["repo"]
	if !ok {
		err := fmt.Errorf("repo not found in config map")
		r.Log.Errorf(err.Error())
		return err
	}

	version, ok := cm.Data["version"]
	if !ok {
		err := fmt.Errorf("chart version not found in config map")
		r.Log.Errorf(err.Error())
		return err
	}

	err := r.HelmClient.Pull(ctx, name, helm.UpgradeOptions{
		Chart:   name,
		Repo:    repo,
		Version: version,
		WorkDir: HelmWorkDir,
	})

	if err != nil {
		r.Log.Errorf("unable to pull chart %s: %v", name, err)
		return err
	}

	r.Log.Infof("successfully pulled chart %s", name)

	// update cm with chart bundle to be processed
	// during the next reconcile
	tarball := fmt.Sprintf("/%s/%s-%s.tgz", HelmWorkDir, name, version)
	rawTarBall, err := os.ReadFile(tarball)
	if err != nil {
		r.Log.Errorf("error reading in pulled archive: %v", err)
		return err
	}

	cm.Data["bundle"] = base64.StdEncoding.EncodeToString(rawTarBall)
	err = r.LocalClient.Update(ctx, cm)
	if err != nil {
		r.Log.Errorf("error updating config map with pulled bundle: %v", err)
		return err
	}

	return nil
}

func (r *InitManifestsConfigMapReconciler) bundleToChart(ctx context.Context, cm *corev1.ConfigMap) (string, error) {
	bundle := cm.Data["bundle"]

	// decode chart bundle
	tarball, err := base64.StdEncoding.DecodeString(bundle)
	if err != nil {
		r.Log.Errorf("unable to decode bundle: %v", err)
		return "", err
	}

	name, _ := r.getChartOrReleaseDetails(ctx, cm)

	filePath := fmt.Sprintf("%s/%s.tar.gz", HelmWorkDir, name)
	f, err := os.Create(filePath)
	if err != nil {
		r.Log.Errorf("unable to create tar file for helm: %v", err)
		return "", err
	}
	defer f.Close()

	_, err = f.Write(tarball)
	if err != nil {
		r.Log.Errorf("unable to write to tar file on disk: %v", err)
		return "", err
	}
	r.Log.Infof("extracted to path: %s", filePath)

	return filePath, err
}

func (r *InitManifestsConfigMapReconciler) parseTimeout(ctx context.Context, cm *corev1.ConfigMap) time.Duration {
	t := cm.Data["timeout"]

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

func (r *InitManifestsConfigMapReconciler) getChartOrReleaseDetails(ctx context.Context, cm *corev1.ConfigMap) (string, string) {
	var name, namespace string
	var ok bool

	name, ok = cm.Data["releaseName"]
	if !ok {
		name = cm.Data["name"]
	}

	namespace, ok = cm.Data["releaseNamespace"]
	if !ok {
		namespace = cm.Data["namespace"]
	}

	return name, namespace
}
