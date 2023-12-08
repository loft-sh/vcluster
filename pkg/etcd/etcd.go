package etcd

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"

	"github.com/loft-sh/log/scanner"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

var (
	dataDir = "/data"

	k3sDataDir  = path.Join(dataDir, "server")
	k3sKineSock = path.Join(k3sDataDir, "kine.sock")

	etcdBackup = path.Join(dataDir, "snapshot.")

	kubeRootCaCrt = regexp.MustCompile(`/registry/configmaps/[^/]+/kube-root-ca.crt`)
)

func Snapshot(ctx context.Context, scheme *runtime.Scheme) (string, error) {
	distro := constants.GetVClusterDistro()
	if distro != constants.K3SDistro && distro != constants.K0SDistro {
		// TODO: support k8s & eks
		return "", fmt.Errorf("unsupported vCluster distro")
	}

	filePath := etcdBackup + random.String(6)
	f, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	etcdClient, err := GetEtcdClient(ctx, "", "unix://"+k3sKineSock)
	if err != nil {
		return "", err
	}

	result, err := etcdClient.Get(ctx, "/registry/", clientv3.WithPrefix())
	if err != nil {
		return "", err
	}

	tarWriter := tar.NewWriter(f)
	defer tarWriter.Close()

	// codecFactory := serializer.NewCodecFactory(scheme)
	for _, r := range result.Kvs {
		if len(r.Key) == 0 || len(r.Value) == 0 {
			continue
		}

		// filter out kube-root-ca.crt configmaps to avoid conflicts when we
		// exchange the certificates later.
		if kubeRootCaCrt.Match(r.Key) {
			continue
		}

		// write key value
		err = writeKeyValue(tarWriter, r.Key, r.Value)
		if err != nil {
			return "", fmt.Errorf("writing key %s: %w", string(r.Key), err)
		}
	}

	return filePath, nil
}

func Restore(ctx context.Context, data io.ReadCloser) error {
	if data == nil {
		return nil
	}
	defer data.Close()

	distro := constants.GetVClusterDistro()
	if distro != constants.K3SDistro && distro != constants.K0SDistro {
		// TODO: support k8s & eks
		return fmt.Errorf("unsupported vCluster distro")
	}

	// run kine in background
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer cancel()
		defer close(done)

		err := runKine(ctx)
		if err != nil {
			klog.ErrorS(err, "error running kine")
		}
	}()

	// get etcd client
	klog.Info("Wait for kine to come up")
	etcdClient, err := WaitForEtcdClient(ctx, "", "unix://"+k3sKineSock)
	if err != nil {
		return err
	}

	// insert
	klog.Info("Restoring etcd state")
	tarReader := tar.NewReader(data)
	for {
		key, value, err := readKeyValue(tarReader)
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("read etcd key/value: %w", err)
		} else if errors.Is(err, io.EOF) || len(key) == 0 {
			break
		}

		// this is only needed for kine / for etcd we can do regular put
		_, err = etcdClient.Txn(ctx).
			If(clientv3.Compare(clientv3.ModRevision(string(key)), "=", 0)).
			Then(clientv3.OpPut(string(key), string(value))).
			Else(clientv3.OpGet(string(key))).
			Commit()
		if err != nil {
			return err
		}
	}

	// wait for kine to exit
	cancel()
	<-done
	return nil
}

func runKine(ctx context.Context) error {
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	defer writer.Close()

	// make data dir
	err = os.MkdirAll(k3sDataDir, 0777)
	if err != nil {
		return err
	}

	// start func
	done := make(chan struct{})
	go func() {
		defer close(done)

		// make sure we scan the output correctly
		scan := scanner.NewScanner(reader)
		for scan.Scan() {
			line := scan.Text()
			if len(line) == 0 {
				continue
			}

			// print to our logs
			args := []interface{}{"component", "kine"}
			loghelper.PrintKlogLine(line, args)
		}
	}()

	cmd := exec.CommandContext(ctx, "/usr/local/bin/kine", "--listen-address", "unix://kine.sock")
	cmd.Dir = k3sDataDir
	cmd.Stdout = writer
	cmd.Stderr = writer
	err = cmd.Run()
	_ = writer.Close()
	<-done
	if err != nil && err.Error() != "signal: killed" {
		return err
	}

	return nil
}
