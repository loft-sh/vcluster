/*
Copyright the Velero contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package s3

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-logr/logr"
	"github.com/loft-sh/vcluster/pkg/snapshot/types"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

type Options struct {
	Bucket string `json:"bucket,omitempty"`
	Key    string `json:"key,omitempty"`

	SkipClientCredentials bool `json:"skip-client-credentials,omitempty" url:"skip-client-credentials"`

	AccessKeyID     string `json:"access-key-id,omitempty" url:"access-key-id,base64"`
	SecretAccessKey string `json:"secret-access-key,omitempty" url:"secret-access-key,base64"`
	SessionToken    string `json:"session-token,omitempty" url:"session-token,base64"`

	Region    string `json:"region,omitempty" url:"region"`
	Profile   string `json:"profile,omitempty" url:"profile"`
	S3URL     string `json:"url,omitempty" url:"url,base64"`
	PublicURL string `json:"public-url,omitempty" url:"public-url,base64"`
	KmsKeyID  string `json:"kms-key-id,omitempty" url:"kms-key-id,base64"`
	Tagging   string `json:"tagging,omitempty" url:"tagging,base64"`

	S3ForcePathStyle      bool `json:"force-path-style,omitempty" url:"force-path-style"`
	InsecureSkipTLSVerify bool `json:"insecure-skip-tls-verify,omitempty" url:"insecure-skip-tls-verify"`

	CustomerKeyEncryptionFile string `json:"custom-key-encryption-file,omitempty" url:"custom-key-encryption-file,base64"`
	CredentialsFile           string `json:"credentials-file,omitempty" url:"credentials-file,base64"`
	ServerSideEncryption      string `json:"server-side-encryption,omitempty" url:"server-side-encryption,base64"`
	CaCert                    string `json:"ca-cert,omitempty" url:"ca-cert,base64"`
	ChecksumAlgorithm         string `json:"checksum-algorithm,omitempty" url:"checksum-algorithm"`
}

func (o *Options) FillCredentials(isClient bool) {
	if (isClient && o.SkipClientCredentials) || o.Bucket == "" || o.AccessKeyID != "" {
		return
	}

	defaultConfig, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		return
	}

	credentials, err := defaultConfig.Credentials.Retrieve(context.Background())
	if err != nil {
		return
	}

	o.AccessKeyID = credentials.AccessKeyID
	o.SecretAccessKey = credentials.SecretAccessKey
	o.SessionToken = credentials.SessionToken
}

type s3Interface interface {
	HeadObject(ctx context.Context, input *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type s3PresignInterface interface {
	PresignGetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(options *s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

type ObjectStore struct {
	log        logr.Logger
	s3         s3Interface
	preSignS3  s3PresignInterface
	s3Uploader *manager.Uploader
	bucket     string
	key        string
	region     string

	kmsKeyID             string
	sseCustomerKey       string
	sseCustomerKeyMd5    string
	serverSideEncryption string
	tagging              string
	checksumAlg          string
}

func NewStore(logger logr.Logger) *ObjectStore {
	return &ObjectStore{log: logger}
}

func (o *ObjectStore) Init(config *Options) error {
	if config.AccessKeyID != "" {
		_ = os.Setenv("AWS_ACCESS_KEY_ID", config.AccessKeyID)
	}
	if config.SecretAccessKey != "" {
		_ = os.Setenv("AWS_SECRET_ACCESS_KEY", config.SecretAccessKey)
	}
	if config.SessionToken != "" {
		_ = os.Setenv("AWS_SESSION_TOKEN", config.SessionToken)
	}
	cfg, err := newConfigBuilder(o.log).WithRegion(config.Region).
		WithProfile(config.Profile).
		WithCredentialsFile(config.CredentialsFile).
		WithTLSSettings(config.InsecureSkipTLSVerify, config.CaCert).Build()
	if err != nil {
		return errors.WithStack(err)
	}

	// AWS (not an alternate S3-compatible API) and region not
	// explicitly specified: determine the bucket's region
	// GetBucketRegion will attempt to get the region for a bucket using the
	// client's configured region to determine which AWS partition to perform the query on.
	if config.S3URL == "" && config.Region == "" {
		regionClient, err := newS3Client(cfg, config.S3URL, config.S3ForcePathStyle)
		if err != nil {
			return errors.WithStack(err)
		}
		config.Region, err = manager.GetBucketRegion(context.Background(), regionClient, config.Bucket, func(o *s3.Options) { o.Region = "us-east-1" })
		if err != nil {
			o.log.Error(err, fmt.Sprintf("Failed to determine bucket's region bucket: %s", config.Bucket))
			return err
		}
		if config.Region == "" {
			return fmt.Errorf("unable to determine bucket's region, bucket: %s", config.Bucket)
		}
		cfg.Region = config.Region
	}

	client, err := newS3Client(cfg, config.S3URL, config.S3ForcePathStyle)
	if err != nil {
		return errors.WithStack(err)
	}
	o.s3 = client
	o.s3Uploader = manager.NewUploader(client)
	o.region = config.Region
	o.kmsKeyID = config.KmsKeyID
	o.serverSideEncryption = config.ServerSideEncryption
	o.tagging = config.Tagging
	o.key = config.Key
	o.bucket = config.Bucket

	if config.CustomerKeyEncryptionFile != "" && config.KmsKeyID != "" {
		return errors.New("you cannot use kms key and encryption file at the same time")
	}

	if config.CustomerKeyEncryptionFile != "" {
		customerKey, err := readCustomerKey(config.CustomerKeyEncryptionFile)
		if err != nil {
			return err
		}
		o.sseCustomerKey = base64.StdEncoding.EncodeToString([]byte(customerKey))
		hash := md5.Sum([]byte(customerKey))
		o.sseCustomerKeyMd5 = base64.StdEncoding.EncodeToString(hash[:])
	}

	if config.PublicURL != "" {
		publicClient, err := newS3Client(cfg, config.PublicURL, config.S3ForcePathStyle)
		if err != nil {
			return err
		}

		o.preSignS3 = s3.NewPresignClient(publicClient)
	} else {
		o.preSignS3 = s3.NewPresignClient(client)
	}
	if config.ChecksumAlgorithm != "" {
		if !validChecksumAlg(config.ChecksumAlgorithm) {
			return errors.Errorf("invalid checksum algorithm: %s", config.ChecksumAlgorithm)
		}
		o.checksumAlg = config.ChecksumAlgorithm
	} else {
		o.checksumAlg = string(s3types.ChecksumAlgorithmCrc32)
	}
	return nil
}

func validChecksumAlg(alg string) bool {
	typedAlg := s3types.ChecksumAlgorithm(alg)
	return alg == "" || slices.Contains(typedAlg.Values(), typedAlg)
}

func readCustomerKey(customerKeyEncryptionFile string) (string, error) {
	if _, err := os.Stat(customerKeyEncryptionFile); err != nil {
		if os.IsNotExist(err) {
			return "", errors.Wrapf(err, "provided key encryption file does not exist: %s", customerKeyEncryptionFile)
		}
		return "", errors.Wrapf(err, "could not stat %s", customerKeyEncryptionFile)
	}

	fileHandle, err := os.Open(customerKeyEncryptionFile)
	if err != nil {
		return "", errors.Wrapf(err, "could not read %s", customerKeyEncryptionFile)
	}

	keyBytes := make([]byte, 32)
	nBytes, err := fileHandle.Read(keyBytes)
	if err != nil {
		return "", errors.Wrapf(err, "could not read %s", customerKeyEncryptionFile)
	}
	fileHandle.Close()

	if nBytes != 32 {
		return "", errors.Errorf("contents of %s are not exactly 32 bytes", customerKeyEncryptionFile)
	}

	key := string(keyBytes)
	return key, nil
}

func (o *ObjectStore) Target() string {
	return toS3URL(o.bucket, o.key, o.region)
}

func (o *ObjectStore) PutObject(ctx context.Context, body io.Reader) error {
	input := &s3.PutObjectInput{
		Bucket:  aws.String(o.bucket),
		Key:     aws.String(o.key),
		Body:    &wrapper{body},
		Tagging: aws.String(o.tagging),
	}

	switch {
	// if kmsKeyID is not empty, assume a server-side encryption (SSE)
	// algorithm of "aws:kms"
	case o.kmsKeyID != "":
		klog.FromContext(ctx).Info("using aws:kms server-side encryption (SSE-KMS)", "kmsKeyId", o.kmsKeyID)
		input.ServerSideEncryption = "aws:kms"
		input.SSEKMSKeyId = &o.kmsKeyID
	// if sseCustomerKey is not empty, assume SSE-C encryption with AES256 algorithm
	case o.sseCustomerKey != "":
		klog.FromContext(ctx).Info("using aws server-side encryption (SSE-C) with customer provided key")
		input.SSECustomerAlgorithm = aws.String("AES256")
		input.SSECustomerKey = &o.sseCustomerKey
		input.SSECustomerKeyMD5 = &o.sseCustomerKeyMd5
	// otherwise, use the SSE algorithm specified, if any
	case o.serverSideEncryption != "":
		klog.FromContext(ctx).Info("using aws server-side encryption (SSE)", "server-side-encryption", o.serverSideEncryption)
		input.ServerSideEncryption = s3types.ServerSideEncryption(o.serverSideEncryption)
	}

	if o.checksumAlg != "" {
		input.ChecksumAlgorithm = s3types.ChecksumAlgorithm(o.checksumAlg)
	}

	_, err := o.s3Uploader.Upload(ctx, input)
	return errors.Wrapf(err, "error putting object %s", o.key)
}

func (o *ObjectStore) GetObject(ctx context.Context) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(o.key),
	}

	if o.sseCustomerKey != "" {
		input.SSECustomerAlgorithm = aws.String("AES256")
		input.SSECustomerKey = &o.sseCustomerKey
		input.SSECustomerKeyMD5 = &o.sseCustomerKeyMd5
	}

	output, err := o.s3.GetObject(ctx, input)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting object %s", o.key)
	}

	return output.Body, nil
}

func (o *ObjectStore) List(ctx context.Context) ([]types.Snapshot, error) {
	prefix := o.key
	if strings.HasSuffix(prefix, "tar.gz") {
		// Use the "parent dir" as the prefix if a file was given
		prefix = filepath.Dir(prefix)

		// Handle if the key is at the root of the bucket.
		if prefix == "." {
			prefix = ""
		}
	}

	paginator := s3.NewListObjectsV2Paginator(o.s3, &s3.ListObjectsV2Input{
		Bucket: aws.String(o.bucket),
		Prefix: aws.String(prefix),
	})

	snapshots := make([]types.Snapshot, 0)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, obj := range output.Contents {
			if obj.Key == nil || obj.LastModified == nil {
				continue
			}

			// Skip non *.tar.gz objects
			if !strings.HasSuffix(*obj.Key, "tar.gz") {
				continue
			}

			// Skip objects not in the "current directory"
			id := strings.TrimPrefix(strings.TrimPrefix(*obj.Key, prefix), "/")
			if filepath.Dir(id) != "." {
				continue
			}

			// ID is the relative object name
			snapshots = append(snapshots, types.Snapshot{
				ID:        id,
				URL:       toS3URL(o.bucket, *obj.Key, o.region),
				Timestamp: *obj.LastModified,
			})
		}
	}
	return snapshots, nil
}

func (o *ObjectStore) Delete(ctx context.Context) error {
	_, err := o.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(o.key),
	})
	return err
}

// this is required because os pipes cause trouble with aws uploader
type wrapper struct {
	io.Reader
}

func toS3URL(bucket, key, region string) string {
	url := "s3://" + bucket + "/" + key
	if region != "" {
		url += "?region=" + region
	}
	return url
}
