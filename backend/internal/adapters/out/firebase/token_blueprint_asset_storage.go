// backend/internal/adapters/out/firebase/token_blueprint_asset_storage.go
package firebase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	usecase "narratives/internal/application/usecase"
)

const tokenBlueprintAssetStorageBucketEnv = "FIREBASE_STORAGE_BUCKET"

type TokenBlueprintAssetStorage struct {
	Client     *gcs.Client
	BucketName string
	ownsClient bool
}

func NewTokenBlueprintAssetStorage(
	client *gcs.Client,
	bucketName string,
) *TokenBlueprintAssetStorage {
	return &TokenBlueprintAssetStorage{
		Client:     client,
		BucketName: strings.TrimSpace(bucketName),
		ownsClient: false,
	}
}

func NewTokenBlueprintAssetStorageFromEnv(
	ctx context.Context,
) (*TokenBlueprintAssetStorage, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}

	bucketName := strings.TrimSpace(
		os.Getenv(
			tokenBlueprintAssetStorageBucketEnv,
		),
	)

	if bucketName == "" {
		return nil, fmt.Errorf(
			"%s is required",
			tokenBlueprintAssetStorageBucketEnv,
		)
	}

	client, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"create cloud storage client: %w",
			err,
		)
	}

	return &TokenBlueprintAssetStorage{
		Client:     client,
		BucketName: bucketName,
		ownsClient: true,
	}, nil
}

var _ usecase.TokenBlueprintAssetStorage = (*TokenBlueprintAssetStorage)(nil)

func (s *TokenBlueprintAssetStorage) DeleteAll(
	ctx context.Context,
	companyID string,
	tokenBlueprintID string,
) error {
	bucketName, prefix, err := s.resolvePrefix(
		ctx,
		companyID,
		tokenBlueprintID,
	)
	if err != nil {
		return err
	}

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
				"list token blueprint storage objects under prefix %q: %w",
				prefix,
				err,
			)
		}

		if attrs == nil ||
			attrs.Name == "" {
			continue
		}

		err = bucket.
			Object(
				attrs.Name,
			).
			Delete(ctx)

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
			"delete token blueprint storage object %q: %w",
			attrs.Name,
			err,
		)
	}

	return nil
}

func (s *TokenBlueprintAssetStorage) Close() error {
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

func (s *TokenBlueprintAssetStorage) resolvePrefix(
	ctx context.Context,
	companyID string,
	tokenBlueprintID string,
) (
	string,
	string,
	error,
) {
	if s == nil {
		return "", "", errors.New(
			"token blueprint asset storage is nil",
		)
	}

	if s.Client == nil {
		return "", "", errors.New(
			"cloud storage client is nil",
		)
	}

	if ctx == nil {
		return "", "", errors.New(
			"context is nil",
		)
	}

	bucketName := strings.TrimSpace(
		s.BucketName,
	)
	if bucketName == "" {
		return "", "", errors.New(
			"cloud storage bucket name is empty",
		)
	}

	companyID = strings.TrimSpace(
		companyID,
	)
	if companyID == "" {
		return "", "", errors.New(
			"companyID is required",
		)
	}

	if strings.ContainsAny(
		companyID,
		"/\\",
	) {
		return "", "", errors.New(
			"companyID contains an invalid path separator",
		)
	}

	tokenBlueprintID = strings.TrimSpace(
		tokenBlueprintID,
	)
	if tokenBlueprintID == "" {
		return "", "", errors.New(
			"tokenBlueprintID is required",
		)
	}

	if strings.ContainsAny(
		tokenBlueprintID,
		"/\\",
	) {
		return "", "", errors.New(
			"tokenBlueprintID contains an invalid path separator",
		)
	}

	prefix := strings.Join(
		[]string{
			"token-blueprints",
			companyID,
			tokenBlueprintID,
			"",
		},
		"/",
	)

	return bucketName, prefix, nil
}
