package csi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/snapshot/volume"
)

const (
	VolumeSnapshotsAnnotation           = "vcluster.loft.sh/volumesnapshots"
	dynamicVolumeSnapshotLabel          = "vcluster.loft.sh/dynamicvolumesnapshot"
	PreProvisionedVolumeSnapshotLabel   = "vcluster.loft.sh/preprovisionedvolumesnapshot"
	persistentVolumeClaimNameAnnotation = "vcluster.loft.sh/persistentvolumeclaim"
)

// VolumeSnapshotter is a volume.Snapshotter interface implementation that creates CSI volume snapshots.
type VolumeSnapshotter struct {
	snapshotHandler

	vConfig              *config.VirtualClusterConfig
	etcdSnapshotLocation string

	// volumeSnapshotClasses maps CSI driver names to names of VolumeSnapshotClass resources that are used for creating
	// volume snapshots.
	volumeSnapshotClasses map[string]string
}

// NewVolumeSnapshotter creates a new instance of the CSI volume snapshotter.
func NewVolumeSnapshotter(ctx context.Context, vConfig *config.VirtualClusterConfig, kubeClient *kubernetes.Clientset, snapshotsClient *snapshotsv1.Clientset, etcdSnapshotLocation string, logger log.Logger) (*VolumeSnapshotter, error) {
	if vConfig == nil {
		return nil, errors.New("virtual cluster config is required")
	}
	if kubeClient == nil {
		return nil, errors.New("kubernetes client is required")
	}
	if snapshotsClient == nil {
		return nil, errors.New("snapshot client is required")
	}
	if etcdSnapshotLocation == "" {
		return nil, errors.New("etcd snapshot location is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}
	volumeSnapshotClasses, err := mapCSIDriversToVolumeSnapshotClasses(ctx, snapshotsClient, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to map CSI drivers to VolumeSnapshotClasses: %w", err)
	}

	snapshotter := &VolumeSnapshotter{
		snapshotHandler: snapshotHandler{
			kubeClient:      kubeClient,
			snapshotsClient: snapshotsClient,
			logger:          logger,
		},
		vConfig:               vConfig,
		etcdSnapshotLocation:  etcdSnapshotLocation,
		volumeSnapshotClasses: volumeSnapshotClasses,
	}
	return snapshotter, nil
}

// CheckIfPersistentVolumeIsSupported checks if the snapshotter can create a CSI volume snapshot of
// the specified persistent volume.
func (s *VolumeSnapshotter) CheckIfPersistentVolumeIsSupported(pv *corev1.PersistentVolume) error {
	hasPersistentVolumeClaim := pv.Spec.ClaimRef != nil &&
		pv.Spec.ClaimRef.Name != "" &&
		pv.Spec.ClaimRef.Namespace != "" &&
		pv.Spec.ClaimRef.Kind == "PersistentVolumeClaim"
	if !hasPersistentVolumeClaim {
		return fmt.Errorf("specified PersistentVolume does not have a PersistentVolumeClaim set: %w", volume.ErrPersistentVolumeNotSupported)
	}

	managedByCSIDriver := pv.Spec.CSI != nil && pv.Spec.CSI.Driver != ""
	if !managedByCSIDriver {
		return fmt.Errorf("specified PersistentVolume is not managed by the CSI driver: %w", volume.ErrPersistentVolumeNotSupported)
	}

	// In the current implementation, VolumeSnapshotClass with deletion policy 'Retain' must be
	// created before creating persistent volume snapshots.
	// Automatic creation of required VolumeSnapshotClasses will be implemented later.
	_, ok := s.volumeSnapshotClasses[pv.Spec.CSI.Driver]
	if !ok {
		return fmt.Errorf(
			"cannnot create snapshot for the specified PersistentVolume %s because VolumeSnapshotClass with deletion policy 'Retain' has not been found for the CSI driver %s: %w",
			pv.Name,
			pv.Spec.CSI.Driver,
			volume.ErrPersistentVolumeNotSupported)
	}

	return nil
}

// CreateSnapshots creates CSI volume snapshots of the specified persistent volumes.
//
// All the snapshots will be created in parallel, where every snapshot is created in a separate
// goroutine. This means that the total time to create all the snapshots should converge to the
// time needed to create the slowest (and probably the largest) snapshot.
func (s *VolumeSnapshotter) CreateSnapshots(ctx context.Context, persistentVolumes []corev1.PersistentVolume) (volume.CreateSnapshotsResult, error) {
	s.logger.Info("Start creating CSI VolumeSnapshots...")
	defer s.logger.Info("Finished creating CSI VolumeSnapshots.")

	var wg sync.WaitGroup
	maxSnapshots := len(persistentVolumes)
	errCh := make(chan error, maxSnapshots)
	persistentVolumeRefCh := make(chan volume.PersistentVolumeReference, maxSnapshots)

	// Since snapshot creation can be a very long-running operation (depending on the size of the
	// volume), every persistent volume snapshot is created in a separate go routine. This way
	// multiple persistent volume snapshots can be created simultaneously.
	for _, persistentVolume := range persistentVolumes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.createVolumeSnapshot(ctx, &persistentVolume)
			if err != nil {
				errCh <- err
			} else {
				persistentVolumeRef := volume.PersistentVolumeReference{
					PersistentVolumeClaim: types.NamespacedName{
						Name:      persistentVolume.Spec.ClaimRef.Name,
						Namespace: persistentVolume.Spec.ClaimRef.Namespace,
					},
					PersistentVolumeName: persistentVolume.Name,
				}
				persistentVolumeRefCh <- persistentVolumeRef
			}
		}()
	}

	// wait for all snapshots to be taken and close the errors channel
	go func() {
		wg.Wait()
		close(errCh)
		close(persistentVolumeRefCh)
	}()

	// aggregate all the errors
	var allErrors []error
	for err := range errCh {
		allErrors = append(allErrors, err)
	}
	result := volume.CreateSnapshotsResult{}
	for persistentVolumeRef := range persistentVolumeRefCh {
		result.SnapshottedPersistentVolumes = append(result.SnapshottedPersistentVolumes, persistentVolumeRef)
	}

	return result, errors.Join(allErrors...)
}

