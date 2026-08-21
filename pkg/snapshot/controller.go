package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ControllerFinalizer = "vcluster.loft.sh/snapshot-controller"
	controllerName      = "vcluster-snapshot-controller"
)

type Reconciler struct {
	reconcilerBase
	vConfig                    *config.VirtualClusterConfig
	snapshotRequestsKubeClient client.Client
	snapshotRequestsManager    ctrl.Manager
	logger                     loghelper.Logger
	eventRecorder              events.EventRecorder
	isHostMode                 bool
	// briefly caches the credentials pulled from the platform. One cell, not a map: credentials are per
	// instance and this controller serves exactly one.
	snapshotCredentials atomic.Pointer[cachedSnapshotOptions]
}

const snapshotCredentialsCacheTTL = time.Minute

// snapshotCredentialsStaleTTL bounds how long the last successful pull may still be used once the
// platform stops answering. A backup with slightly stale credentials beats no backup: rotated ones are
// rejected by the object store, which fails the request with the real reason. A refusal is terminal and
// never falls back, so the platform can stop this the moment it answers.
const snapshotCredentialsStaleTTL = 24 * time.Hour * 7

type cachedSnapshotOptions struct {
	options *snapshotapi.Options
	expiry  time.Time
	// staleExpiry is when the entry stops being usable even as a fallback.
	staleExpiry time.Time
}

// credentialResolveDeadline bounds how long a request may keep failing to pull its storage
// credentials before it is failed instead of requeued. The pull is a single call to the platform, so a
// failure lasting this long is not transient: the token has no access to the instance, the license
// lapsed, or the instance never registered. Without a bound the request stays in a non-terminal phase
// forever, and because the platform blocks scheduling while anything is in progress, that instance
// silently stops taking snapshots altogether. Failing here puts the real reason in the request's
// Status.Error instead of leaving the platform to report a generic timeout.
const credentialResolveDeadline = 10 * time.Minute

