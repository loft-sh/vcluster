package snapshot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	snapshotReconciler reconcilerKind = "snapshot"
	restoreReconciler  reconcilerKind = "restore"
)

type reconcilerKind string

func (r reconcilerKind) ToCapital() string {
	return strings.ToTitle(string(r)[:1]) + string(r[1:])
}

type reconcilerBase struct {
	vConfig            *config.VirtualClusterConfig
	requestsKubeClient client.Client
	requestsManager    ctrl.Manager
	logger             loghelper.Logger
	eventRecorder      record.EventRecorder
	isHostMode         bool
	kind               reconcilerKind
	finalizer          string
	requestKey         string
}

func (c *reconcilerBase) getRequestNamespace() string {
	if c.isHostMode {
		return c.vConfig.HostNamespace
	}
	return "kube-system"
}

func (c *reconcilerBase) client() client.Client {
	return c.requestsKubeClient
}

func (c *reconcilerBase) addFinalizer(ctx context.Context, configMap *corev1.ConfigMap) (bool, error) {
	if controllerutil.ContainsFinalizer(configMap, c.finalizer) {
		return false, nil
	}

	// before the change
	oldConfigMap := client.MergeFrom(configMap.DeepCopy())

	// add the snapshot/restore controller finalizer to the snapshot/restore request ConfigMap
	controllerutil.AddFinalizer(configMap, c.finalizer)

	// patch the object
	err := c.client().Patch(ctx, configMap, oldConfigMap)
	if err != nil {
		return false, fmt.Errorf(
			"failed to patch %s request ConfigMap %s/%s finalizers: %w",
			c.kind,
			configMap.Namespace,
			configMap.Name,
			err)
	}

	c.logger.Debugf(
		"Added vCluster %s controller finalizer %s to the %s request ConfigMap %s/%s",
		c.kind,
		c.finalizer,
		c.kind,
		configMap.Namespace,
		configMap.Name)
	return true, nil
}

func (c *reconcilerBase) removeFinalizer(ctx context.Context, configMap *corev1.ConfigMap) error {
	if !controllerutil.ContainsFinalizer(configMap, c.finalizer) {
		return nil
	}

	c.logger.Debugf(
		"Removing vCluster %s controller finalizer %s from the %s request ConfigMap %s/%s",
		c.kind,
		c.finalizer,
		c.kind,
		configMap.Namespace,
		configMap.Name)

	// before the change
	oldConfigMap := client.MergeFrom(configMap.DeepCopy())

	// add the snapshot/restore controller finalizer to the snapshot/restore request ConfigMap
	controllerutil.RemoveFinalizer(configMap, c.finalizer)

	// patch the object
	err := c.client().Patch(ctx, configMap, oldConfigMap)
	if err != nil {
		return fmt.Errorf("failed to patch %s request ConfigMap %s/%s finalizers: %w", c.kind, configMap.Namespace, configMap.Name, err)
	}

	c.logger.Debugf(
		"Removed vCluster %s controller finalizer %s from the %s request ConfigMap %s/%s",
		c.kind,
		c.finalizer,
		c.kind,
		configMap.Namespace,
		configMap.Name)
	return nil
}

// reconcileCompletedRequest cleans up the completed snapshot/restore request resources.
func (c *reconcilerBase) reconcileCompletedRequest(ctx context.Context, configMap *corev1.ConfigMap, requestMetadata RequestMetadata) error {
	c.logger.Debugf("%s request from ConfigMap %s/%s has been completed", c.kind.ToCapital(), configMap.Namespace, configMap.Name)
	err := c.reconcileDoneRequest(ctx, configMap, requestMetadata)
	if err != nil {
		return fmt.Errorf("failed to delete %s request Secret %s/%s: %w", c.kind, configMap.Namespace, configMap.Name, err)
	}
	return nil
}

// reconcileFailedRequest cleans up the failed snapshot/restore request resources.
func (c *reconcilerBase) reconcileFailedRequest(ctx context.Context, configMap *corev1.ConfigMap, requestMetadata RequestMetadata) error {
	c.logger.Errorf("%s request from ConfigMap %s/%s has failed", c.kind.ToCapital(), configMap.Namespace, configMap.Name)
	err := c.reconcileDoneRequest(ctx, configMap, requestMetadata)
	if err != nil {
		return fmt.Errorf("failed to delete %s request Secret %s/%s: %w", c.kind, configMap.Namespace, configMap.Name, err)
	}
	return nil
}

