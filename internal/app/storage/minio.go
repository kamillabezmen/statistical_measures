package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	minioEndpoint = "localhost:9000"
	minioUser     = "admin"
	minioPassword = "admin12345"
	bucketName    = "statistical-measures"
)

func newClient() (*minio.Client, error) {
	return minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioUser, minioPassword, ""),
		Secure: false,
	})
}

// UploadFile сохраняет файл в MinIO.
// Имя генерируется на латинице и сохраняется затем в БД.
func UploadFile(
	ctx context.Context,
	fileHeader *multipart.FileHeader,
	prefix string,
) (string, error) {

	client, err := newClient()
	if err != nil {
		return "", err
	}

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return "", err
	}

	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return "", err
		}
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	extension := strings.ToLower(filepath.Ext(fileHeader.Filename))

	fileName := fmt.Sprintf(
		"%s_%d%s",
		prefix,
		time.Now().UnixNano(),
		extension,
	)

	_, err = client.PutObject(
		ctx,
		bucketName,
		fileName,
		file,
		fileHeader.Size,
		minio.PutObjectOptions{
			ContentType: fileHeader.Header.Get("Content-Type"),
		},
	)

	if err != nil {
		return "", err
	}

	return fileName, nil
}
