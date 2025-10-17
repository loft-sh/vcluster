package snapshot

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/test/framework"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

// createPVCWithData creates a PVC, and it writes test data into the specified test file.
// Test file is saved under	the '/data/$testFileName' path.
func createPVCWithData(ctx context.Context, client *kubernetes.Clientset, pvcNamespace, pvcName, testFileName, testData string) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: pvcNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
			StorageClassName: ptr.To("csi-hostpath-sc"),
		},
	}
	_, err := client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	// deploy a Job that writes test data to the PVC
	deployJob(ctx, client, pvc.Namespace, "write-test-data", pvc.Name, fmt.Sprintf("echo '%s' > /data/%s", testData, testFileName), testFileName)
}

func deletePVC(ctx context.Context, client *kubernetes.Clientset, pvcNamespace, pvcName string) {
	err := client.CoreV1().PersistentVolumeClaims(pvcNamespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
	framework.ExpectNoError(err)

	Eventually(func() error {
		pvc, err := client.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(ctx, pvcName, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			// PVC deleted successfully
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to get PVC %s/%s: %w", pvcNamespace, pvcName, err)
		}
		return fmt.Errorf("PVC %s/%s is not deleted", pvc.Namespace, pvc.Name)
	}).WithPolling(framework.PollInterval).
		WithTimeout(framework.PollTimeout).
		Should(Succeed())
}

func checkPVCData(ctx context.Context, client *kubernetes.Clientset, pvcNamespace, pvcName, testFileName, testData string) {
	script := fmt.Sprintf(`actual=$(cat "/data/%s"); expected=%q;
if [ "$actual" = "$expected" ]; then
  echo "OK: content matches";
else
  echo "FAIL: expected [$expected], got [$actual]" >&2;
  exit 1;
fi`, testFileName, testData)
	deployJob(ctx, client, pvcNamespace, "check-test-data", pvcName, script, testFileName)
}

func deployJob(ctx context.Context, client *kubernetes.Clientset, jobNamespace, jobName, pvcName, command, testFile string) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", jobName),
			Namespace:    jobNamespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: ptr.To(int32(0)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:    "test-job",
						Image:   "busybox:1.36",
						Command: []string{"sh", "-c", command},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "data",
							MountPath: "/data",
						}},
						WorkingDir: "/data",
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{Command: []string{"sh", "-c", "test -f /data/" + testFile}},
							},
							InitialDelaySeconds: 1,
							PeriodSeconds:       1,
							FailureThreshold:    10,
							TimeoutSeconds:      2,
						},
					}},
					Volumes: []corev1.Volume{{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					}},
				},
			},
		},
	}
	job, err := client.BatchV1().Jobs(job.Namespace).Create(ctx, job, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	Eventually(func(ctx context.Context) error {
		job, err = client.BatchV1().Jobs(job.Namespace).Get(ctx, job.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get job %s/%s: %w", job.Namespace, job.Name, err)
		}
		if job.Status.Succeeded != 1 {
			return fmt.Errorf("job %s/%s did not succeed", job.Namespace, job.Name)
		}
		return nil
	}).WithContext(ctx).
		WithPolling(framework.PollInterval).
		WithTimeout(framework.PollTimeoutLong).
		Should(Succeed())

	// delete the job
	err = client.BatchV1().Jobs(job.Namespace).Delete(ctx, job.Name, metav1.DeleteOptions{
		PropagationPolicy: ptr.To(metav1.DeletePropagationBackground),
	})
	framework.ExpectNoError(err)
	Eventually(func(ctx context.Context) error {
		job, err = client.BatchV1().Jobs(job.Namespace).Get(ctx, job.Name, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			// job deleted successfully
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to get job %s/%s: %w", job.Namespace, job.Name, err)
		}
		return fmt.Errorf("job %s/%s did not delete", job.Namespace, job.Name)
	}).WithContext(ctx).
		WithPolling(framework.PollInterval).
		WithTimeout(framework.PollTimeoutLong).
		Should(Succeed())

	Eventually(func(ctx context.Context) error {
		jobPods, err := client.CoreV1().Pods(job.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
		})
		if len(jobPods.Items) == 0 {
			// job Pod deleted successfully
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to get pods for job %s/%s: %w", job.Namespace, job.Name, err)
		}
		return fmt.Errorf("pods for job %s/%s have not been deleted", job.Namespace, job.Name)
	}).WithContext(ctx).
		WithPolling(framework.PollInterval).
		WithTimeout(framework.PollTimeoutLong).
		Should(Succeed())
}
