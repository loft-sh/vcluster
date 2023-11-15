package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/apiserver/pkg/builders"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	"github.com/loft-sh/loftctl/v3/pkg/kube"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	_ "github.com/loft-sh/api/v3/pkg/apis/management/install" // Install the management group
)

func StreamTask(ctx context.Context, managementClient kube.Interface, task *managementv1.Task, out io.Writer, log log.Logger) (err error) {
	// cleanup on ctrl+c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		_ = managementClient.Loft().ManagementV1().Tasks().Delete(context.TODO(), task.Name, metav1.DeleteOptions{})
		os.Exit(1)
	}()

	defer func() {
		signal.Stop(c)
	}()

	log.Infof("Waiting for task to start...")
	createdTask, err := managementClient.Loft().ManagementV1().Tasks().Create(ctx, task, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "create task")
	}

	// wait for the task to be ready
	err = wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), true, func(ctx context.Context) (done bool, err error) {
		task, err := managementClient.Loft().ManagementV1().Tasks().Get(ctx, createdTask.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if task.Status.PodPhase == corev1.PodSucceeded || task.Status.PodPhase == corev1.PodFailed {
			return true, nil
		}
		if task.Status.PodPhase == corev1.PodRunning && task.Status.ContainerState != nil && task.Status.ContainerState.Ready {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		// compile an understandable error message
		task, err := managementClient.Loft().ManagementV1().Tasks().Get(ctx, createdTask.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("the task couldn't be retrieved from server: %w", err)
		}

		// task pending
		if task.Status.PodPhase == corev1.PodPending {
			if task.Status.ContainerState != nil && task.Status.ContainerState.State.Waiting != nil {
				return fmt.Errorf("the task is still pending which means that there was a problem starting a pod in the host cluster. The task container is pending because: %s (%s). For more information please take a look in the namespace loft-tasks in the Loft management cluster", task.Status.ContainerState.State.Waiting.Message, task.Status.ContainerState.State.Waiting.Reason)
			}

			return fmt.Errorf("the task is still pending which means that there was a problem starting a pod in the host cluster. Please check the pods in namespace loft-tasks in the Loft management cluster as well as any events there that might indicate an error why this task couldn't be started")
		}

		if task.Status.ContainerState != nil {
			out, err := json.MarshalIndent(task.Status.ContainerState, "", "  ")
			if err == nil {
				return fmt.Errorf("the task somehow still hasn't started which means that there was a problem starting a pod in the host cluster. Please check the pods in namespace loft-tasks in the Loft management cluster as well as any events there that might indicate an error why this task couldn't be started. The task container has the following status: \n%v", string(out))
			}
		}
		return fmt.Errorf("the task somehow still hasn't started which means that there was a problem starting a pod in the host cluster. Please check the pods in namespace loft-tasks in the Loft management cluster as well as any events there that might indicate an error why this task couldn't be started")
	}

	// now stream the logs
	for retry := 3; retry >= 0; retry-- {
		request := managementClient.Loft().ManagementV1().RESTClient().Get().Name(createdTask.Name).Resource("tasks").SubResource("log").VersionedParams(&managementv1.TaskLogOptions{
			Follow: true,
		}, runtime.NewParameterCodec(builders.ParameterScheme))

		reader, err := request.Stream(ctx)
		if err != nil {
			log.Warnf("Error streaming task logs: %v", err)
			time.Sleep(time.Second * 5)
			continue
		}

		_, err = io.Copy(out, reader)
		if err != nil {
			log.Warnf("Error reading task logs: %v", err)
			time.Sleep(time.Second * 5)
			continue
		}

		break
	}

	// check task result
	task, err = managementClient.Loft().ManagementV1().Tasks().Get(ctx, createdTask.Name, metav1.GetOptions{})
	if err != nil {
		return err
	} else if task.Status.PodPhase == corev1.PodFailed {
		if task.Status.ContainerState != nil && task.Status.ContainerState.State.Terminated != nil {
			return fmt.Errorf("task failed: %v (%v)", task.Status.ContainerState.State.Terminated.Message, task.Status.ContainerState.State.Terminated.Reason)
		}

		return fmt.Errorf("task failed")
	}

	return nil
}
