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
	"slices"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-logr/logr"
	"github.com/spf13/pflag"

	"github.com/pkg/errors"
)

type Options struct {
	Region    string `json:"s3-region"`
	Bucket    string `json:"s3-bucket"`
	Key       string `json:"s3-key"`
	Profile   string `json:"s3-profile"`
	S3URL     string `json:"s3-url"`
	PublicURL string `json:"s3-public-url"`
	KmsKeyID  string `json:"s3-kms-key-id"`
	Tagging   string `json:"s3-tagging"`

	S3ForcePathStyle      bool `json:"s3-force-path-style"`
	InsecureSkipTLSVerify bool `json:"s3-insecure-skip-tls-verify"`

	CustomerKeyEncryptionFile string `json:"s3-custom-key-encryption-file"`
	CredentialsFile           string `json:"s3-credentials-file"`
	ServerSideEncryption      string `json:"s3-server-side-encryption"`
	CaCert                    string `json:"s3-ca-cert"`
	ChecksumAlgorithm         string `json:"s3-checksum-algorithm"`
}

func AddS3Flags(fs *pflag.FlagSet, s3Options *Options) {
	fs.StringVar(&s3Options.Region, "s3-region", s3Options.Region, "The s3 region to use")
	fs.StringVar(&s3Options.Key, "s3-key", s3Options.Key, "The key where to save the snapshot in the bucket")
	fs.StringVar(&s3Options.Bucket, "s3-bucket", s3Options.Bucket, "The s3 bucket to use")
	fs.StringVar(&s3Options.Profile, "s3-profile", s3Options.Profile, "The aws profile to use")
	fs.StringVar(&s3Options.S3URL, "s3-url", s3Options.S3URL, "The s3 url to use")
	fs.StringVar(&s3Options.PublicURL, "s3-public-url", s3Options.PublicURL, "The s3 public url to use")
	fs.StringVar(&s3Options.KmsKeyID, "s3-kms-key-id", s3Options.KmsKeyID, "The s3 kms key id to use")
	fs.StringVar(&s3Options.Tagging, "s3-tags", s3Options.Tagging, "The s3 tags to use")
	fs.BoolVar(&s3Options.S3ForcePathStyle, "s3-force-path-style", s3Options.S3ForcePathStyle, "If s3 path style should be forced")
	fs.BoolVar(&s3Options.InsecureSkipTLSVerify, "s3-insecure-skip-tls-verify", s3Options.InsecureSkipTLSVerify, "If s3 connection should be insecure")
	fs.StringVar(&s3Options.CredentialsFile, "s3-credentials-file", s3Options.CredentialsFile, "The credentials file to use when connecting to s3")
	fs.StringVar(&s3Options.CaCert, "s3-ca-cert", s3Options.CaCert, "The ca cert to use when connecting to s3")
	fs.StringVar(&s3Options.ChecksumAlgorithm, "s3-checksum-algorithm", s3Options.ChecksumAlgorithm, "The checksum algorithm to use")
	fs.StringVar(&s3Options.ServerSideEncryption, "s3-server-side-encryption", s3Options.ServerSideEncryption, "The server side encryption that is used")
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

	kmsKeyID             string
	sseCustomerKey       string
	sseCustomerKeyMd5    string
	signatureVersion     string
	serverSideEncryption string
	tagging              string
	checksumAlg          string
}

func NewObjectStore(logger logr.Logger) *ObjectStore {
	return &ObjectStore{log: logger}
}

func (o *ObjectStore) Init(config *Options) error {
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
		o.checksumAlg = string(types.ChecksumAlgorithmCrc32)
	}
	return nil
}

func validChecksumAlg(alg string) bool {
	typedAlg := types.ChecksumAlgorithm(alg)
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
	return "s3://" + o.bucket + "/" + o.key
}

