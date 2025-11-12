package file

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"io"
)

// StorageClient defines the methods required by FileController to work with any
// object storage backend.
type StorageClient interface {
	UploadFile(objectName string, fileData io.Reader) error
	DownloadFile(objectName string) (io.ReadCloser, int64, error)
}

type CloudStorageClient struct {
	BucketName string
	Ctx        context.Context
	Client     *storage.Client
}

func NewCloudStorageClient(bucketName string) (*CloudStorageClient, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("CLOUD_STORAGE_BUCKET is required when cloud storage is enabled")
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud storage client: %v", err)
	}
	return &CloudStorageClient{
		BucketName: bucketName,
		Ctx:        ctx,
		Client:     client,
	}, nil
}

func (c *CloudStorageClient) UploadFile(objectName string, fileData io.Reader) error {
	bucket := c.Client.Bucket(c.BucketName)
	obj := bucket.Object(objectName)
	wc := obj.NewWriter(c.Ctx)
	if _, err := io.Copy(wc, fileData); err != nil {
		return fmt.Errorf("failed to write data to object: %v", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("failed to close object writer: %v", err)
	}
	return nil
}

func (c *CloudStorageClient) DownloadFile(objectName string) (io.ReadCloser, int64, error) {
	bucket := c.Client.Bucket(c.BucketName)
	obj := bucket.Object(objectName)
	rc, err := obj.NewReader(c.Ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create object reader: %v", err)
	}
	return rc, rc.Attrs.Size, nil
}
