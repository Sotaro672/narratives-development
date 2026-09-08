// backend/internal/adapters/out/firebase/news_image_storage.go
package firebase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"

	gcs "cloud.google.com/go/storage"
	"github.com/google/uuid"

	applicationport "narratives/internal/application/port"
)

const (
	newsImageStorageBucketEnv                     = "FIREBASE_STORAGE_BUCKET"
	newsImageStoragePrefix                        = "news/"
	newsImageMaxFileSize                    int64 = 5 * 1024 * 1024
	firebaseStorageDownloadTokenMetadataKey       = "firebaseStorageDownloadTokens"
)

type NewsImageStorage struct {
	Client     *gcs.Client
	BucketName string
	ownsClient bool
}

func NewNewsImageStorage(client *gcs.Client, bucketName string) *NewsImageStorage {
	return &NewsImageStorage{
		Client:     client,
		BucketName: bucketName,
		ownsClient: false,
	}
}

func NewNewsImageStorageFromEnv(ctx context.Context) (*NewsImageStorage, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}

	bucketName := os.Getenv(newsImageStorageBucketEnv)
	if bucketName == "" {
		return nil, fmt.Errorf("%s is required", newsImageStorageBucketEnv)
	}

	client, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create cloud storage client: %w", err)
	}

	return &NewsImageStorage{
		Client:     client,
		BucketName: bucketName,
		ownsClient: true,
	}, nil
}

var _ applicationport.NewsImageStorage = (*NewsImageStorage)(nil)

// ============================================================
// Upload
// ============================================================

func (s *NewsImageStorage) Upload(
	ctx context.Context,
	input applicationport.NewsImageUploadInput,
) (applicationport.NewsImageUploadResult, error) {
	if err := s.validateConfiguration(ctx); err != nil {
		return applicationport.NewsImageUploadResult{}, err
	}
	if err := validateNewsImageUploadInput(input); err != nil {
		return applicationport.NewsImageUploadResult{}, err
	}

	mimeType := strings.ToLower(input.ContentType)
	extension, err := newsImageExtension(mimeType)
	if err != nil {
		return applicationport.NewsImageUploadResult{}, err
	}

	objectPath := newsImageObjectPath(input.NewsID, extension)
	downloadToken := uuid.NewString()

	uploadCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	object := s.Client.
		Bucket(s.BucketName).
		Object(objectPath).
		If(gcs.Conditions{
			DoesNotExist: true,
		})

	writer := object.NewWriter(uploadCtx)
	writer.ContentType = mimeType
	writer.Metadata = map[string]string{
		firebaseStorageDownloadTokenMetadataKey: downloadToken,
	}

	reader := io.LimitReader(input.Reader, newsImageMaxFileSize+1)
	written, copyErr := io.Copy(writer, reader)
	if copyErr != nil {
		abortNewsImageUpload(cancel, writer)
		return applicationport.NewsImageUploadResult{}, fmt.Errorf(
			"upload news image %q to bucket %q: %w",
			objectPath,
			s.BucketName,
			copyErr,
		)
	}

	if written > newsImageMaxFileSize {
		abortNewsImageUpload(cancel, writer)
		return applicationport.NewsImageUploadResult{}, fmt.Errorf(
			"news image exceeds maximum file size of %d bytes",
			newsImageMaxFileSize,
		)
	}

	if written != input.FileSize {
		abortNewsImageUpload(cancel, writer)
		return applicationport.NewsImageUploadResult{}, fmt.Errorf(
			"news image file size mismatch: expected %d bytes, received %d bytes",
			input.FileSize,
			written,
		)
	}

	if err := writer.Close(); err != nil {
		return applicationport.NewsImageUploadResult{}, fmt.Errorf(
			"close news image upload %q in bucket %q: %w",
			objectPath,
			s.BucketName,
			err,
		)
	}

	fileURL := buildFirebaseStorageDownloadURL(
		s.BucketName,
		objectPath,
		downloadToken,
	)

	return applicationport.NewsImageUploadResult{
		FileURL:    fileURL,
		ObjectPath: objectPath,
		FileName:   input.FileName,
		MimeType:   mimeType,
		FileSize:   written,
	}, nil
}

// ============================================================
// Delete
// ============================================================

func (s *NewsImageStorage) Delete(ctx context.Context, objectPath string) error {
	if err := s.validateConfiguration(ctx); err != nil {
		return err
	}
	if err := validateNewsImageObjectPath(objectPath); err != nil {
		return err
	}

	err := s.Client.
		Bucket(s.BucketName).
		Object(objectPath).
		Delete(ctx)

	if err == nil || errors.Is(err, gcs.ErrObjectNotExist) {
		return nil
	}

	return fmt.Errorf(
		"delete news image object %q from bucket %q: %w",
		objectPath,
		s.BucketName,
		err,
	)
}

