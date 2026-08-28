// backend/internal/adapters/out/firebase/list_save_operation_storage.go
package firebase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	applicationport "narratives/internal/application/port"
)

const listSaveOperationStorageBucketEnv = "FIREBASE_STORAGE_BUCKET"

type ListSaveOperationStorage struct {
	Client     *gcs.Client
	BucketName string
	ownsClient bool
}

func NewListSaveOperationStorage(client *gcs.Client, bucketName string) *ListSaveOperationStorage {
	return &ListSaveOperationStorage{
		Client:     client,
		BucketName: bucketName,
		ownsClient: false,
	}
}

func NewListSaveOperationStorageFromEnv(ctx context.Context) (*ListSaveOperationStorage, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}

	bucketName := os.Getenv(listSaveOperationStorageBucketEnv)
	if bucketName == "" {
		return nil, fmt.Errorf("%s is required", listSaveOperationStorageBucketEnv)
	}

	client, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create cloud storage client: %w", err)
	}

	return &ListSaveOperationStorage{
		Client:     client,
		BucketName: bucketName,
		ownsClient: true,
	}, nil
}

var _ applicationport.ListSaveOperationStorage = (*ListSaveOperationStorage)(nil)

func (s *ListSaveOperationStorage) Exists(ctx context.Context, storagePath string) (bool, error) {
	bucketName, objectPath, err := s.resolveObjectPath(ctx, storagePath)
	if err != nil {
		return false, err
	}

	_, err = s.Client.Bucket(bucketName).Object(objectPath).Attrs(ctx)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, gcs.ErrObjectNotExist) {
		return false, nil
	}

	return false, fmt.Errorf(
		"get cloud storage object attributes %q from bucket %q: %w",
		objectPath,
		bucketName,
		err,
	)
}

func (s *ListSaveOperationStorage) Delete(ctx context.Context, storagePath string) error {
	bucketName, objectPath, err := s.resolveObjectPath(ctx, storagePath)
	if err != nil {
		return err
	}

	err = s.Client.Bucket(bucketName).Object(objectPath).Delete(ctx)
	if err == nil {
		return nil
	}

	if errors.Is(err, gcs.ErrObjectNotExist) {
		return nil
	}

	return fmt.Errorf(
		"delete cloud storage object %q from bucket %q: %w",
		objectPath,
		bucketName,
		err,
	)
}

func (s *ListSaveOperationStorage) DeleteAll(ctx context.Context, listID string) error {
	if s == nil {
		return errors.New("list save operation storage is nil")
	}

	if s.Client == nil {
		return errors.New("cloud storage client is nil")
	}

	if ctx == nil {
		return errors.New("context is nil")
	}

	bucketName := s.BucketName
	if bucketName == "" {
		return errors.New("cloud storage bucket name is empty")
	}

	if listID == "" {
		return errors.New("listID is required")
	}

	if !utf8.ValidString(listID) {
		return errors.New("listID must be valid UTF-8")
	}

	if strings.ContainsRune(listID, '\x00') {
		return errors.New("listID contains a null character")
	}

	if strings.ContainsAny(listID, "\r\n") {
		return errors.New("listID contains an invalid control character")
	}

	if strings.ContainsAny(listID, "/\\:") {
		return errors.New("listID contains an invalid path separator")
	}

	if listID == "." || listID == ".." {
		return errors.New("listID is invalid")
	}

	prefix := "lists/" + listID + "/images/"
	bucket := s.Client.Bucket(bucketName)

	it := bucket.Objects(
		ctx,
		&gcs.Query{
			Prefix: prefix,
		},
	)

	for {
		attrs, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}

			return fmt.Errorf(
				"list cloud storage objects under prefix %q from bucket %q: %w",
				prefix,
				bucketName,
				err,
			)
		}

		if attrs == nil || attrs.Name == "" {
			continue
		}

		err = bucket.Object(attrs.Name).Delete(ctx)
		if err == nil {
			continue
		}

		if errors.Is(err, gcs.ErrObjectNotExist) {
			continue
		}

		return fmt.Errorf(
			"delete cloud storage object %q from bucket %q: %w",
			attrs.Name,
			bucketName,
			err,
		)
	}

	return nil
}

func (s *ListSaveOperationStorage) Close() error {
	if s == nil || s.Client == nil || !s.ownsClient {
		return nil
	}

	if err := s.Client.Close(); err != nil {
		return fmt.Errorf("close cloud storage client: %w", err)
	}

	s.Client = nil
	s.ownsClient = false

	return nil
}

func (s *ListSaveOperationStorage) resolveObjectPath(ctx context.Context, storagePath string) (string, string, error) {
	if s == nil {
		return "", "", errors.New("list save operation storage is nil")
	}

	if s.Client == nil {
		return "", "", errors.New("cloud storage client is nil")
	}

	if ctx == nil {
		return "", "", errors.New("context is nil")
	}

	bucketName := s.BucketName
	if bucketName == "" {
		return "", "", errors.New("cloud storage bucket name is empty")
	}

	objectPath, err := normalizeListSaveOperationObjectPath(storagePath)
	if err != nil {
		return "", "", err
	}

	return bucketName, objectPath, nil
}

func normalizeListSaveOperationObjectPath(storagePath string) (string, error) {
	value := storagePath
	if value == "" {
		return "", errors.New("storagePath is required")
	}

	if !utf8.ValidString(value) {
		return "", errors.New("storagePath must be valid UTF-8")
	}

	if strings.ContainsRune(value, '\x00') {
		return "", errors.New("storagePath contains a null character")
	}

	if strings.ContainsAny(value, "\r\n") {
		return "", errors.New("storagePath contains an invalid control character")
	}

	lowerValue := strings.ToLower(value)
	if strings.HasPrefix(lowerValue, "gs://") ||
		strings.HasPrefix(lowerValue, "http://") ||
		strings.HasPrefix(lowerValue, "https://") {
		return "", errors.New("storagePath must be an object path, not a URL")
	}

	value = strings.TrimLeft(value, "/")
	parts := strings.Split(value, "/")

	if len(parts) != 5 {
		return "", errors.New("storagePath must match lists/{listId}/images/{imageId}/{fileName}")
	}

	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("storagePath contains an invalid path segment")
		}

		if part != strings.Trim(part, " \t") {
			return "", errors.New("storagePath contains surrounding whitespace")
		}
	}

	if parts[0] != "lists" {
		return "", errors.New("storagePath must start with lists/")
	}

	if parts[2] != "images" {
		return "", errors.New("storagePath must contain the images directory")
	}

	if strings.ContainsAny(parts[1], "\\:") {
		return "", errors.New("storagePath listId is invalid")
	}

	if strings.ContainsAny(parts[3], "\\:") {
		return "", errors.New("storagePath imageId is invalid")
	}

	if strings.Contains(parts[4], "\\") {
		return "", errors.New("storagePath fileName is invalid")
	}

	return value, nil
}