// resolveSnapshotOptions returns the snapshot storage options (location + credentials) for a request.
// The pushed options Secret wins whenever there is one, because a request the tenant created itself
// carries its own credentials; only a platform-scheduled request on a standalone tenant arrives without
// one, and there they are pulled from the platform. The returned requeue flag signals a transient
// condition where the caller must retry rather than fail the snapshot.
func (c *Reconciler) resolveSnapshotOptions(ctx context.Context, configMap *corev1.ConfigMap, snapshotRequest *snapshotapi.Request) (*snapshotapi.Options, bool, error) {
	var secret corev1.Secret
	err := c.client().Get(ctx, client.ObjectKey{Namespace: configMap.Namespace, Name: configMap.Name}, &secret)
	switch {
	case err == nil:
		options, unmarshalErr := snapshotapi.UnmarshalOptions(&secret)
		if unmarshalErr != nil {
			return nil, false, fmt.Errorf("failed to unmarshal vcluster snapshot options from Secret %s/%s: %w", secret.Namespace, secret.Name, unmarshalErr)
		}
		return options, false, nil
	case !kerrors.IsNotFound(err):
		return nil, false, fmt.Errorf("failed to get snapshot request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	if c.vConfig != nil && c.vConfig.ControlPlane.Standalone.Enabled && pro.ResolveSnapshotCredentials != nil {
		return c.pullSnapshotOptions(ctx, configMap, snapshotRequest)
	}

	// Too soon and the client cache is not up to date? Requeue if the request was created recently.
	if time.Since(configMap.CreationTimestamp.Time) < 10*time.Second {
		return nil, true, nil
	}
	return nil, false, fmt.Errorf("can't find snapshot request Secret %s/%s: %w", configMap.Namespace, configMap.Name, err)
}

// pullSnapshotOptions builds the options for a request that carries no Secret: the credentials come from
// the platform (in memory, briefly cached), the location from the request's non-secret URL.
func (c *Reconciler) pullSnapshotOptions(ctx context.Context, configMap *corev1.ConfigMap, snapshotRequest *snapshotapi.Request) (*snapshotapi.Options, bool, error) {
	location := &snapshotapi.Options{}
	if err := Parse(snapshotRequest.Spec.URL, location); err != nil {
		return nil, false, fmt.Errorf("failed to parse snapshot URL %q: %w", snapshotRequest.Spec.URL, err)
	}

	credentials, err := c.cachedSnapshotCredentials(ctx)
	if err != nil {
		if terminalErr := terminalCredentialError(err); terminalErr != nil {
			return nil, false, terminalErr
		}
		// An unreachable platform must not cost a backup we already hold the credentials for. This runs
		// only after the refusals above, so the platform can still stop the pull whenever it answers.
		stale, ok := c.staleSnapshotCredentials()
		if !ok {
			// The ConfigMap timestamp is set by the API server, so the deadline holds across restarts of this
			// controller without tracking attempts.
			age := time.Since(configMap.CreationTimestamp.Time)
			if !configMap.CreationTimestamp.IsZero() && age > credentialResolveDeadline {
				return nil, false, fmt.Errorf("could not resolve snapshot credentials from platform after %s: %w", age.Round(time.Second), err)
			}
			// transient (platform unreachable / not registered yet): requeue, do not fail
			c.logger.Infof("could not resolve snapshot credentials from platform yet, requeueing: %v", err)
			return nil, true, nil
		}
		c.logger.Infof("could not reach the platform for snapshot credentials, continuing with the last ones it supplied: %v", err)
		credentials = stale
	}

	// the credentials belong to the platform's configured backend, so a request aimed elsewhere would
	// authenticate against the wrong one instead of failing
	if credentials.Type != location.Type {
		return nil, false, fmt.Errorf("snapshot request targets %q storage, but the platform supplied credentials for %q", location.Type, credentials.Type)
	}

	options := *credentials
	if err := overlaySnapshotLocation(&options, location); err != nil {
		return nil, false, err
	}
	return &options, false, nil
}

// terminalCredentialError reports why a failed pull will not succeed on retry, or nil when it might. The
// platform answers NotFound when the instance has no snapshot storage configured and Forbidden when the
// license lapsed or the token may not use the instance; neither changes while the request waits.
func terminalCredentialError(err error) error {
	switch {
	case kerrors.IsNotFound(err):
		return fmt.Errorf("the platform has no snapshot storage configured for this virtual cluster, so create the snapshot request with its own options: %w", err)
	case kerrors.IsForbidden(err):
		return fmt.Errorf("the platform refused to supply snapshot credentials: %w", err)
	case errors.Is(err, pro.ErrSnapshotCredentialsMalformed):
		return err
	}

	return nil
}

// errCredentialsNotProvisioned reports an answer that parsed but named no backend, which is how an
// instance looks before its storage is configured. Transient: a zero-value Options would otherwise read
// as a backend mismatch and fail the request on a condition that clears itself.
var errCredentialsNotProvisioned = errors.New("the platform has not provisioned snapshot storage for this virtual cluster yet")

// cachedSnapshotCredentials pulls this instance's snapshot storage credentials from the platform,
// caching them briefly. Credentials are per instance (not per snapshot), so a single cache entry
// is sufficient.
func (c *Reconciler) cachedSnapshotCredentials(ctx context.Context) (*snapshotapi.Options, error) {
	if cached := c.snapshotCredentials.Load(); cached != nil && time.Now().Before(cached.expiry) {
		return cached.options, nil
	}
	credentials, err := pro.ResolveSnapshotCredentials(ctx)
	if err != nil {
		return nil, err
	}
	// not cached: storing it would make every request in the cache window inherit the same verdict
	if credentials == nil || credentials.Type == "" {
		return nil, errCredentialsNotProvisioned
	}
	now := time.Now()
	c.snapshotCredentials.Store(&cachedSnapshotOptions{
		options:     credentials,
		expiry:      now.Add(snapshotCredentialsCacheTTL),
		staleExpiry: now.Add(snapshotCredentialsStaleTTL),
	})
	return credentials, nil
}

// staleSnapshotCredentials returns the last successful pull when it may still be used as a fallback.
// The entry's expiry is deliberately not extended, so every later request retries the platform first
// and stops falling back as soon as one succeeds.
func (c *Reconciler) staleSnapshotCredentials() (*snapshotapi.Options, bool) {
	cached := c.snapshotCredentials.Load()
	if cached == nil || !time.Now().Before(cached.staleExpiry) {
		return nil, false
	}

	return cached.options, true
}

// overlaySnapshotLocation copies the per-snapshot location from src onto dst, keeping dst's connection
// settings and credentials. The request may choose where inside the platform's backend a snapshot lands,
// never which host: dst carries live credentials, and for OCI and Azure the location field is the host.
// S3 keeps dst's endpoint and container is a local path, so neither needs the check.
func overlaySnapshotLocation(dst, src *snapshotapi.Options) error {
	switch dst.Type {
	case "s3":
		dst.S3.Bucket = src.S3.Bucket
		dst.S3.Key = src.S3.Key
	case "container":
		dst.Container.Path = src.Container.Path
	case "oci":
		if err := sameHost(ociHost(dst.OCI.Repository), ociHost(src.OCI.Repository)); err != nil {
			return fmt.Errorf("snapshot request targets a different OCI registry than the platform supplied credentials for: %w", err)
		}
		dst.OCI.Repository = src.OCI.Repository
	case "azure":
		dstHost, err := blobHost(dst.Azure.BlobURL)
		if err != nil {
			return fmt.Errorf("parse the platform's Azure blob URL: %w", err)
		}
		srcHost, err := blobHost(src.Azure.BlobURL)
		if err != nil {
			return fmt.Errorf("parse the snapshot request's Azure blob URL: %w", err)
		}
		if err := sameHost(dstHost, srcHost); err != nil {
			return fmt.Errorf("snapshot request targets a different Azure host than the platform supplied credentials for: %w", err)
		}
		dst.Azure.BlobURL = src.Azure.BlobURL
	default:
		return fmt.Errorf("unsupported snapshot storage type %q", dst.Type)
	}

	return nil
}

// ociHost returns the registry from an OCI repository, which Parse builds as host/path.
func ociHost(repository string) string {
	host, _, _ := strings.Cut(repository, "/")

	return host
}

// blobHost returns the host of an Azure blob URL.
func blobHost(blobURL string) (string, error) {
	parsed, err := url.Parse(blobURL)
	if err != nil {
		return "", err
	}

	return parsed.Host, nil
}

func sameHost(platform, request string) error {
	if platform == "" {
		return errors.New("the platform supplied no host to compare against")
	}
	if !strings.EqualFold(platform, request) {
		return fmt.Errorf("request names %q, platform credentials are for %q", request, platform)
	}

	return nil
}

func NewController(registerContext *synccontext.RegisterContext) (*Reconciler, error) {
	logger := loghelper.New(controllerName)

	if registerContext == nil {
		return nil, errors.New("register context is nil")
	}
	if registerContext.Config == nil {
		return nil, errors.New("virtual cluster config is nil")
	}
	isHostMode, err := IsSnapshotRequestCreatedInHostCluster(registerContext.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to check if snapshot request is created in host cluster: %w", err)
	}

	var snapshotRequestsManager ctrl.Manager
	if isHostMode {
		snapshotRequestsManager = registerContext.HostManager
		logger.Infof("vcluster-snapshot-controller will reconcile snapshot requests in the host cluster")
	} else {
		snapshotRequestsManager = registerContext.VirtualManager
		logger.Infof("vcluster-snapshot-controller will reconcile snapshot requests in the virtual cluster")
	}
	eventRecorder := snapshotRequestsManager.GetEventRecorder(controllerName)

	reconciler := reconcilerBase{
		vConfig:            registerContext.Config,
		requestsKubeClient: snapshotRequestsManager.GetClient(),
		requestsManager:    snapshotRequestsManager,
		logger:             logger,
		eventRecorder:      eventRecorder,
		isHostMode:         isHostMode,
		kind:               snapshotReconciler,
		finalizer:          ControllerFinalizer,
		requestKey:         snapshotapi.RequestKey,
	}
	return &Reconciler{
		reconcilerBase:             reconciler,
		vConfig:                    registerContext.Config,
		snapshotRequestsKubeClient: snapshotRequestsManager.GetClient(),
		snapshotRequestsManager:    snapshotRequestsManager,
		logger:                     logger,
		eventRecorder:              eventRecorder,
		isHostMode:                 isHostMode,
	}, nil
}

func (c *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, retErr error) {
	c.logger.Debugf("Reconciling snapshot request ConfigMap %s", req.NamespacedName)

	var configMap corev1.ConfigMap
	err := c.client().Get(ctx, req.NamespacedName, &configMap)
	if kerrors.IsNotFound(err) {
		c.logger.Debugf("Snapshot request ConfigMap %s not found", req.NamespacedName)
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get snapshot request ConfigMap %s/%s: %w", req.Namespace, req.Name, err)
	}
	c.logger.Debugf("Found ConfigMap %s/%s with vcluster snapshot request", configMap.Namespace, configMap.Name)

	// Snapshot request ConfigMap deleted -> we've got some cleaning up to do 🧹
	if !configMap.DeletionTimestamp.IsZero() {
		err = c.reconcileDeletedRequest(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile deletion of snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		return ctrl.Result{}, nil
	}

	// Extract snapshot request details from the ConfigMap and the Secret 🔎
	snapshotRequest, err := snapshotapi.UnmarshalRequest(&configMap)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}

	// Not done? Add the finalizer if it's not already set! 🔒
	if !snapshotRequest.Done() {
		updated, err := c.addFinalizer(ctx, &configMap)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add vCluster snapshot controller finalizer to the snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		if updated {
			c.eventRecorder.Eventf(
				&configMap,
				nil,
				corev1.EventTypeNormal,
				"Created",
				"CreateSnapShotRequest",
				"Snapshot request %s/%s has been created",
				configMap.Namespace,
				configMap.Name,
			)
			return ctrl.Result{}, nil
		}
	}
	canContinue, err := c.cancelPreviousRequests(ctx, snapshotRequest)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to cancel previous snapshot requests: %w", err)
	}
	if !canContinue {
		return ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	// patch snapshot request ConfigMap after the reconciliation
	configMapBeforeChange := client.MergeFrom(configMap.DeepCopy())
	defer func() {
		if retErr != nil {
			// something went wrong, recorde error and update snapshot request phase to Failed
			snapshotRequest.Status.Phase = snapshotapi.RequestPhaseFailed
			snapshotRequest.Status.Error.Message = retErr.Error()
			c.eventRecorder.Eventf(
				&configMap,
				nil,
				corev1.EventTypeWarning,
				"SnapshotRequestFailed",
				"ReconcileSnapShotRequest",
				"Snapshot request %s/%s has failed with error: %v",
				configMap.Namespace,
				configMap.Name, retErr)
		}
		updateErr := c.updateRequest(ctx, configMapBeforeChange, &configMap, *snapshotRequest)
		if updateErr != nil {
			retErr = fmt.Errorf("failed to update snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, updateErr)
		}
		if retErr != nil {
			retErr = errors.Join(retErr, updateErr)
		} else {
			retErr = updateErr
		}
	}()

	switch snapshotRequest.Status.Phase {
	case snapshotapi.RequestPhaseNotStarted:
		err = c.reconcileNewRequest(ctx, &configMap, snapshotRequest)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile new snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case snapshotapi.RequestPhaseCanceling:
		snapshotRequest.Status.Phase = snapshotRequest.Status.Phase.Next()
		return ctrl.Result{}, nil
	case snapshotapi.RequestPhaseCreatingEtcdBackup:
		requeue, err := c.reconcileCreatingEtcdBackup(ctx, &configMap, snapshotRequest)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile etcd backup creation for snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		if requeue {
			return ctrl.Result{
				RequeueAfter: 10 * time.Second,
			}, nil
		}
	case snapshotapi.RequestPhaseCanceled:
		fallthrough
	case snapshotapi.RequestPhaseDeleted:
		fallthrough
	case snapshotapi.RequestPhasePartiallyFailed:
		fallthrough
	case snapshotapi.RequestPhaseCompleted:
		err = c.reconcileCompletedRequest(ctx, &configMap, snapshotRequest.RequestMetadata)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile completed snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case snapshotapi.RequestPhaseFailed:
		err = c.reconcileFailedRequest(ctx, &configMap, snapshotRequest.RequestMetadata)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile failed snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case snapshotapi.RequestPhaseDeleting:
		err = c.reconcileDeleting(ctx, &configMap, snapshotRequest)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile snapshot deletion request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
	case snapshotapi.RequestPhaseDeletingEtcdBackup:
		requeue, err := c.reconcileDeletingEtcdBackup(ctx, &configMap, snapshotRequest)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile snapshot deletion request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		if requeue {
			return ctrl.Result{
				RequeueAfter: 10 * time.Second,
			}, nil
		}
	default:
		return ctrl.Result{}, fmt.Errorf("invalid snapshot request phase %s", snapshotRequest.Status.Phase)
	}

	return ctrl.Result{}, nil
}

func (c *Reconciler) Register() error {
	isSnapshotRequestConfig := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		if obj.GetNamespace() != c.getRequestNamespace() {
			return false
		}

		objLabels := obj.GetLabels()
		if objLabels == nil {
			return false
		}
		_, ok := objLabels[constants.SnapshotRequestLabel]
		return ok
	})

	return ctrl.NewControllerManagedBy(c.snapshotRequestsManager).
		WithOptions(controller.Options{
			CacheSyncTimeout:        constants.DefaultCacheSyncTimeout,
			MaxConcurrentReconciles: 1,
		}).
		Named("snapshot-requests-controller").
		For(&corev1.ConfigMap{}, builder.WithPredicates(isSnapshotRequestConfig)).
		Complete(c)
}