func (s *VolumeSnapshotter) Cleanup(ctx context.Context) error {
	s.logger.Debugf("Delete pre-provisioned VolumeSnapshots and VolumeSnapshotContents resources")

	// get all volume snapshots by label
	listOptions := metav1.ListOptions{
		LabelSelector: PreProvisionedVolumeSnapshotLabel,
	}

	// 1. Delete all VolumeSnapshot resources that have been created while creating vcluster snapshot.
	//
	// Currently, all VolumeSnapshot resources are using a VolumeSnapshotClass with delete policy set to
	// 'Retain', so it's safe to delete the VolumeSnapshots that have been already  added to a vcluster
	// snapshot.
	volumeSnapshots, err := s.snapshotsClient.SnapshotV1().VolumeSnapshots("").List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("failed to list VolumeSnapshots: %w", err)
	}
	for _, volumeSnapshot := range volumeSnapshots.Items {
		s.logger.Debugf("Delete pre-provisioned VolumeSnapshot %s/%s", volumeSnapshot.Namespace, volumeSnapshot.Name)
		err = s.snapshotsClient.SnapshotV1().VolumeSnapshots(volumeSnapshot.Namespace).Delete(ctx, volumeSnapshot.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete pre-provisioned VolumeSnapshot %s/%s: %w", volumeSnapshot.Name, volumeSnapshot.Name, err)
		}
		s.logger.Debugf("Deleted pre-provisioned VolumeSnapshot %s/%s", volumeSnapshot.Namespace, volumeSnapshot.Name)
	}

	// 2. Delete all VolumeSnapshotContent resources that have been created while creating vcluster snapshot.
	volumeSnapshotContents, err := s.snapshotsClient.SnapshotV1().VolumeSnapshotContents().List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("failed to list VolumeSnapshotContents: %w", err)
	}
	for _, volumeSnapshotContent := range volumeSnapshotContents.Items {
		s.logger.Debugf("Delete pre-provisioned VolumeSnapshotContent %s", volumeSnapshotContent.Name)
		err = s.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Delete(ctx, volumeSnapshotContent.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete VolumeSnapshotContent %s: %w", volumeSnapshotContent.Name, err)
		}
		s.logger.Debugf("Deleted pre-provisioned VolumeSnapshotContent %s", volumeSnapshotContent.Name)
	}

	s.logger.Debugf("Deleted pre-provisioned VolumeSnapshots and VolumeSnapshotContents resources")
	return nil
}

