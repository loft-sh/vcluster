package manifests

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/klog"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/util/compress"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InitObjectStatus string

const (
	InitChartsKey      = "charts"
	InitManifestsKey   = "manifests"
	InitManifestSuffix = "-init-manifests"

	StatusFailed  InitObjectStatus = "Failed"
	StatusSuccess InitObjectStatus = "Success"
	StatusPending InitObjectStatus = "Pending"
	StatusKey                      = "vcluster.loft.sh/status"

	DefaultTimeOut = 180 * time.Second
	HelmWorkDir    = "/tmp"

	ChartPullError = "ChartPullFailed"
	InstallError   = "InstallFailed"
	UpgradeError   = "UpgradeFailed"
	UninstallError = "UninstallFailed"
)

type InitManifestsConfigMapReconciler struct {
	Log loghelper.Logger

	LocalClient    client.Client
	VirtualManager ctrl.Manager

	HelmClient helm.Client
}

func (r *InitManifestsConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	// TODO: implement better filtration through predicates
	if req.Name != translate.Suffix+InitManifestSuffix {
		return ctrl.Result{}, nil
	}

	r.Log.Debugf("process init manifests")

	// get config map
	cm := &corev1.ConfigMap{}
	err = r.LocalClient.Get(ctx, req.NamespacedName, cm)
	if err != nil {
		return ctrl.Result{}, err
	}

	// patch the status to the configmap
	oldConfigMap := cm.DeepCopy()
	defer func() {
		patchErr := r.UpdateConfigMap(ctx, err, result.Requeue, oldConfigMap, cm)
		if patchErr != nil && err == nil {
			err = patchErr
		}
	}()

	// process the init manifests
	requeue, err := r.ProcessInitManifests(ctx, cm)
	if err != nil {
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	// process the helm charts
	requeue, err = r.ProcessHelmChart(ctx, cm)
	if err != nil {
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	// indicates that we have applied all manifests and charts
	return ctrl.Result{}, nil
}

func (r *InitManifestsConfigMapReconciler) UpdateConfigMap(ctx context.Context, lastError error, requeue bool, oldConfigMap *corev1.ConfigMap, newConfigMap *corev1.ConfigMap) error {
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
	err = r.LocalClient.Patch(ctx, newConfigMap, client.RawPatch(patch.Type(), rawPatch))
	if err != nil {
		r.Log.Errorf("error updating configmap status: %v", err)
		return err
	}

	return nil
}

func (r *InitManifestsConfigMapReconciler) ProcessInitManifests(ctx context.Context, cm *corev1.ConfigMap) (bool, error) {
	var err error
	manifests := cm.Data[InitManifestsKey]

	// make array stable or otherwise order is random
	status := ParseStatus(cm)

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
		return false, r.setManifestsStatus(cm, StatusSuccess, "", "")
	}

	// apply manifests
	err = ApplyGivenInitManifests(ctx, r.VirtualManager.GetClient(), r.VirtualManager.GetConfig(), manifests, lastAppliedManifests)
	if err != nil {
		r.Log.Errorf("error applying init manifests: %v", err)
		_ = r.setManifestsStatus(cm, StatusFailed, InstallError, err.Error())
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
	err = r.encodeStatus(cm, status)
	if err != nil {
		return false, err
	}
	return true, r.setManifestsStatus(cm, StatusSuccess, "", "")
}

func (r *InitManifestsConfigMapReconciler) ProcessHelmChart(ctx context.Context, cm *corev1.ConfigMap) (bool, error) {
	statusMap, err := r.getStatusMap(cm)
	if err != nil {
		return false, err
	}

	var charts []Chart
	if cm.Data[InitChartsKey] != "" {
		err = yaml.Unmarshal([]byte(cm.Data[InitChartsKey]), &charts)
		if err != nil {
			return false, err
		}
	}

	for _, chart := range charts {
		releaseName, releaseNamespace := r.getTargetRelease(chart)
		r.Log.Debugf("processing helm chart for %s/%s", releaseNamespace, releaseName)
		delete(statusMap, releaseNamespace+"/"+releaseName)

		err := r.pullChartArchive(ctx, chart)
		if err != nil {
			_ = r.setChartStatus(cm, &chart, StatusFailed, ChartPullError, err.Error())
			return false, err
		}

		// check if we should upgrade the helm release
		exists, err := r.releaseExists(chart)
		if err != nil {
			return false, err
		} else if exists {
			r.Log.Debugf("release %s/%s already exists", releaseNamespace, releaseName)

			// check if upgrade is needed
			upgradedNeeded, err := r.checkIfUpgradeNeeded(cm, chart)
			if err != nil {
				return false, err
			} else if upgradedNeeded {
				// initiate upgrade
				err = r.initiateUpgrade(ctx, chart)
				if err != nil {
					_ = r.setChartStatus(cm, &chart, StatusFailed, UpgradeError, err.Error())
					return false, err
				}

				// update last applied chart config
				err = r.setChartStatusLastApplied(cm, &chart)
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
			_ = r.setChartStatus(cm, &chart, StatusFailed, InstallError, err.Error())
			return false, err
		}

		// update last applied chart config
		err = r.setChartStatusLastApplied(cm, &chart)
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
			err := r.deleteHelmRelease(cm, chartStatus)
			if err != nil {
				return false, errors.Wrap(err, "delete helm release")
			}
		}
	}

	return false, nil
}

func (r *InitManifestsConfigMapReconciler) checkIfUpgradeNeeded(cm *corev1.ConfigMap, chart Chart) (bool, error) {
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

func (r *InitManifestsConfigMapReconciler) initiateUpgrade(ctx context.Context, chart Chart) error {
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

func (r *InitManifestsConfigMapReconciler) findChart(chart Chart) (string, error) {
	if chart.Version == "" {
		files, err := os.ReadDir(HelmWorkDir)
		if err != nil {
			return "", err
		}

		for _, f := range files {
			if strings.HasPrefix(f.Name(), chart.Name+"-") {
				return f.Name(), nil
			}
		}
	} else {
		tarball := fmt.Sprintf("%s/%s-%s.tgz", HelmWorkDir, chart.Name, chart.Version)
		_, err := os.Stat(tarball)
		if err != nil {
			if os.IsNotExist(err) {
				if chart.Version[0] != 'v' {
					tarball = fmt.Sprintf("%s/%s-v%s.tgz", HelmWorkDir, chart.Name, chart.Version)
					_, err = os.Stat(tarball)
					if err == nil {
						return tarball, nil
					}
				}

				return "", nil
			}

			return "", err
		}

		return tarball, nil
	}

	return "", nil
}

func (r *InitManifestsConfigMapReconciler) initiateInstall(ctx context.Context, chart Chart) error {
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

func (r *InitManifestsConfigMapReconciler) getHashedConfig(chart Chart) (string, error) {
	rawData, err := json.Marshal(chart)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", md5.Sum(rawData)), nil
}

func (r *InitManifestsConfigMapReconciler) releaseExists(chart Chart) (bool, error) {
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

func (r *InitManifestsConfigMapReconciler) pullChartArchive(ctx context.Context, chart Chart) error {
	tarball, err := r.findChart(chart)
	if err != nil {
		return err
	}

	// check if tarball exists
	if tarball == "" {
		if chart.Bundle != "" {
			bytes, err := base64.StdEncoding.DecodeString(chart.Bundle)
			if err != nil {
				return err
			}

			err = os.WriteFile(tarball, bytes, 0666)
			if err != nil {
				return errors.Wrap(err, "write bundle to file")
			}
		} else {
			helmErr := r.HelmClient.Pull(ctx, chart.Name, helm.UpgradeOptions{
				Chart:    chart.Name,
				Repo:     chart.Repo,
				Insecure: chart.Insecure,
				Version:  chart.Version,
				Username: chart.Username,
				Password: chart.Password,

				WorkDir: HelmWorkDir,
			})
			if helmErr != nil {
				r.Log.Errorf("unable to pull chart %s: %v", chart.Name, helmErr)
				return helmErr
			}

			r.Log.Debugf("successfully pulled chart %s", chart.Name)
		}
	}

	return nil
}

func (r *InitManifestsConfigMapReconciler) parseTimeout(chart Chart) time.Duration {
	t := chart.Timeout
	timeout, err := time.ParseDuration(t)
	if err != nil {
		r.Log.Debugf("unable to parse timeout: %v", err)
		r.Log.Debugf("falling back to default timeout")
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

func (r *InitManifestsConfigMapReconciler) getTargetRelease(chart Chart) (string, string) {
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

func (r *InitManifestsConfigMapReconciler) getStatusMap(cm *corev1.ConfigMap) (map[string]ChartStatus, error) {
	statusMap := make(map[string]ChartStatus)
	status := ParseStatus(cm)
	for _, status := range status.Charts {
		statusMap[status.Namespace+"/"+status.Name] = status
	}

	return statusMap, nil
}

func (r *InitManifestsConfigMapReconciler) encodeStatus(cm *corev1.ConfigMap, status *Status) error {
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

func (r *InitManifestsConfigMapReconciler) setManifestsStatus(cm *corev1.ConfigMap, phase InitObjectStatus, reason string, message string) error {
	status := ParseStatus(cm)
	status.Manifests.Phase = string(phase)
	status.Manifests.Reason = reason
	status.Manifests.Message = message
	return r.encodeStatus(cm, status)
}

func (r *InitManifestsConfigMapReconciler) setChartStatusLastApplied(cm *corev1.ConfigMap, chart *Chart) error {
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

func (r *InitManifestsConfigMapReconciler) setChartStatus(cm *corev1.ConfigMap, chart *Chart, phase InitObjectStatus, reason string, message string) error {
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

func (r *InitManifestsConfigMapReconciler) popFromStatus(cm *corev1.ConfigMap, chartStatus ChartStatus) error {
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

func (r *InitManifestsConfigMapReconciler) deleteHelmRelease(cm *corev1.ConfigMap, chartStatus ChartStatus) error {
	err := r.HelmClient.Delete(chartStatus.Name, chartStatus.Namespace)
	if err != nil {
		r.Log.Infof("error deleting helm release %s/%s: %v", chartStatus.Namespace, chartStatus.Name, err)
		return r.setChartStatus(cm, &Chart{
			Name:             chartStatus.Name,
			ReleaseNamespace: chartStatus.Namespace,
		}, StatusFailed, UninstallError, err.Error())
	}

	return r.popFromStatus(cm, chartStatus)
}

func (r *InitManifestsConfigMapReconciler) SetupWithManager(hostMgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(hostMgr).
		Named("init_manifests").
		For(&corev1.ConfigMap{}).
		Complete(r)
}
