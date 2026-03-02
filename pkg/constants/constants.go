package constants

import "path/filepath"

var (
	K3sKineEndpoint   = "unix:///data/server/kine.sock"
	K3sSqliteDatabase = "/data/server/db/state.db"

	DataDir = "/data"
	PKIDir  = filepath.Join(DataDir, "pki")

	K8sKineEndpoint   = "unix://" + filepath.Join(DataDir, "kine.sock")
	K8sSqliteDatabase = filepath.Join(DataDir, "state.db")

	EmbeddedEtcdData         = filepath.Join(DataDir, "etcd")
	EmbeddedCoreDNSAdminConf = filepath.Join(DataDir, "vcluster", "admin.conf")

	ServerCAKey         = filepath.Join(PKIDir, "server-ca.key")
	ServerCACert        = filepath.Join(PKIDir, "server-ca.crt")
	ClientCACert        = filepath.Join(PKIDir, "client-ca.crt")
	RequestHeaderCACert = filepath.Join(PKIDir, "front-proxy-ca.crt")

	FrontProxyClientCert = filepath.Join(PKIDir, "front-proxy-client.crt")
	FrontProxyClientKey  = filepath.Join(PKIDir, "front-proxy-client.key")

	SAKey  = filepath.Join(PKIDir, "sa.key")
	SACert = filepath.Join(PKIDir, "sa.pub")

	APIServerCert = filepath.Join(PKIDir, "apiserver.crt")
	APIServerKey  = filepath.Join(PKIDir, "apiserver.key")

	APIServerKubeletClientCert = filepath.Join(PKIDir, "apiserver-kubelet-client.crt")
	APIServerKubeletClientKey  = filepath.Join(PKIDir, "apiserver-kubelet-client.key")

	AdminKubeConfig       = filepath.Join(PKIDir, "admin.conf")
	ControllerManagerConf = filepath.Join(PKIDir, "controller-manager.conf")
	SchedulerConf         = filepath.Join(PKIDir, "scheduler.conf")

	BinariesDir                = "/binaries"
	K8sAPIServerBinary         = filepath.Join(BinariesDir, "kube-apiserver")
	K8sControllerManagerBinary = filepath.Join(BinariesDir, "kube-controller-manager")
	K8sSchedulerBinary         = filepath.Join(BinariesDir, "kube-scheduler")
	KineBinary                 = filepath.Join(BinariesDir, "kine")
	HelmBinary                 = filepath.Join(BinariesDir, "helm")

	// DefaultVClusterConfigLocation is the default location of the vCluster config within the container
	DefaultVClusterConfigLocation = "/var/lib/vcluster/config.yaml"

	// VClusterNamespaceInHostMappingSpecialCharacter is an empty string that mean vCluster host namespace
	// in the config.sync.fromHost.*.selector.mappings
	VClusterNamespaceInHostMappingSpecialCharacter = ""

	SystemPriorityClassesAllowList = []string{
		"system-node-critical",
		"system-cluster-critical",
	}
)