func (s *VolumeSnapshotter) createVolumeSnapshot(ctx context.Context, pv *corev1.PersistentVolume) error {
	s.logger.Infof("Create volume snapshot for PersistentVolume %s", pv.Name)

	volumeSnapshotClass, ok := s.volumeSnapshotClasses[pv.Spec.CSI.Driver]
	if !ok {
		return fmt.Errorf("cannot create snapshot for the PersistentVolume %s because required VolumeSnapshotClass has not been found for the CSI driver %s", pv.Name, pv.Spec.CSI.Driver)
	}

	// VolumeSnapshot is created from a PVC, so we need PVC namespace and name
	pvc := types.NamespacedName{
		Namespace: pv.Spec.ClaimRef.Namespace,
		Name:      pv.Spec.ClaimRef.Name,
	}

	// Step 1 - create a dynamic VolumeSnapshot from the existing PersistentVolumeClaim
	volumeSnapshot, volumeSnapshotContent, err := s.createDynamicVolumeSnapshot(ctx, pvc, volumeSnapshotClass)
	if err != nil {
		return fmt.Errorf("failed to create a dynamic VolumeSnapshot for the PersistentVolumeClaim %s: %w", pvc, err)
	}

	// Step 2 - create a pre-provisioned VolumeSnapshot from the previously created dynamic VolumeSnapshot
	//
	// The pre-provisioned VolumeSnapshot will not depend on the PersistentVolumeClaim from which the
	// dynamic snapshot has been created, so it can be restored when the original PersistentVolumeClaim
	// does not exist, e.g. in another virtual cluster.
	err = s.transformDynamicVolumeSnapshotToPreprovisioned(ctx, volumeSnapshot)
	if err != nil {
		return fmt.Errorf("failed to create a preprovisioned VolumeSnapshot for the PersistentVolume %s: %w", pvc, err)
	}

	// Step 3 - delete the dynamic VolumeSnapshot and VolumeSnapshotContent because we only need the
	// pre-provisioned VolumeSnapshot and VolumeSnapshotContent.
	err = s.deleteDynamicVolumeSnapshot(ctx, volumeSnapshot, volumeSnapshotContent)
	if err != nil {
		return fmt.Errorf(
			"failed to delete a dynamic VolumeSnapshot %s/%s and VolumeSnapshotContent %s that were created for the PersistentVolumeClaim %s: %w",
			volumeSnapshot.Namespace,
			volumeSnapshot.Name,
			volumeSnapshotContent.Name,
			pvc,
			err)
	}

	s.logger.Infof("Created volume snapshot for PersistentVolume %s", pv.Name)
	return nil
}

func (s *VolumeSnapshotter) createDynamicVolumeSnapshot(ctx context.Context, pvc types.NamespacedName, volumeSnapshotClassName string) (*snapshotsv1api.VolumeSnapshot, *snapshotsv1api.VolumeSnapshotContent, error) {
	s.logger.Debugf("Create dynamic VolumeSnapshot for PersistentVolumeClaim %s", pvc)

	volumeSnapshot := &snapshotsv1api.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", pvc.Name),
			Namespace:    pvc.Namespace,
			Labels: map[string]string{
				dynamicVolumeSnapshotLabel: "",
			},
		},
		Spec: snapshotsv1api.VolumeSnapshotSpec{
			Source: snapshotsv1api.VolumeSnapshotSource{
				PersistentVolumeClaimName: &pvc.Name,
			},
			VolumeSnapshotClassName: ptr.To(volumeSnapshotClassName),
		},
	}
	var err error
	volumeSnapshot, err = s.snapshotsClient.SnapshotV1().VolumeSnapshots(pvc.Namespace).Create(ctx, volumeSnapshot, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("could not create VolumeSnapshot resource for the PersistentVolumeClaim %s: %w", pvc, err)
	}
	s.logger.Debugf("Created VolumeSnapshot resource %s/%s for the PersistentVolumeClaim %s", volumeSnapshot.Namespace, volumeSnapshot.Name, pvc)

	// wait for the dynamic VolumeSnapshot to be ready
	s.logger.Debugf("Waiting for VolumeSnapshot resource %s/%s to be ready for use...", volumeSnapshot.Namespace, volumeSnapshot.Name)
	dynamicVolumeSnapshot, dynamicVolumeSnapshotContent, err := s.waitForReadyToUse(ctx, volumeSnapshot.Namespace, volumeSnapshot.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("error waiting for the creation of the dynamic VolumeSnapshot resource for the PersistentVolumeClaim %s: %w", pvc, err)
	}
	s.logger.Debugf("Dynamic VolumeSnapshot %s/%s is ready for use!", dynamicVolumeSnapshot.Namespace, dynamicVolumeSnapshot.Name)
	s.logger.Debugf("Dynamic VolumeSnapshotContent %s with snapshot handle '%s' is ready for use!", dynamicVolumeSnapshotContent.Name, *dynamicVolumeSnapshotContent.Status.SnapshotHandle)

	return dynamicVolumeSnapshot, dynamicVolumeSnapshotContent, nil
}