// reconcileNewRequest updates the snapshot request phase to "InProgress".
func (c *Reconciler) reconcileNewRequest(_ context.Context, configMap *corev1.ConfigMap, snapshotRequest *snapshotapi.Request) error {
	snapshotRequest.Status.Phase = snapshotapi.RequestPhaseCreatingEtcdBackup
	c.eventRecorder.Eventf(
		configMap,
		nil,
		corev1.EventTypeNormal,
		"CreatingEtcdBackup",
		"ReconcileSnapShotRequest",
		"Started to create etcd backup for snapshot request %s/%s",
		configMap.Namespace,
		configMap.Name,
	)
	return nil
}

// reconcileCreatingEtcdBackup creates the snapshot, uploads it to the specified storage, and updates
// the snapshot request phase to "Completed".
func (c *Reconciler) reconcileCreatingEtcdBackup(ctx context.Context, configMap *corev1.ConfigMap, snapshotRequest *snapshotapi.Request) (bool, error) {
	// Obtain the snapshot options (URL + credentials). Standalone, platform-connected tenants pull
	// them from the platform in memory (credentials are never pushed into the tenant); everyone
	// else reads them from the pushed request Secret.
	snapshotOptions, requeue, err := c.resolveSnapshotOptions(ctx, configMap, snapshotRequest)
	if err != nil {
		return false, err
	}
	if requeue {
		return true, nil
	}
	snapshotRequest.Spec.Options = *snapshotOptions

	// Create and save the snapshot! 💾
	c.logger.Infof("Creating vCluster snapshot in storage type %q", snapshotOptions.Type)
	snapshotClient := &Client{
		Request: snapshotRequest,
		Options: *snapshotOptions,
	}
	if !c.isHostMode {
		configMapsToSkip, secretsToSkip, err := c.getOngoingSnapshotRequestsResourceNames(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to get ongoing snapshot requests resource names: %w", err)
		}
		for _, configMapNamespacedName := range configMapsToSkip {
			snapshotClient.addResourceToSkip(string(corev1.ResourceConfigMaps), configMapNamespacedName.String())
		}
		for _, secretNamespacedName := range secretsToSkip {
			snapshotClient.addResourceToSkip(string(corev1.ResourceSecrets), secretNamespacedName.String())
		}
	}
	err = snapshotClient.Run(ctx, c.vConfig)
	if err != nil {
		return false, fmt.Errorf("failed to run snapshot client: %w", err)
	}
	c.logger.Infof("Created vCluster snapshot in storage type %q", snapshotOptions.Type)

	// All done, now update the snapshot request phase to "Completed"! ✅
	snapshotRequest.Status.Phase = snapshotapi.RequestPhaseCompleted

	if snapshotRequest.Status.Phase == snapshotapi.RequestPhaseCompleted {
		c.eventRecorder.Eventf(
			configMap,
			nil,
			corev1.EventTypeNormal,
			"Completed",
			"ReconcileSnapShotRequest",
			"Snapshot request %s/%s has been completed",
			configMap.Namespace,
			configMap.Name,
		)
	} else {
		c.eventRecorder.Eventf(
			configMap,
			nil,
			corev1.EventTypeNormal,
			"PartiallyFailed",
			"ReconcileSnapShotRequest",
			"Snapshot request %s/%s has partially failed",
			configMap.Namespace,
			configMap.Name,
		)
	}
	return false, nil
}

