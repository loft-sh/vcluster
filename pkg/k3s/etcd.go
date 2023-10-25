package k3s

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/certs"
	"k8s.io/klog/v2"
)

const (
	etcdDataDir = "/data/etcd"
)

func StartEmbeddedEtcd(ctx context.Context, certificatesDir string) error {
	args := []string{
		"--data-dir=" + etcdDataDir,
		"--cert-file=" + filepath.Join(certificatesDir, certs.EtcdServerCertName),
		"--client-cert-auth=true",
		"--advertise-client-urls=https://127.0.0.1:2379",
		"--initial-advertise-peer-urls=https://127.0.0.1:2380",
		"--initial-cluster=vcluster=https://127.0.0.1:2380",
		"--initial-cluster-token=vcluster",
		"--initial-cluster-state=new",
		"--listen-client-urls=https://127.0.0.1:2379",
		"--listen-metrics-urls=http://127.0.0.1:2381",
		"--listen-peer-urls=https://127.0.0.1:2380",
		"--key-file=" + filepath.Join(certificatesDir, certs.EtcdServerKeyName),
		"--name=vcluster",
		"--peer-cert-file=" + filepath.Join(certificatesDir, certs.EtcdPeerCertName),
		"--peer-client-cert-auth=true",
		"--peer-key-file=" + filepath.Join(certificatesDir, certs.EtcdPeerKeyName),
		"--peer-trusted-ca-file=" + filepath.Join(certificatesDir, certs.EtcdCACertName),
		"--snapshot-count=10000",
		"--trusted-ca-file=" + filepath.Join(certificatesDir, certs.EtcdCACertName),
	}

	// start the command
	klog.InfoS("Starting embedded etcd", "args", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "etcd", args...)

	// start the command
	go func() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			klog.Fatalf("Error running embedded etcd: %v", err)
		}
	}()

	// wait until etcd is reachable
	for {
		_, err := getClient(ctx, certificatesDir, "https://127.0.0.1:2379")
		if err != nil {
			klog.Infof("Couldn't connect to embedded etcd (will retry in a second): %v", err)
			time.Sleep(time.Second)
		}

		return nil
	}
}
