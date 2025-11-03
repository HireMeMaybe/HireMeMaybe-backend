package file

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"google.golang.org/api/iterator"
	"io"
	"log"
	"os"
)

type CloudStorageClient struct {
	BucketName string
	Ctx        context.Context
	Client     *storage.Client
}

func NewCloudStorageClient(bucketName string) (*CloudStorageClient, error) {
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