func (s *VolumeSnapshotter) transformDynamicVolumeSnapshotToPreprovisioned(ctx context.Context, dynamicVolumeSnapshot *snapshotsv1api.VolumeSnapshot) error {
	dynamicVolumeSnapshotNamespacedName := types.NamespacedName{
		Namespace: dynamicVolumeSnapshot.Namespace,
		Name:      dynamicVolumeSnapshot.Name,
	}
	s.logger.Debugf("Transform dynamic VolumeSnapshot %s into a pre-provisioned snapshot", dynamicVolumeSnapshotNamespacedName)

	// Ensure that the dynamic VolumeSnapshot is ready to use.
	// These checks are a safety net, but should never fail, because createDynamicVolumeSnapshot
	// function above should have made sure that the dynamic VolumeSnapshot is ready for use.
	persistentVolumeClaimName := dynamicVolumeSnapshot.Spec.Source.PersistentVolumeClaimName
	if persistentVolumeClaimName == nil || *persistentVolumeClaimName == "" {
		return fmt.Errorf("dynamic VolumeSnapshot '%s' does not have a PersistentVolumeClaim as a source", dynamicVolumeSnapshotNamespacedName)
	}
	if dynamicVolumeSnapshot.Status == nil {
		return fmt.Errorf("dynamic VolumeSnapshot '%s' does not have Status yet", dynamicVolumeSnapshotNamespacedName)
	}
	if dynamicVolumeSnapshot.Status.ReadyToUse == nil || !*dynamicVolumeSnapshot.Status.ReadyToUse {
		return fmt.Errorf("dynamic VolumeSnapshot %s is not ready to be used", dynamicVolumeSnapshotNamespacedName)
	}

	boundVolumeSnapshotContentName := dynamicVolumeSnapshot.Status.BoundVolumeSnapshotContentName
	if boundVolumeSnapshotContentName == nil || *boundVolumeSnapshotContentName == "" {
		return fmt.Errorf("dynamic VolumeSnapshot %s does not have bound VolumeSnapshotContent name set", dynamicVolumeSnapshotNamespacedName)
	}

	// get dynamic VolumeSnapshotContent - here we will find the snapshot handle that we need for the pre-provisioned volume snapshot
	dynamicVolumeSnapshotContent, err := s.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Get(ctx, *boundVolumeSnapshotContentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("could not get bound VolumeSnapshotContent '%s' for dynamic VolumeSnapshot '%s': %w", *boundVolumeSnapshotContentName, dynamicVolumeSnapshotNamespacedName, err)
	}

	if dynamicVolumeSnapshotContent.Status.SnapshotHandle == nil {
		return fmt.Errorf("dynamic VolumeSnapshotContent '%s' does not have status.snapshotHandle set", dynamicVolumeSnapshotContent.Name)
	}

	// get the source PersistentVolumeClaim
	persistentVolumeClaim, err := s.kubeClient.CoreV1().PersistentVolumeClaims(dynamicVolumeSnapshot.Namespace).Get(ctx, *persistentVolumeClaimName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get PersistentVolumeClaim %s/%s: %w", dynamicVolumeSnapshot.Namespace, *persistentVolumeClaimName, err)
	}

	persistentVolumeClaimJSON, err := json.Marshal(persistentVolumeClaim)
	if err != nil {
		return fmt.Errorf("failed to marshal PersistentVolumeClaim %s/%s: %w", persistentVolumeClaim.Namespace, persistentVolumeClaim.Name, err)
	}

	snapshotNameBase := dynamicVolumeSnapshot.Name
	preProvisionedVolumeSnapshotContentName := fmt.Sprintf("%s-snap-content", snapshotNameBase)
	preProvisionedVolumeSnapshotName := fmt.Sprintf("%s-snap", snapshotNameBase)

	preProvisionedVolumeSnapshotContent := &snapshotsv1api.VolumeSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      preProvisionedVolumeSnapshotContentName,
			Namespace: dynamicVolumeSnapshot.Namespace,
			Labels: map[string]string{
				PreProvisionedVolumeSnapshotLabel: "",
			},
			Annotations: map[string]string{
				persistentVolumeClaimNameAnnotation: string(persistentVolumeClaimJSON),
			},
		},
		Spec: snapshotsv1api.VolumeSnapshotContentSpec{
			DeletionPolicy: snapshotsv1api.VolumeSnapshotContentRetain,
			Driver:         dynamicVolumeSnapshotContent.Spec.Driver,
			Source: snapshotsv1api.VolumeSnapshotContentSource{
				SnapshotHandle: dynamicVolumeSnapshotContent.Status.SnapshotHandle,
			},
			VolumeSnapshotClassName: dynamicVolumeSnapshotContent.Spec.VolumeSnapshotClassName,
			VolumeSnapshotRef: corev1.ObjectReference{
				Name:      preProvisionedVolumeSnapshotName,
				Namespace: dynamicVolumeSnapshot.Namespace,
			},
			SourceVolumeMode: dynamicVolumeSnapshotContent.Spec.SourceVolumeMode,
		},
	}
	s.logger.Debugf("Create pre-provisioned VolumeSnapshotContent %s", preProvisionedVolumeSnapshotContentName)
	_, err = s.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Create(ctx, preProvisionedVolumeSnapshotContent, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create the pre-provisioned VolumeSnapshotContent '%s': %w", preProvisionedVolumeSnapshotContent.Name, err)
	}
	s.logger.Debugf("Created pre-provisioned VolumeSnapshotContent %s", preProvisionedVolumeSnapshotContentName)

	s.logger.Debugf("Create pre-provisioned VolumeSnapshot %s/%s", dynamicVolumeSnapshot.Namespace, preProvisionedVolumeSnapshotName)
	preProvisionedVolumeSnapshot := &snapshotsv1api.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      preProvisionedVolumeSnapshotName,
			Namespace: dynamicVolumeSnapshot.Namespace,
			Labels: map[string]string{
				PreProvisionedVolumeSnapshotLabel: "",
			},
			Annotations: map[string]string{
				persistentVolumeClaimNameAnnotation: string(persistentVolumeClaimJSON),
			},
		},
		Spec: snapshotsv1api.VolumeSnapshotSpec{
			Source: snapshotsv1api.VolumeSnapshotSource{
				VolumeSnapshotContentName: &preProvisionedVolumeSnapshotContentName,
			},
			VolumeSnapshotClassName: dynamicVolumeSnapshot.Spec.VolumeSnapshotClassName,
		},
	}
	preProvisionedVolumeSnapshot, err = s.snapshotsClient.SnapshotV1().VolumeSnapshots(dynamicVolumeSnapshot.Namespace).Create(ctx, preProvisionedVolumeSnapshot, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create the pre-provisioned VolumeSnapshot '%s/%s': %w", dynamicVolumeSnapshot.Namespace, preProvisionedVolumeSnapshotName, err)
	}
	s.logger.Debugf(
		"Created pre-provisioned VolumeSnapshot %s/%s",
		dynamicVolumeSnapshot.Namespace,
		preProvisionedVolumeSnapshotName)
	s.logger.Debugf(
		"Transformed dynamic VolumeSnapshot %s into a pre-provisioned snapshot %s/%s",
		dynamicVolumeSnapshotNamespacedName,
		dynamicVolumeSnapshot.Namespace,
		preProvisionedVolumeSnapshotName)

	s.logger.Debugf("Waiting for pre-provisioned VolumeSnapshot resource %s/%s to be ready for use...", preProvisionedVolumeSnapshot.Namespace, preProvisionedVolumeSnapshot.Name)
	preProvisionedVolumeSnapshot, preProvisionedVolumeSnapshotContent, err = s.waitForReadyToUse(ctx, preProvisionedVolumeSnapshot.Namespace, preProvisionedVolumeSnapshot.Name)
	if err != nil {
		return fmt.Errorf(
			"error waiting for the creation of the pre-provisioned VolumeSnapshot %s/%s and VolumeSnapshotContent %s: %w",
			dynamicVolumeSnapshot.Namespace, preProvisionedVolumeSnapshotName, preProvisionedVolumeSnapshotContentName, err)
	}
	s.logger.Debugf(
		"Pre-provisioned VolumeSnapshot %s/%s is ready for use!",
		preProvisionedVolumeSnapshot.Namespace, preProvisionedVolumeSnapshot.Name)
	s.logger.Debugf(
		"Pre-provisioned VolumeSnapshotContent %s with snapshot handle '%s' is ready for use!",
		preProvisionedVolumeSnapshotContent.Name, *preProvisionedVolumeSnapshotContent.Status.SnapshotHandle)

	// update PVC's VolumeSnapshots annotation
	var volumeSnapshotsMap map[string]string
	volumeSnapshotsMapJson, ok := persistentVolumeClaim.Annotations[VolumeSnapshotsAnnotation]
	if ok {
		err = json.Unmarshal([]byte(volumeSnapshotsMapJson), &volumeSnapshotsMap)
		if err != nil {
			return fmt.Errorf("failed to unmarshal VolumeSnapshots map from annotation {%s: %s}: %w", VolumeSnapshotsAnnotation, volumeSnapshotsMapJson, err)
		}
	}
	if volumeSnapshotsMap == nil {
		volumeSnapshotsMap = make(map[string]string)
	}
	volumeSnapshotsMap[s.etcdSnapshotLocation] = preProvisionedVolumeSnapshot.Name
	volumeSnapshotsMapJsonBytes, err := json.Marshal(volumeSnapshotsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updated VolumeSnapshots map annotation %s: %w", VolumeSnapshotsAnnotation, err)
	}
	annotationPatch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				VolumeSnapshotsAnnotation: string(volumeSnapshotsMapJsonBytes),
			},
		},
	}
	annotationPatchBytes, err := json.Marshal(annotationPatch)
	if err != nil {
		return fmt.Errorf("failed to marshal PVC annotation '%s' patch: %w", VolumeSnapshotsAnnotation, err)
	}

	_, err = s.kubeClient.CoreV1().PersistentVolumeClaims(dynamicVolumeSnapshot.Namespace).Patch(ctx, persistentVolumeClaim.Name, types.StrategicMergePatchType, annotationPatchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch PersistentVolumeClaim annotation '%s': %w", VolumeSnapshotsAnnotation, err)
	}
	_, err = s.kubeClient.CoreV1().PersistentVolumes().Patch(ctx, persistentVolumeClaim.Spec.VolumeName, types.StrategicMergePatchType, annotationPatchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch PersistentVolume annotation '%s': %w", VolumeSnapshotsAnnotation, err)
	}

	return nil
}