// ============================================================
// Validation
// ============================================================

func (s *NewsImageStorage) validateConfiguration(ctx context.Context) error {
	if s == nil {
		return errors.New("news image storage is nil")
	}
	if s.Client == nil {
		return errors.New("cloud storage client is nil")
	}
	if ctx == nil {
		return errors.New("context is nil")
	}
	if s.BucketName == "" {
		return errors.New("cloud storage bucket name is empty")
	}
	return nil
}

func validateNewsImageUploadInput(input applicationport.NewsImageUploadInput) error {
	if err := validateNewsImageID(input.NewsID); err != nil {
		return err
	}
	if input.FileName == "" {
		return errors.New("news image file name is required")
	}
	if !utf8.ValidString(input.FileName) {
		return errors.New("news image file name must be valid UTF-8")
	}
	if strings.ContainsRune(input.FileName, '\x00') {
		return errors.New("news image file name contains a null character")
	}
	if strings.ContainsAny(input.FileName, "\r\n") {
		return errors.New("news image file name contains an invalid control character")
	}
	if input.ContentType == "" {
		return errors.New("news image content type is required")
	}
	if _, err := newsImageExtension(strings.ToLower(input.ContentType)); err != nil {
		return err
	}
	if input.FileSize <= 0 {
		return errors.New("news image file size must be greater than zero")
	}
	if input.FileSize > newsImageMaxFileSize {
		return fmt.Errorf(
			"news image exceeds maximum file size of %d bytes",
			newsImageMaxFileSize,
		)
	}
	if input.Reader == nil {
		return errors.New("news image reader is required")
	}
	return nil
}

func validateNewsImageID(newsID string) error {
	if newsID == "" {
		return errors.New("newsID is required")
	}
	if !utf8.ValidString(newsID) {
		return errors.New("newsID must be valid UTF-8")
	}
	if strings.ContainsRune(newsID, '\x00') {
		return errors.New("newsID contains a null character")
	}
	if strings.ContainsAny(newsID, "\r\n") {
		return errors.New("newsID contains an invalid control character")
	}
	if strings.ContainsAny(newsID, "/\\:") {
		return errors.New("newsID contains an invalid path separator")
	}
	if newsID == "." || newsID == ".." {
		return errors.New("newsID is invalid")
	}
	return nil
}

func validateNewsImageObjectPath(objectPath string) error {
	if objectPath == "" {
		return errors.New("news image object path is required")
	}
	if !utf8.ValidString(objectPath) {
		return errors.New("news image object path must be valid UTF-8")
	}
	if strings.ContainsRune(objectPath, '\x00') {
		return errors.New("news image object path contains a null character")
	}
	if strings.ContainsAny(objectPath, "\r\n") {
		return errors.New("news image object path contains an invalid control character")
	}
	if !strings.HasPrefix(objectPath, newsImageStoragePrefix) {
		return errors.New("news image object path is outside the news storage prefix")
	}
	if !strings.Contains(objectPath, "/image/") {
		return errors.New("news image object path is invalid")
	}
	return nil
}

// ============================================================
// Object path / MIME
// ============================================================

func newsImageObjectPath(newsID string, extension string) string {
	return newsImageStoragePrefix +
		newsID +
		"/image/" +
		uuid.NewString() +
		extension
}

func newsImageExtension(mimeType string) (string, error) {
	switch mimeType {
	case "image/jpeg":
		return ".jpg", nil
	case "image/png":
		return ".png", nil
	case "image/webp":
		return ".webp", nil
	default:
		return "", fmt.Errorf(
			"unsupported news image content type: %s",
			mimeType,
		)
	}
}

// ============================================================
// Firebase download URL
// ============================================================

func buildFirebaseStorageDownloadURL(
	bucketName string,
	objectPath string,
	downloadToken string,
) string {
	return "https://firebasestorage.googleapis.com/v0/b/" +
		url.PathEscape(bucketName) +
		"/o/" +
		url.PathEscape(objectPath) +
		"?alt=media&token=" +
		url.QueryEscape(downloadToken)
}

// ============================================================
// Upload abort
// ============================================================

func abortNewsImageUpload(
	cancel context.CancelFunc,
	writer *gcs.Writer,
) {
	if cancel != nil {
		cancel()
	}
	if writer != nil {
		_ = writer.Close()
	}
}

// ============================================================
// Close
// ============================================================

func (s *NewsImageStorage) Close() error {
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
