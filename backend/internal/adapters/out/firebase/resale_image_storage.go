// backend/internal/adapters/out/firebase/resale_image_storage.go
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

const resaleImageStorageBucketEnv = "FIREBASE_STORAGE_BUCKET"

type ResaleImageStorage struct {
	Client     *gcs.Client
	BucketName string
	ownsClient bool
}

func NewResaleImageStorage(
	client *gcs.Client,
	bucketName string,
) *ResaleImageStorage {
	return &ResaleImageStorage{
		Client:     client,
		BucketName: bucketName,
		ownsClient: false,
	}
}

func NewResaleImageStorageFromEnv(
	ctx context.Context,
) (*ResaleImageStorage, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}

	bucketName := os.Getenv(
		resaleImageStorageBucketEnv,
	)

	if bucketName == "" {
		return nil, fmt.Errorf(
			"%s is required",
			resaleImageStorageBucketEnv,
		)
	}

	client, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"create cloud storage client: %w",
			err,
		)
	}

	return &ResaleImageStorage{
		Client:     client,
		BucketName: bucketName,
		ownsClient: true,
	}, nil
}

var _ applicationport.ResaleImageStorage = (*ResaleImageStorage)(nil)

func (s *ResaleImageStorage) DeleteAll(
	ctx context.Context,
	resaleID string,
) error {
	if s == nil {
		return errors.New(
			"resale image storage is nil",
		)
	}

	if s.Client == nil {
		return errors.New(
			"cloud storage client is nil",
		)
	}

	if ctx == nil {
		return errors.New(
			"context is nil",
		)
	}

	bucketName := s.BucketName

	if bucketName == "" {
		return errors.New(
			"cloud storage bucket name is empty",
		)
	}

	if resaleID == "" {
		return errors.New(
			"resaleID is required",
		)
	}

	if !utf8.ValidString(resaleID) {
		return errors.New(
			"resaleID must be valid UTF-8",
		)
	}

	if strings.ContainsRune(
		resaleID,
		'\x00',
	) {
		return errors.New(
			"resaleID contains a null character",
		)
	}

	if strings.ContainsAny(
		resaleID,
		"\r\n",
	) {
		return errors.New(
			"resaleID contains an invalid control character",
		)
	}

	if strings.ContainsAny(
		resaleID,
		"/\\:",
	) {
		return errors.New(
			"resaleID contains an invalid path separator",
		)
	}

	if resaleID == "." ||
		resaleID == ".." {
		return errors.New(
			"resaleID is invalid",
		)
	}

	prefix :=
		"resale-condition-images/" +
			resaleID +
			"/"

	bucket := s.Client.Bucket(
		bucketName,
	)

	it := bucket.Objects(
		ctx,
		&gcs.Query{
			Prefix: prefix,
		},
	)

	for {
		attrs, err := it.Next()
		if err != nil {
			if errors.Is(
				err,
				iterator.Done,
			) {
				break
			}

			return fmt.Errorf(
				"list cloud storage objects under prefix %q from bucket %q: %w",
				prefix,
				bucketName,
				err,
			)
		}

		if attrs == nil ||
			attrs.Name == "" {
			continue
		}

		err = bucket.Object(
			attrs.Name,
		).Delete(ctx)

		if err == nil {
			continue
		}

		if errors.Is(
			err,
			gcs.ErrObjectNotExist,
		) {
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

func (s *ResaleImageStorage) Close() error {
	if s == nil ||
		s.Client == nil ||
		!s.ownsClient {
		return nil
	}

	if err := s.Client.Close(); err != nil {
		return fmt.Errorf(
			"close cloud storage client: %w",
			err,
		)
	}

	s.Client = nil
	s.ownsClient = false

	return nil
}
