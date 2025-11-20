package file

import (
	"cloud.google.com/go/storage"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/api/option"
)

// StorageClient defines the methods required by FileController to work with any
// object storage backend.
type StorageClient interface {
	UploadFile(objectName string, fileData io.Reader) error
	DownloadFile(objectName string) (io.ReadCloser, int64, error)
}

// CloudStorageClient implements StorageClient using Google Cloud Storage.
type CloudStorageClient struct {
	BucketName string
	Ctx        context.Context
	Client     *storage.Client
}

const serviceAccountEnvVar = "CLOUD_STORAGE_SERVICE_ACCOUNT"

// NewCloudStorageClient initializes a CloudStorageClient for the provided bucket.
func NewCloudStorageClient(bucketName string) (*CloudStorageClient, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("CLOUD_STORAGE_BUCKET is required when cloud storage is enabled")
	}
	ctx := context.Background()

	credsJSON, err := loadServiceAccountJSON()
	if err != nil {
		return nil, err
	}

	var client *storage.Client
	if len(credsJSON) > 0 {
		client, err = storage.NewClient(ctx, option.WithCredentialsJSON(credsJSON))
	} else {
		client, err = storage.NewClient(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud storage client: %w", err)
	}
	return &CloudStorageClient{
		BucketName: bucketName,
		Ctx:        ctx,
		Client:     client,
	}, nil
}

// UploadFile streams the provided reader into the named object in the bucket.
func (c *CloudStorageClient) UploadFile(objectName string, fileData io.Reader) error {
	bucket := c.Client.Bucket(c.BucketName)
	obj := bucket.Object(objectName)
	wc := obj.NewWriter(c.Ctx)
	if _, err := io.Copy(wc, fileData); err != nil {
		return fmt.Errorf("failed to write data to object: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("failed to close object writer: %w", err)
	}
	return nil
}

// DownloadFile retrieves an object reader and reported size for the given object.
func (c *CloudStorageClient) DownloadFile(objectName string) (io.ReadCloser, int64, error) {
	bucket := c.Client.Bucket(c.BucketName)
	obj := bucket.Object(objectName)
	rc, err := obj.NewReader(c.Ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create object reader: %w", err)
	}
	return rc, rc.Attrs.Size, nil
}

func loadServiceAccountJSON() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(serviceAccountEnvVar))
	if raw == "" {
		return nil, nil
	}

	if strings.HasPrefix(raw, "{") {
		return []byte(raw), nil
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", serviceAccountEnvVar, err)
	}
	return decoded, nil
}
