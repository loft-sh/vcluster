package k3s

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/certs"
	clientv3 "go.etcd.io/etcd/client/v3"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"
	_ "modernc.org/sqlite"
)

const databaseFile = "/data/server/db/state.db"

type kineEntry struct {
	ID    int    `json:"id,omitempty" sql:"id"`
	Name  int    `json:"name,omitempty" sql:"name"`
	Value string `json:"value,omitempty" sql:"value"`
}

func NeedsSQLiteMigration() bool {
	_, err := os.Stat(databaseFile)
	return err == nil
}

func NeedsEtcdMigration(etcdEndpoint string) bool {
	// if etcd point is local we don't need to migrate
	if etcdEndpoint == "https://127.0.0.1:2379" {
		return false
	}

	return false
}

func MigrateFromEtcd(ctx context.Context, certificatesDir string, fromEndpoint, toEndpoint string) error {
	return nil
}

func MigrateFromSQLite(ctx context.Context, certificatesDir string, etcdEndpoint string) error {
	if !NeedsSQLiteMigration() {
		return nil
	}

	klog.Infof("Migrating content from sqlite to etcd")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// open sqlite database
	db, err := sql.Open("sqlite", databaseFile+"?_journal=WAL&cache=shared&_busy_timeout=30000")
	if err != nil {
		return err
	}
	defer db.Close()

	etcdClient, err := getClient(ctx, certificatesDir, etcdEndpoint)
	if err != nil {
		return err
	}
	defer etcdClient.Close()

	// migrate database
	rows, err := db.QueryContext(ctx, `
	SELECT *
	FROM (
		SELECT kv.id AS id, kv.name AS name, kv.value AS value
		FROM kine AS kv
		JOIN (
			SELECT MAX(mkv.id) AS id
			FROM kine AS mkv
			GROUP BY mkv.name
		) AS maxkv
		ON maxkv.id = kv.id
		WHERE kv.deleted = 0
	) AS lkv
	ORDER BY lkv.id ASC
`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// loop over values
	for rows.Next() {
		entry := &kineEntry{}
		err := rows.Scan(&entry.ID, &entry.Name, &entry.Value)
		if err != nil {
			return fmt.Errorf("scanning sqlite row: %w", err)
		}

		klog.Infof("Migrating etcd key %d", entry.Name)
		_, err = etcdClient.Put(ctx, strconv.Itoa(entry.Name), entry.Value)
		if err != nil {
			return fmt.Errorf("adding key %d to etcd: %w", entry.Name, err)
		}
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("error retrieving sqlite data: %w", err)
	}

	return os.Rename(databaseFile, databaseFile+".migrated")
}

// getClient returns an etcd client connected to the specified endpoints.
// If no endpoints are provided, endpoints are retrieved from the provided runtime config.
// If the runtime config does not list any endpoints, the default endpoint is used.
// The returned client should be closed when no longer needed, in order to avoid leaking GRPC
// client goroutines.
func getClient(ctx context.Context, certificatesDir string, endpoints ...string) (*clientv3.Client, error) {
	cfg, err := getClientConfig(ctx, certificatesDir, endpoints...)
	if err != nil {
		return nil, err
	}

	return clientv3.New(*cfg)
}

// getClientConfig generates an etcd client config connected to the specified endpoints.
// If no endpoints are provided, getEndpoints is called to provide defaults.
func getClientConfig(ctx context.Context, certificatesDir string, endpoints ...string) (*clientv3.Config, error) {
	config := &clientv3.Config{
		Endpoints:            endpoints,
		Context:              ctx,
		DialTimeout:          2 * time.Second,
		DialKeepAliveTime:    30 * time.Second,
		DialKeepAliveTimeout: 10 * time.Second,
		AutoSyncInterval:     10 * time.Second,
		PermitWithoutStream:  true,
	}

	var err error
	if strings.HasPrefix(endpoints[0], "https://") {
		config.TLS, err = toTLSConfig(certificatesDir)
	}
	return config, err
}

func toTLSConfig(certificatesDir string) (*tls.Config, error) {
	clientCert, err := tls.LoadX509KeyPair(
		filepath.Join(certificatesDir, certs.APIServerEtcdClientCertName),
		filepath.Join(certificatesDir, certs.APIServerEtcdClientKeyName),
	)
	if err != nil {
		return nil, err
	}

	pool, err := certutil.NewPool(filepath.Join(certificatesDir, certs.EtcdCACertName))
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{clientCert},
	}, nil
}
