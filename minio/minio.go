// Package pkgminio is general utils for minio storage
package pkgminio

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient interface {
	Connect() error
	GetObject(
		ctx context.Context,
		fileName string,
		bucketName *string,
		objOpt *minio.GetObjectOptions,
	) (*minio.Object, error)
	Upload(
		ctx context.Context,
		bucketName *string,
		directory, filename string,
		file multipart.File,
		header *multipart.FileHeader,
	) (*string, error)
	StatObject(
		ctx context.Context,
		bucketName *string,
		fileName string,
	) (*minio.ObjectInfo, error)
	BucketCheck(ctx context.Context, bucketName *string) (bool, error)
	CreateBucket(ctx context.Context, bucketName string) error
	CopyDoc(
		ctx context.Context,
		bucketName, objectName string,
		reader io.Reader, objectSize int64,
		opts minio.PutObjectOptions,
	) (*string, error)

	Remove(
		ctx context.Context,
		fileName string,
		bucketName *string,
	) error
}

func NewMinioClient(opts ...OptFunc) MinioClient {
	newConf := configMinio{}
	for _, opt := range opts {
		opt(&newConf)
	}

	newConf.bucketName = "mekdi_dev"
	if newConf.isLocal {
		newConf.bucketName = "mekdi-dev"
	}

	c := commonMinio{
		configMinio: newConf,
	}

	return &c
}

func (c *commonMinio) Connect() error {

	httpTransport := &http.Transport{
		MaxIdleConns:        c.iddleConn,
		MaxIdleConnsPerHost: c.iddleConnHost,
		IdleConnTimeout:     time.Duration(c.iddleConnTimeout) * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(c.tlsHandShakeTimeout) * time.Second,
			KeepAlive: time.Duration(c.iddleConnAlive) * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: time.Duration(c.tlsHandShakeTimeout) * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.tlsSkipVerify,
		},
	}
	// Initialize minio client object.
	minioClient, err := minio.New(c.endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(c.accessKeyID, c.secretAccessKey, ""),
		Secure:    c.useSSL,
		Transport: httpTransport,
	})
	if err != nil {
		return err
	}

	// check connection
	_, err = minioClient.ListBuckets(context.Background())
	if err != nil {
		msg := fmt.Sprintf("failed connect minio %s:", err.Error())
		return errors.New(msg)
	}

	c.minioClient = minioClient

	return nil
}

func (c *commonMinio) GetObject(
	ctx context.Context,
	fileName string,
	bucketName *string,
	objOpt *minio.GetObjectOptions,
) (*minio.Object, error) {
	if bucketName != nil {
		c.bucketName = *bucketName
	}
	var objectOption minio.GetObjectOptions
	if objOpt != nil {
		objectOption = *objOpt
	}

	object, err := c.minioClient.GetObject(ctx, c.bucketName, fileName, objectOption)
	if err != nil {
		return nil, err
	}

	return object, nil
}

func (c *commonMinio) Upload(
	ctx context.Context,
	bucketName *string,
	directory, filename string,
	file multipart.File,
	header *multipart.FileHeader,
) (*string, error) {

	if bucketName != nil {
		c.bucketName = *bucketName
	}

	// calculate checksum before upload
	checksum, err := calculateSHA256FromMultipartFile(file)
	if err != nil {
		return nil, err
	}
	file.Seek(0, io.SeekStart)

	filenameWithDir := strings.TrimSuffix(directory, "/") + "/" + filename

	_, err = c.minioClient.PutObject(
		ctx,
		c.bucketName,
		filenameWithDir,
		file,
		header.Size,
		minio.PutObjectOptions{
			ContentType: header.Header.Get("Content-Type"),
			UserMetadata: map[string]string{
				"sha256": checksum,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return &checksum, nil
}

func (c *commonMinio) StatObject(
	ctx context.Context,
	bucketName *string,
	fileName string,
) (*minio.ObjectInfo, error) {

	if bucketName != nil {
		c.bucketName = *bucketName
	}

	info, err := c.minioClient.StatObject(ctx, c.bucketName, fileName, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *commonMinio) BucketCheck(ctx context.Context, bucketName *string) (bool, error) {
	if bucketName != nil {
		c.bucketName = *bucketName
	}

	exists, err := c.minioClient.BucketExists(ctx, c.bucketName)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (c *commonMinio) CreateBucket(ctx context.Context, bucketName string) error {
	location := ""

	exists, err := c.minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	err = c.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		return err
	}

	return nil
}

func (c *commonMinio) CopyDoc(
	ctx context.Context,
	bucketName, objectName string,
	reader io.Reader, objectSize int64,
	opts minio.PutObjectOptions,
) (*string, error) {

	_, err := c.minioClient.PutObject(
		ctx,
		bucketName,
		objectName,
		reader,
		objectSize,
		opts,
	)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *commonMinio) Remove(
	ctx context.Context,
	fileName string,
	bucketName *string,
) error {
	if bucketName != nil {
		c.bucketName = *bucketName
	}
	err := c.minioClient.RemoveObject(
		ctx,
		c.bucketName,
		fileName,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return err
	}

	return nil
}
