package deploy

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/k0s"
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/util/compress"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InitObjectStatus string

const (
	StatusFailed  InitObjectStatus = "Failed"
	StatusSuccess InitObjectStatus = "Success"
	StatusPending InitObjectStatus = "Pending"

	StatusKey = "vcluster.loft.sh/status"

	DefaultTimeOut = 180 * time.Second
	HelmWorkDir    = "/tmp"

	ChartPullError = "ChartPullFailed"
	InstallError   = "InstallFailed"
	UpgradeError   = "UpgradeFailed"
	UninstallError = "UninstallFailed"

	VClusterDeployConfigMap          = "vcluster-deploy"
	VClusterDeployConfigMapNamespace = "kube-system"
)

type Deployer struct {
	Log loghelper.Logger

	VirtualManager ctrl.Manager
	HelmClient     helm.Client
}

func (r *Deployer) Apply(ctx context.Context, vConfig *config.VirtualClusterConfig) (result ctrl.Result, err error) {
	// get config map
	configMap := &corev1.ConfigMap{}
	err = r.VirtualManager.GetClient().Get(ctx, types.NamespacedName{Name: VClusterDeployConfigMap, Namespace: VClusterDeployConfigMapNamespace}, configMap)
	if kerrors.IsNotFound(err) {
		if vConfig.Experimental.Deploy.VCluster.Manifests == "" && vConfig.Experimental.Deploy.VCluster.ManifestsTemplate == "" && len(vConfig.Experimental.Deploy.VCluster.Helm) == 0 {
			return ctrl.Result{}, nil
		}

		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      VClusterDeployConfigMap,
				Namespace: VClusterDeployConfigMapNamespace,
			},
		}
		err = r.VirtualManager.GetClient().Create(ctx, configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("create deploy status config map: %w", err)
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// patch the status to the configmap
	oldConfigMap := configMap.DeepCopy()
	defer func() {
		patchErr := r.UpdateConfigMap(ctx, err, result.Requeue, oldConfigMap, configMap)
		if patchErr != nil && err == nil {
			err = patchErr
		}
	}()

	// process the init manifests
	requeue, err := r.ProcessInitManifests(ctx, vConfig, configMap)
	if err != nil {
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	// process the helm charts
	requeue, err = r.ProcessHelmChart(ctx, vConfig, configMap)
	if err != nil {
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	// indicates that we have applied all manifests and charts
	return ctrl.Result{}, nil
}

func (r *Deployer) UpdateConfigMap(ctx context.Context, lastError error, requeue bool, oldConfigMap *corev1.ConfigMap, newConfigMap *corev1.ConfigMap) error {
	currentStatus := ParseStatus(newConfigMap)

	// set phase initially to pending
	currentStatus.Phase = string(StatusPending)
	currentStatus.Reason = ""
	currentStatus.Message = ""

	// check manifests status
	if currentStatus.Manifests.Phase != string(StatusSuccess) {
		currentStatus.Phase = string(StatusFailed)
		currentStatus.Reason = currentStatus.Manifests.Reason
		currentStatus.Message = currentStatus.Manifests.Message
	} else {
		// check if all charts were deployed correctly
		for _, chartStatus := range currentStatus.Charts {
			if chartStatus.Phase != string(StatusSuccess) {
				currentStatus.Phase = string(StatusFailed)
				currentStatus.Reason = chartStatus.Reason
				currentStatus.Message = chartStatus.Message
				break
			}
		}
	}

	// check if there was an error otherwise set to success
	if currentStatus.Phase == string(StatusPending) {
		if lastError == nil {
			if requeue {
				currentStatus.Phase = string(StatusPending)
			} else {
				currentStatus.Phase = string(StatusSuccess)
			}
		} else {
			currentStatus.Phase = string(StatusFailed)
			currentStatus.Reason = "Unknown"
			currentStatus.Message = lastError.Error()
		}
	}

	// marshal status
	err := r.encodeStatus(newConfigMap, currentStatus)
	if err != nil {
		return err
	}

	// create a patch
	patch := client.MergeFrom(oldConfigMap)
	rawPatch, err := patch.Data(newConfigMap)
	if err != nil {
		return err
	} else if string(rawPatch) == "{}" {
		return nil
	}

	// try patching the configmap
	r.Log.Debugf("Patch init config map with: %s", string(rawPatch))
	err = r.VirtualManager.GetClient().Patch(ctx, newConfigMap, client.RawPatch(patch.Type(), rawPatch))
	if err != nil {
		r.Log.Errorf("error updating configmap status: %v", err)
		return err
	}

	return nil
}

func (r *Deployer) ProcessInitManifests(ctx context.Context, vConfig *config.VirtualClusterConfig, configMap *corev1.ConfigMap) (bool, error) {
	var err error
	manifests := vConfig.Experimental.Deploy.VCluster.Manifests
	if vConfig.Experimental.Deploy.VCluster.ManifestsTemplate != "" {
		templatedManifests, err := k0s.ExecTemplate(vConfig.Experimental.Deploy.VCluster.ManifestsTemplate, vConfig.Name, vConfig.WorkloadTargetNamespace, &vConfig.Config)
		if err != nil {
			return false, fmt.Errorf("exec manifests template: %w", err)
		}

		manifests += "\n---\n" + string(templatedManifests)
	}

	// make array stable or otherwise order is random
	status := ParseStatus(configMap)

	// get last applied manifests
	lastAppliedManifests := ""
	if status.Manifests.LastAppliedManifests != "" {
		lastAppliedManifests, err = compress.Uncompress(status.Manifests.LastAppliedManifests)
		if err != nil {
			r.Log.Errorf("error decompressing manifests: %v", err)
			// fallthrough here as we just apply them again
		}
	}

	// should skip?
	if manifests == lastAppliedManifests {
		return false, r.setManifestsStatus(configMap, StatusSuccess, "", "")
	}

	// apply manifests
	err = ApplyGivenInitManifests(ctx, r.VirtualManager.GetClient(), r.VirtualManager.GetConfig(), manifests, lastAppliedManifests)
	if err != nil {
		r.Log.Errorf("error applying init manifests: %v", err)
		_ = r.setManifestsStatus(configMap, StatusFailed, InstallError, err.Error())
		return false, err
	}

	// apply successful, store in an annotation in the configmap itself
	compressedManifests, err := compress.Compress(manifests)
	if err != nil {
		r.Log.Errorf("error compressing manifests: %v", err)
		return false, err
	}

	// update annotation
	status.Manifests.LastAppliedManifests = compressedManifests
	err = r.encodeStatus(configMap, status)
	if err != nil {
		return false, err
	}
	return true, r.setManifestsStatus(configMap, StatusSuccess, "", "")
}

func (r *Deployer) ProcessHelmChart(ctx context.Context, vConfig *config.VirtualClusterConfig, configMap *corev1.ConfigMap) (bool, error) {
	statusMap, err := r.getStatusMap(configMap)
	if err != nil {
		return false, err
	}

	charts := vConfig.Experimental.Deploy.VCluster.Helm
	for _, chart := range charts {
		releaseName, releaseNamespace := r.getTargetRelease(chart)
		r.Log.Debugf("processing helm chart for %s/%s", releaseNamespace, releaseName)
		delete(statusMap, releaseNamespace+"/"+releaseName)

		err := r.pullChartArchive(ctx, chart)
		if err != nil {
			_ = r.setChartStatus(configMap, &chart, StatusFailed, ChartPullError, err.Error())
			return false, err
		}

		// check if we should upgrade the helm release
		exists, err := r.releaseExists(chart)
		if err != nil {
			return false, err
		} else if exists {
			r.Log.Debugf("release %s/%s already exists", releaseNamespace, releaseName)

			// check if upgrade is needed
			upgradedNeeded, err := r.checkIfUpgradeNeeded(configMap, chart)
			if err != nil {
				return false, err
			} else if upgradedNeeded {
				// initiate upgrade
				err = r.initiateUpgrade(ctx, chart)
				if err != nil {
					_ = r.setChartStatus(configMap, &chart, StatusFailed, UpgradeError, err.Error())
					return false, err
				}

				// update last applied chart config
				err = r.setChartStatusLastApplied(configMap, &chart)
				if err != nil {
					r.Log.Errorf("error updating config map with last applied chart annotation: %v", err)
					return false, err
				}

				return true, nil
			}

			// continue to process next chart
			continue
		}

		// initiate install
		r.Log.Debugf("initiating installation for release %s/%s", releaseNamespace, releaseName)
		err = r.initiateInstall(ctx, chart)
		if err != nil {
			r.Log.Errorf("error installing release %s/%s", releaseNamespace, releaseName)
			_ = r.setChartStatus(configMap, &chart, StatusFailed, InstallError, err.Error())
			return false, err
		}

		// update last applied chart config
		err = r.setChartStatusLastApplied(configMap, &chart)
		if err != nil {
			r.Log.Errorf("error updating config map with last applied chart annotation: %v", err)
			return false, err
		}

		// install only one chart successfully in each reconcile
		// hence reconcile here without error
		return true, nil
	}

	if len(statusMap) > 0 {
		r.Log.Debugf("following charts left in status map, should be deleted: %v", statusMap)
		for _, chartStatus := range statusMap {
			err := r.deleteHelmRelease(configMap, chartStatus)
			if err != nil {
				return false, errors.Wrap(err, "delete helm release")
			}
		}
	}

	return false, nil
}

func (r *Deployer) checkIfUpgradeNeeded(cm *corev1.ConfigMap, chart vclusterconfig.ExperimentalDeployHelm) (bool, error) {
	name, namespace := r.getTargetRelease(chart)

	// find chart status
	status := ParseStatus(cm)

	// check if hashed config has changed
	for _, chartStatus := range status.Charts {
		if chartStatus.Name != name || chartStatus.Namespace != namespace {
			continue
		}

		hash, _ := r.getHashedConfig(chart)
		return hash != chartStatus.LastAppliedChartConfigHash, nil
	}

	return true, nil
}

func (r *Deployer) initiateUpgrade(ctx context.Context, chart vclusterconfig.ExperimentalDeployHelm) error {
	name, namespace := r.getTargetRelease(chart)
	path, err := r.findChart(chart)
	if err != nil {
		return err
	} else if path == "" {
		return fmt.Errorf("couldn't find chart")
	}

	values := chart.Values
	if values == "" {
		r.Log.Debugf("no value overrides set for chart %s", name)
	}

	r.Log.Infof("Upgrading release %s/%s", namespace, name)

	timedOutContext, cancel := context.WithTimeout(ctx, r.parseTimeout(chart))
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
			r.Log.Errorf("error while rolling back or uninstalling chart %s: %v", name, rollbackErr)
			return rollbackErr
		}

		return err
	}

	r.Log.Infof("successfully upgraded chart %s", name)
	return nil
}