func (c *Reconciler) updateRequest(ctx context.Context, previousConfigMapState client.Patch, configMap *corev1.ConfigMap, snapshotRequest snapshotapi.Request) error {
	snapshotRequestJSON, err := json.Marshal(snapshotRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot request to JSON: %w", err)
	}
	configMap.Data[snapshotapi.RequestKey] = string(snapshotRequestJSON)

	// patch snapshot request ConfigMap
	err = c.client().Patch(ctx, configMap, previousConfigMapState)
	if err != nil {
		return fmt.Errorf("failed to patch snapshot request ConfigMap %s/%s: %w", configMap.Namespace, configMap.Name, err)
	}
	c.logger.Debugf("Patched snapshot request %s/%s", configMap.Namespace, configMap.Name)
	return nil
}

func (c *Reconciler) getOngoingSnapshotRequestsResourceNames(ctx context.Context) ([]types.NamespacedName, []types.NamespacedName, error) {
	// list options with label selector
	var configMaps corev1.ConfigMapList
	listOptions := &client.ListOptions{
		Namespace: c.getRequestNamespace(),
		LabelSelector: labels.SelectorFromSet(map[string]string{
			constants.SnapshotRequestLabel: "",
		}),
	}
	err := c.client().List(ctx, &configMaps, listOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list snapshot requests: %w", err)
	}

	var ongoingRequestConfigMaps []types.NamespacedName
	for _, configMap := range configMaps.Items {
		snapshotRequest, err := snapshotapi.UnmarshalRequest(&configMap)
		if err != nil {
			c.logger.Errorf("Failed to unmarshal vcluster snapshot request from ConfigMap %s/%s: %v", configMap.Namespace, configMap.Name, err)
		}
		if !snapshotRequest.Done() {
			namespacedName := types.NamespacedName{
				Namespace: configMap.Namespace,
				Name:      configMap.Name,
			}
			ongoingRequestConfigMaps = append(ongoingRequestConfigMaps, namespacedName)
		}
	}

	var ongoingRequestSecrets []types.NamespacedName
	var secrets corev1.SecretList
	err = c.client().List(ctx, &secrets, listOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list snapshot request Secrets: %w", err)
	}
	for _, secret := range secrets.Items {
		namespacedName := types.NamespacedName{
			Namespace: secret.Namespace,
			Name:      secret.Name,
		}
		ongoingRequestSecrets = append(ongoingRequestSecrets, namespacedName)
	}

	return ongoingRequestConfigMaps, ongoingRequestSecrets, nil
}