func (s *VolumeSnapshotter) deleteDynamicVolumeSnapshot(ctx context.Context, dynamicVolumeSnapshot *snapshotsv1api.VolumeSnapshot, dynamicVolumeSnapshotContent *snapshotsv1api.VolumeSnapshotContent) error {
	s.logger.Debugf("Delete dynamic VolumeSnapshot %s/%s", dynamicVolumeSnapshot.Namespace, dynamicVolumeSnapshot.Name)
	err := s.snapshotsClient.SnapshotV1().VolumeSnapshots(dynamicVolumeSnapshot.Namespace).Delete(ctx, dynamicVolumeSnapshot.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete the dynamic VolumeSnapshot '%s': %w", dynamicVolumeSnapshot.Name, err)
	}
	s.logger.Debugf("Deleted dynamic VolumeSnapshot %s/%s", dynamicVolumeSnapshot.Namespace, dynamicVolumeSnapshot.Name)

	s.logger.Debugf("Delete dynamic VolumeSnapshotContent %s", dynamicVolumeSnapshotContent.Name)
	err = s.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Delete(ctx, dynamicVolumeSnapshotContent.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete the dynamic VolumeSnapshotContents '%s': %w", dynamicVolumeSnapshotContent.Name, err)
	}
	s.logger.Debugf("Deleted dynamic VolumeSnapshotContent %s", dynamicVolumeSnapshotContent.Name)

	err = s.waitForVolumeSnapshotDeleted(ctx, dynamicVolumeSnapshot.Namespace, dynamicVolumeSnapshot.Name, dynamicVolumeSnapshotContent.Name)
	if err != nil {
		return fmt.Errorf(
			"failed to delete the dynamic VolumeSnapshot %s/%s and VolumeSnapshotContent '%s': %w",
			dynamicVolumeSnapshot.Namespace,
			dynamicVolumeSnapshot.Name,
			dynamicVolumeSnapshotContent.Name,
			err)
	}

	return nil
}