func getTarballPath(helmWorkDir, repo, name, version string) (tarballPath, tarballDir string) {
	var repoDir string
	// hashing the name so that slashes in url characters and other unaccounted-for characters
	// don't fail making the directory
	if repo != "" {
		repoDigest := sha256.Sum256([]byte(repo))
		repoDir = hex.EncodeToString(repoDigest[0:])[0:10]
	}
	// empty repoDir is ignored
	tarballDir = filepath.Join(helmWorkDir, repoDir)
	tarballPath = filepath.Join(tarballDir, fmt.Sprintf("%s-%s.tgz", name, version))

	return tarballPath, tarballDir
}

func (r *Deployer) findChart(chart vclusterconfig.ExperimentalDeployHelm) (string, error) {
	tarballPath, tarballDir := getTarballPath(HelmWorkDir, chart.Chart.Repo, chart.Chart.Name, chart.Chart.Version)
	// if version specified, look for specific file
	if chart.Chart.Version != "" {
		pathsToTry := []string{tarballPath}
		// try with alternate names as well
		if chart.Chart.Version[0] != 'v' {
			tarballPathWithV, _ := getTarballPath(HelmWorkDir, chart.Chart.Repo, chart.Chart.Name, fmt.Sprintf("v%s", chart.Chart.Version))
			pathsToTry = append(pathsToTry, tarballPathWithV)
		}
		for _, path := range pathsToTry {
			_, err := os.Stat(path)
			if err == nil {
				return path, nil
			} else if !os.IsNotExist(err) {
				return "", err
			}
		}
		// if version not specified, look for any version
	} else {
		files, err := os.ReadDir(tarballDir)
		if os.IsNotExist(err) {
			return "", nil
		} else if err != nil {
			return "", err
		}
		for _, f := range files {
			if strings.HasPrefix(f.Name(), chart.Chart.Name+"-") {
				return filepath.Join(tarballDir, f.Name()), nil
			}
		}
	}

	return "", nil
}