func (c *Reconciler) cancelPreviousRequests(ctx context.Context, request *snapshotapi.Request) (bool, error) {
	if request.Status.Phase != snapshotapi.RequestPhaseNotStarted {
		// the current request has already started, previous requests should be already canceled
		return true, nil
	}

	var configMaps corev1.ConfigMapList
	listOptions := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			constants.SnapshotRequestLabel: "",
		}),
		Namespace: c.getRequestNamespace(),
	}
	err := c.client().List(ctx, &configMaps, listOptions)
	if err != nil {
		return false, fmt.Errorf("failed to list snapshot request ConfigMaps: %w", err)
	}
	currentRequestCanContinue := true

	for _, configMap := range configMaps.Items {
		otherRequest, err := snapshotapi.UnmarshalRequest(&configMap)
		if err != nil {
			c.logger.Errorf("Failed to unmarshal previous snapshot request from ConfigMap %s/%s: %v", configMap.Namespace, configMap.Name, err)
			continue
		}
		if !request.ShouldCancel(otherRequest) {
			if otherRequest.Status.Phase == snapshotapi.RequestPhaseCanceling {
				// the other request is still being canceled, so this one can't continue
				currentRequestCanContinue = false
			}
			continue
		}

		// cancel the previous request
		otherRequest.Status.Phase = snapshotapi.RequestPhaseCanceling
		oldValue := client.MergeFrom(configMap.DeepCopy())
		err = c.updateRequest(ctx, oldValue, &configMap, *otherRequest)
		if err != nil {
			return false, fmt.Errorf("failed to update snapshot request %s/%s: %w", configMap.Namespace, configMap.Name, err)
		}
		// the other request has been just canceled, so this one can't continue yet
		currentRequestCanContinue = false
	}

	return currentRequestCanContinue, nil
}
