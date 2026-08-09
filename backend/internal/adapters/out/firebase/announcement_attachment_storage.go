// backend/internal/adapters/out/firebase/announcement_attachment_storage.go
package firebase

import (
	"context"
	"errors"
	"fmt"
	usecase "narratives/internal/application/usecase"
	"os"
	"strings"
	"unicode/utf8"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

const announcementAttachmentStorageBucketEnv = "FIREBASE_STORAGE_BUCKET"

type AnnouncementAttachmentStorage struct {
	Client     *gcs.Client
	BucketName string
	ownsClient bool
}

func NewAnnouncementAttachmentStorage(
	client *gcs.Client,
	bucketName string,
) *AnnouncementAttachmentStorage {
	return &AnnouncementAttachmentStorage{
		Client:     client,
		BucketName: bucketName,
		ownsClient: false,
	}
}

func NewAnnouncementAttachmentStorageFromEnv(
	ctx context.Context,
) (*AnnouncementAttachmentStorage, error) {
	if ctx == nil {
		return nil, errors.New(
			"context is nil",
		)
	}

	bucketName := os.Getenv(
		announcementAttachmentStorageBucketEnv,
	)

	if bucketName == "" {
		return nil, fmt.Errorf(
			"%s is required",
			announcementAttachmentStorageBucketEnv,
		)
	}

	client, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"create cloud storage client: %w",
			err,
		)
	}

	return &AnnouncementAttachmentStorage{
		Client:     client,
		BucketName: bucketName,
		ownsClient: true,
	}, nil
}

var _ usecase.AnnouncementAttachmentStorage = (*AnnouncementAttachmentStorage)(nil)

func (s *AnnouncementAttachmentStorage) DeleteAll(
	ctx context.Context,
	announcementID string,
) error {
	if s == nil {
		return errors.New(
			"announcement attachment storage is nil",
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

	if announcementID == "" {
		return errors.New(
			"announcementID is required",
		)
	}

	if !utf8.ValidString(announcementID) {
		return errors.New(
			"announcementID must be valid UTF-8",
		)
	}

	if strings.ContainsRune(
		announcementID,
		'\x00',
	) {
		return errors.New(
			"announcementID contains a null character",
		)
	}

	if strings.ContainsAny(
		announcementID,
		"\r\n",
	) {
		return errors.New(
			"announcementID contains an invalid control character",
		)
	}

	if strings.ContainsAny(
		announcementID,
		"/\\:",
	) {
		return errors.New(
			"announcementID contains an invalid path separator",
		)
	}

	if announcementID == "." ||
		announcementID == ".." {
		return errors.New(
			"announcementID is invalid",
		)
	}

	prefix :=
		"announcements/" +
			announcementID +
			"/attachments/"

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
				"list announcement attachment storage objects under prefix %q from bucket %q: %w",
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
			"delete announcement attachment storage object %q from bucket %q: %w",
			attrs.Name,
			bucketName,
			err,
		)
	}

	return nil
}

func (s *AnnouncementAttachmentStorage) Close() error {
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