func (r *Deployer) initiateInstall(ctx context.Context, chart vclusterconfig.ExperimentalDeployHelm) error {
	// initiate install
	name, namespace := r.getTargetRelease(chart)
	path, err := r.findChart(chart)
	if err != nil {
		return err
	} else if path == "" {
		return fmt.Errorf("couldn't find chart")
	}

	values := chart.Values
	if values == "" {
		r.Log.Debugf("no value overrides set for chart %s", name)
	}

	r.Log.Infof("Installing release %s/%s", namespace, name)

	timedOutContext, cancel := context.WithTimeout(ctx, r.parseTimeout(chart))
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
			r.Log.Errorf("error while rollingBack or uninstall chart %s: %v", name, rollbackErr)
			return rollbackErr
		}

		return err
	}

	r.Log.Infof("successfully installed chart %s", name)
	return nil
}

func (r *Deployer) getHashedConfig(chart vclusterconfig.ExperimentalDeployHelm) (string, error) {
	rawData, err := json.Marshal(chart)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", md5.Sum(rawData)), nil
}

func (r *Deployer) releaseExists(chart vclusterconfig.ExperimentalDeployHelm) (bool, error) {
	name, namespace := r.getTargetRelease(chart)
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

func (r *Deployer) pullChartArchive(ctx context.Context, chart vclusterconfig.ExperimentalDeployHelm) error {
	tarballPath, err := r.findChart(chart)
	if err != nil {
		return err
	}

	// check if tarball exists
	if tarballPath == "" {
		tarballPath, tarballDir := getTarballPath(HelmWorkDir, chart.Chart.Repo, chart.Chart.Name, chart.Chart.Version)
		err := os.MkdirAll(tarballDir, 0755)
		if err != nil {
			return err
		}
		if chart.Bundle != "" {
			bytes, err := base64.StdEncoding.DecodeString(chart.Bundle)
			if err != nil {
				return err
			}

			err = os.WriteFile(tarballPath, bytes, 0666)
			if err != nil {
				return errors.Wrap(err, "write bundle to file")
			}
		} else {
			helmErr := r.HelmClient.Pull(ctx, chart.Chart.Name, helm.UpgradeOptions{
				Chart:    chart.Chart.Name,
				Repo:     chart.Chart.Repo,
				Insecure: chart.Chart.Insecure,
				Version:  chart.Chart.Version,
				Username: chart.Chart.Username,
				Password: chart.Chart.Password,

				WorkDir: tarballDir,
			})
			if helmErr != nil {
				r.Log.Errorf("unable to pull chart %s: %v", chart.Chart.Name, helmErr)
				return helmErr
			}

			r.Log.Debugf("successfully pulled chart %s", chart.Chart.Name)
		}
	}

	return nil
}

func (r *Deployer) parseTimeout(chart vclusterconfig.ExperimentalDeployHelm) time.Duration {
	t := chart.Timeout
	timeout, err := time.ParseDuration(t)
	if err != nil {
		r.Log.Debugf("unable to parse timeout: %v", err)
		r.Log.Debugf("falling back to default timeout")
		timeout = DefaultTimeOut
	}

	return timeout
}

func (r *Deployer) rollbackOrUninstall(ctx context.Context, chartName, namespace string) error {
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

func (r *Deployer) getTargetRelease(chart vclusterconfig.ExperimentalDeployHelm) (string, string) {
	name := chart.Chart.Name
	namespace := corev1.NamespaceDefault
	if chart.Release.Name != "" {
		name = chart.Release.Name
	}
	if chart.Release.Namespace != "" {
		namespace = chart.Release.Namespace
	}

	return name, namespace
}

func (r *Deployer) getStatusMap(cm *corev1.ConfigMap) (map[string]ChartStatus, error) {
	statusMap := make(map[string]ChartStatus)
	status := ParseStatus(cm)
	for _, status := range status.Charts {
		statusMap[status.Namespace+"/"+status.Name] = status
	}

	return statusMap, nil
}

func (r *Deployer) encodeStatus(cm *corev1.ConfigMap, status *Status) error {
	if cm.Annotations == nil {
		cm.Annotations = map[string]string{}
	}

	marshalled, err := json.Marshal(status)
	if err != nil {
		r.Log.Errorf("error marshal status: %v", err)
		return err
	}
	cm.Annotations[StatusKey] = string(marshalled)
	return nil
}

func ParseStatus(cm *corev1.ConfigMap) *Status {
	status := &Status{}
	if cm.Annotations[StatusKey] != "" {
		err := yaml.Unmarshal([]byte(cm.Annotations[StatusKey]), &status)
		if err != nil {
			klog.Errorf("error unmarshalling rawStatus: %v", err)
			return &Status{}
		}
	}

	return status
}

func (r *Deployer) setManifestsStatus(cm *corev1.ConfigMap, phase InitObjectStatus, reason string, message string) error {
	status := ParseStatus(cm)
	status.Manifests.Phase = string(phase)
	status.Manifests.Reason = reason
	status.Manifests.Message = message
	return r.encodeStatus(cm, status)
}

func (r *Deployer) setChartStatusLastApplied(cm *corev1.ConfigMap, chart *vclusterconfig.ExperimentalDeployHelm) error {
	status := ParseStatus(cm)

	// get release name & namespace
	releaseName, releaseNamespace := r.getTargetRelease(*chart)
	found := false

	// find the correct chart in the array and update it
	hashedConfig, _ := r.getHashedConfig(*chart)
	for i, releaseStatus := range status.Charts {
		if releaseStatus.Name == releaseName && releaseStatus.Namespace == releaseNamespace {
			status.Charts[i].LastAppliedChartConfigHash = hashedConfig
			status.Charts[i].Phase = string(StatusSuccess)
			status.Charts[i].Reason = ""
			status.Charts[i].Message = ""
			found = true
			break
		}
	}
	if !found {
		status.Charts = append(status.Charts, ChartStatus{
			Name:                       releaseName,
			Namespace:                  releaseNamespace,
			LastAppliedChartConfigHash: hashedConfig,
			Phase:                      string(StatusSuccess),
			Reason:                     "",
			Message:                    "",
		})
	}

	return r.encodeStatus(cm, status)
}

func (r *Deployer) setChartStatus(cm *corev1.ConfigMap, chart *vclusterconfig.ExperimentalDeployHelm, phase InitObjectStatus, reason string, message string) error {
	status := ParseStatus(cm)

	// get release name & namespace
	releaseName, releaseNamespace := r.getTargetRelease(*chart)
	found := false

	// find the correct chart in the array and update it
	for i, releaseStatus := range status.Charts {
		if releaseStatus.Name == releaseName && releaseStatus.Namespace == releaseNamespace {
			status.Charts[i].Phase = string(phase)
			status.Charts[i].Reason = reason
			status.Charts[i].Message = message
			found = true
			break
		}
	}
	if !found {
		status.Charts = append(status.Charts, ChartStatus{
			Name:      releaseName,
			Namespace: releaseNamespace,
			Phase:     string(phase),
			Reason:    reason,
			Message:   message,
		})
	}

	return r.encodeStatus(cm, status)
}

func (r *Deployer) popFromStatus(cm *corev1.ConfigMap, chartStatus ChartStatus) error {
	status := ParseStatus(cm)

	found := -1
	for i, status := range status.Charts {
		if status.Name == chartStatus.Name && status.Namespace == chartStatus.Namespace {
			found = i
			break
		}
	}
	status.Charts = append(status.Charts[:found], status.Charts[found+1:]...)

	return r.encodeStatus(cm, status)
}

func (r *Deployer) deleteHelmRelease(cm *corev1.ConfigMap, chartStatus ChartStatus) error {
	err := r.HelmClient.Delete(chartStatus.Name, chartStatus.Namespace)
	if err != nil {
		r.Log.Infof("error deleting helm release %s/%s: %v", chartStatus.Namespace, chartStatus.Name, err)
		return r.setChartStatus(cm, &vclusterconfig.ExperimentalDeployHelm{
			Chart: vclusterconfig.ExperimentalDeployHelmChart{
				Name: chartStatus.Name,
			},
			Release: vclusterconfig.ExperimentalDeployHelmRelease{
				Namespace: chartStatus.Namespace,
			},
		}, StatusFailed, UninstallError, err.Error())
	}

	return r.popFromStatus(cm, chartStatus)
}