// reconcileDeletedRequest deletes the snapshot/restore request Secret and removes the finalizer from the
// snapshot/restore request ConfigMap.
func (c *reconcilerBase) reconcileDeletedRequest(ctx context.Context, configMap *corev1.ConfigMap) (retErr error) {
	// snapshot/restore request ConfigMap deleted, so delete Secret as well
	c.logger.Infof("%s request ConfigMap %s/%s deleted", c.kind.ToCapital(), configMap.Namespace, configMap.Name)
	defer func() {
		if retErr != nil {
			// an error occurred, don't remove the finalizer
			return
		}
		err := c.removeFinalizer(ctx, configMap)
		if err != nil {
			retErr = fmt.Errorf(
				"failed to remove vCluster %s controller finalizer from the %s request ConfigMap %s/%s: %w",
				c.kind,
				c.kind,
				configMap.Namespace,
				configMap.Name,
				err)
		}
	}()

	err := c.deleteRequestSecret(ctx, configMap)
	if err != nil {
		return fmt.Errorf(
			"failed to delete %s request Secret %s/%s: %w",
			c.kind,
			configMap.Namespace,
			configMap.Name,
			err)
	}
	return nil
}

// reconcileDoneRequest deletes the snapshot/restore request Secret and removes the finalizer from the
// snapshot/restore request ConfigMap.
func (c *reconcilerBase) reconcileDoneRequest(ctx context.Context, configMap *corev1.ConfigMap, requestMetadata RequestMetadata) (retErr error) {
	defer func() {
		if retErr != nil {
			// an error occurred, don't remove the finalizer
			return
		}
		err := c.removeFinalizer(ctx, configMap)
		if err != nil {
			retErr = fmt.Errorf(
				"failed to remove vCluster %s controller finalizer from the %s request ConfigMap %s/%s: %w",
				c.kind,
				c.kind,
				configMap.Namespace,
				configMap.Name,
				err)
		}
	}()

	err := c.deleteRequestSecret(ctx, configMap)
	if err != nil {
		return fmt.Errorf(
			"failed to delete %s request Secret %s/%s: %w",
			c.kind,
			configMap.Namespace,
			configMap.Name,
			err)
	}

	if time.Since(requestMetadata.CreationTimestamp.Time) >= DefaultRequestTTL {
		err = c.deleteRequestConfigMap(ctx, configMap)
		if err != nil {
			return fmt.Errorf("failed to delete %s request ConfigMap %s/%s: %w", c.kind, configMap.Namespace, configMap.Name, err)
		}
	}
	return nil
}

func (c *reconcilerBase) deleteRequestConfigMap(ctx context.Context, configMap *corev1.ConfigMap) error {
	// delete snapshot/restore request secret
	err := c.client().Delete(ctx, configMap)
	if kerrors.IsNotFound(err) {
		c.logger.Debugf("%s request ConfigMap %s/%s aleady deleted", c.kind.ToCapital(), configMap.Namespace, configMap.Name)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to delete %s request ConfigMap %s/%s: %w", c.kind, configMap.Namespace, configMap.Name, err)
	}

	c.logger.Infof("Deleted %s request ConfigMap %s/%s", c.kind, configMap.Namespace, configMap.Name)
	return nil
}

func (c *reconcilerBase) deleteRequestSecret(ctx context.Context, configMap *corev1.ConfigMap) error {
	namespace := configMap.Namespace
	name := configMap.Name
	c.logger.Debugf("Deleting %s request Secret %s/%s", c.kind, namespace, name)

	// find snapshot/restore request secret
	var secret corev1.Secret
	secretObjectKey := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	err := c.client().Get(ctx, secretObjectKey, &secret)
	if kerrors.IsNotFound(err) {
		c.logger.Debugf("%s request Secret %s/%s aleady deleted", c.kind.ToCapital(), namespace, name)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get %s request Secret %s/%s: %w", c.kind, namespace, name, err)
	}

	// delete snapshot/restore request secret
	err = c.client().Delete(ctx, &secret)
	if kerrors.IsNotFound(err) {
		c.logger.Debugf("%s request Secret %s/%s aleady deleted", c.kind.ToCapital(), namespace, name)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to delete %s request Secret %s/%s: %w", c.kind, namespace, name, err)
	}

	c.logger.Debugf("Deleted %s request Secret %s/%s", c.kind, namespace, name)
	c.eventRecorder.Eventf(configMap, corev1.EventTypeNormal, "SecretDeleted", "%s request Secret %s/%s has been deleted", c.kind.ToCapital(), configMap.Namespace, configMap.Name)
	return nil
}

func createClients(restConfig *rest.Config) (*kubernetes.Clientset, *snapshotsv1.Clientset, error) {
	if restConfig == nil {
		return nil, nil, errors.New("rest config is nil")
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create kube client: %w", err)
	}

	snapshotClient, err := snapshotsv1.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create snapshot client: %w", err)
	}

	return kubeClient, snapshotClient, nil
}