func (o *ObjectStore) PutObject(body io.Reader) error {
	input := &s3.PutObjectInput{
		Bucket:  aws.String(o.bucket),
		Key:     aws.String(o.key),
		Body:    body,
		Tagging: aws.String(o.tagging),
	}

	switch {
	// if kmsKeyID is not empty, assume a server-side encryption (SSE)
	// algorithm of "aws:kms"
	case o.kmsKeyID != "":
		input.ServerSideEncryption = "aws:kms"
		input.SSEKMSKeyId = &o.kmsKeyID
	// if sseCustomerKey is not empty, assume SSE-C encryption with AES256 algorithm
	case o.sseCustomerKey != "":
		input.SSECustomerAlgorithm = aws.String("AES256")
		input.SSECustomerKey = &o.sseCustomerKey
		input.SSECustomerKeyMD5 = &o.sseCustomerKeyMd5
	// otherwise, use the SSE algorithm specified, if any
	case o.serverSideEncryption != "":
		input.ServerSideEncryption = types.ServerSideEncryption(o.serverSideEncryption)
	}

	if o.checksumAlg != "" {
		input.ChecksumAlgorithm = types.ChecksumAlgorithm(o.checksumAlg)
	}

	_, err := o.s3Uploader.Upload(context.Background(), input)

	return errors.Wrapf(err, "error putting object %s", o.key)
}

// ObjectExists checks if there is an object with the given key in the object storage bucket.
func (o *ObjectStore) ObjectExists() (bool, error) {
	log := o.log.WithValues("bucket", o.bucket, "key", o.key)
	input := &s3.HeadObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(o.key),
	}

	if o.sseCustomerKey != "" {
		input.SSECustomerAlgorithm = aws.String("AES256")
		input.SSECustomerKey = &o.sseCustomerKey
		input.SSECustomerKeyMD5 = &o.sseCustomerKeyMd5
	}

	log.V(1).Info("Checking if object exists")
	if _, err := o.s3.HeadObject(context.Background(), input); err != nil {
		log.V(1).Info("Checking for AWS specific error information")
		var ne *types.NotFound
		if errors.As(err, &ne) {
			log.WithValues(
				"code", ne.ErrorCode(),
				"message", ne.ErrorMessage(),
			).V(1).Info("Object doesn't exist - got not found")
			return false, nil
		}
		return false, errors.WithStack(err)
	}

	log.V(1).Info("Object exists")
	return true, nil
}

func (o *ObjectStore) GetObject() (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(o.key),
	}

	if o.sseCustomerKey != "" {
		input.SSECustomerAlgorithm = aws.String("AES256")
		input.SSECustomerKey = &o.sseCustomerKey
		input.SSECustomerKeyMD5 = &o.sseCustomerKeyMd5
	}

	output, err := o.s3.GetObject(context.Background(), input)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting object %s", o.key)
	}

	return output.Body, nil
}

func (o *ObjectStore) ListCommonPrefixes(bucket, prefix, delimiter string) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimiter),
	}
	var ret []string
	p := s3.NewListObjectsV2Paginator(o.s3, input)
	for p.HasMorePages() {
		page, err := p.NextPage(context.Background())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		for _, prefix := range page.CommonPrefixes {
			ret = append(ret, *prefix.Prefix)
		}
	}
	return ret, nil
}

func (o *ObjectStore) ListObjects(bucket, prefix string) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	var ret []string
	p := s3.NewListObjectsV2Paginator(o.s3, input)
	for p.HasMorePages() {
		page, err := p.NextPage(context.Background())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		for _, obj := range page.Contents {
			ret = append(ret, *obj.Key)
		}
	}
	// ensure that returned objects are in a consistent order so that the deletion logic deletes the objects before
	// the pseudo-folder prefix object for s3 providers (such as Quobyte) that return the pseudo-folder as an object.
	// See https://github.com/vmware-tanzu/velero/pull/999
	sort.Sort(sort.Reverse(sort.StringSlice(ret)))

	return ret, nil
}

func (o *ObjectStore) DeleteObject(bucket, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := o.s3.DeleteObject(context.Background(), input)

	return errors.Wrapf(err, "error deleting object %s", key)
}
