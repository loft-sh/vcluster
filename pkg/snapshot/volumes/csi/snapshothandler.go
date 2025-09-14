package csi

import (
	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/client-go/kubernetes"
)

type snapshotHandler struct {
	kubeClient      *kubernetes.Clientset
	snapshotsClient *snapshotsv1.Clientset
	logger          loghelper.Logger
}
