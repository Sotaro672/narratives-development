// backend/internal/adapters/out/gcs/announcementAttachment_repository_gcs.go
package gcs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	aa "narratives/internal/domain/announcementAttachment"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// =====================================================
// GCS-based Repository & Object Storage for Attachments
// =====================================================

type AnnouncementAttachmentRepositoryGCS struct {
	Client *storage.Client
	Bucket string
}

// NewAnnouncementAttachmentRepositoryGCS creates a repository backed by GCS.
// If bucket is empty, fallback to aa.DefaultBucket.
func NewAnnouncementAttachmentRepositoryGCS(client *storage.Client, bucket string) *AnnouncementAttachmentRepositoryGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = aa.DefaultBucket
	}
	return &AnnouncementAttachmentRepositoryGCS{
		Client: client,
		Bucket: b,
	}
}

func (r *AnnouncementAttachmentRepositoryGCS) bucketName() (string, error) {
	if r.Client == nil {
		return "", errors.New("announcementAttachment: GCS client is nil")
	}
	b := strings.TrimSpace(r.Bucket)
	if b == "" {
		b = aa.DefaultBucket
	}
	if b == "" {
		return "", errors.New("announcementAttachment: bucket is empty")
	}
	return b, nil
}

// objectPath builds "announcementID/fileName".
func objectPath(announcementID, fileName string) (string, error) {
	a := strings.TrimSpace(announcementID)
	f := strings.TrimSpace(fileName)
	if a == "" || f == "" {
		return "", fmt.Errorf("invalid announcementID or fileName: %q, %q", announcementID, fileName)
	}
	f = strings.TrimLeft(f, "/")
	return a + "/" + f, nil
}

// buildAttachmentFromAttrs maps GCS object attrs -> AttachmentFile.
func buildAttachmentFromAttrs(
	announcementID string,
	bucket string,
	attrs *storage.ObjectAttrs,
) aa.AttachmentFile {
	prefix := strings.TrimSpace(announcementID) + "/"
	fileName := strings.TrimPrefix(attrs.Name, prefix)

	return aa.AttachmentFile{
		AnnouncementID: announcementID,
		FileName:       fileName,
		Bucket:         bucket,
		ObjectPath:     attrs.Name,
	}
}

// =======================
// List (page-based)
// =======================

func (r *AnnouncementAttachmentRepositoryGCS) List(
	ctx context.Context,
	filter aa.Filter,
	_ aa.Sort,
	page aa.Page,
) (aa.PageResult[aa.AttachmentFile], error) {
	b, err := r.bucketName()
	if err != nil {
		return aa.PageResult[aa.AttachmentFile]{}, err
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	var prefix string
	if filter.AnnouncementID != nil {
		if v := strings.TrimSpace(*filter.AnnouncementID); v != "" {
			prefix = v + "/"
		}
	}

	it := r.Client.Bucket(b).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	var all []aa.AttachmentFile
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return aa.PageResult[aa.AttachmentFile]{}, err
		}

		annID := ""
		if filter.AnnouncementID != nil {
			annID = strings.TrimSpace(*filter.AnnouncementID)
		}
		if annID == "" {
			parts := strings.SplitN(attrs.Name, "/", 2)
			if len(parts) != 2 || parts[0] == "" {
				continue
			}
			annID = parts[0]
		}

		all = append(all, buildAttachmentFromAttrs(annID, b, attrs))
	}

	total := len(all)
	if total == 0 {
		return aa.PageResult[aa.AttachmentFile]{}, nil
	}

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}

	items := all[offset:end]

	return aa.PageResult[aa.AttachmentFile]{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// =======================
// ListByCursor
// =======================

func (r *AnnouncementAttachmentRepositoryGCS) ListByCursor(
	ctx context.Context,
	filter aa.Filter,
	_ aa.Sort,
	cpage aa.CursorPage,
) (aa.CursorPageResult[aa.AttachmentFile], error) {
	b, err := r.bucketName()
	if err != nil {
		return aa.CursorPageResult[aa.AttachmentFile]{}, err
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var prefix string
	if filter.AnnouncementID != nil {
		if v := strings.TrimSpace(*filter.AnnouncementID); v != "" {
			prefix = v + "/"
		}
	}

	it := r.Client.Bucket(b).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	var items []aa.AttachmentFile
	for len(items) < limit {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return aa.CursorPageResult[aa.AttachmentFile]{}, err
		}

		annID := ""
		if filter.AnnouncementID != nil {
			annID = strings.TrimSpace(*filter.AnnouncementID)
		}
		if annID == "" {
			parts := strings.SplitN(attrs.Name, "/", 2)
			if len(parts) != 2 || parts[0] == "" {
				continue
			}
			annID = parts[0]
		}

		items = append(items, buildAttachmentFromAttrs(annID, b, attrs))
	}

	return aa.CursorPageResult[aa.AttachmentFile]{
		Items:      items,
		NextCursor: nil, // extend with real cursor if needed
	}, nil
}

// =======================
// Single / Batch operations
// =======================

func (r *AnnouncementAttachmentRepositoryGCS) Get(
	ctx context.Context,
	announcementID,
	fileName string,
) (aa.AttachmentFile, error) {
	b, err := r.bucketName()
	if err != nil {
		return aa.AttachmentFile{}, err
	}
	path, err := objectPath(announcementID, fileName)
	if err != nil {
		return aa.AttachmentFile{}, err
	}

	attrs, err := r.Client.Bucket(b).Object(path).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return aa.AttachmentFile{}, aa.ErrNotFound
		}
		return aa.AttachmentFile{}, err
	}

	return buildAttachmentFromAttrs(announcementID, b, attrs), nil
}

func (r *AnnouncementAttachmentRepositoryGCS) GetByAnnouncementID(
	ctx context.Context,
	announcementID string,
) ([]aa.AttachmentFile, error) {
	b, err := r.bucketName()
	if err != nil {
		return nil, err
	}

	aid := strings.TrimSpace(announcementID)
	if aid == "" {
		return nil, aa.ErrNotFound
	}

	prefix := aid + "/"
	it := r.Client.Bucket(b).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	var out []aa.AttachmentFile
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		out = append(out, buildAttachmentFromAttrs(aid, b, attrs))
	}
	return out, nil
}

func (r *AnnouncementAttachmentRepositoryGCS) Exists(
	ctx context.Context,
	announcementID,
	fileName string,
) (bool, error) {
	b, err := r.bucketName()
	if err != nil {
		return false, err
	}
	path, err := objectPath(announcementID, fileName)
	if err != nil {
		return false, nil
	}

	_, err = r.Client.Bucket(b).Object(path).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *AnnouncementAttachmentRepositoryGCS) Count(
	ctx context.Context,
	filter aa.Filter,
) (int, error) {
	b, err := r.bucketName()
	if err != nil {
		return 0, err
	}

	var prefix string
	if filter.AnnouncementID != nil {
		if v := strings.TrimSpace(*filter.AnnouncementID); v != "" {
			prefix = v + "/"
		}
	}

	it := r.Client.Bucket(b).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	total := 0
	for {
		_, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		total++
	}
	return total, nil
}

func (r *AnnouncementAttachmentRepositoryGCS) Create(
	ctx context.Context,
	f aa.AttachmentFile,
) (aa.AttachmentFile, error) {
	ok, err := r.Exists(ctx, f.AnnouncementID, f.FileName)
	if err != nil {
		return aa.AttachmentFile{}, err
	}
	if !ok {
		return aa.AttachmentFile{}, aa.ErrNotFound
	}
	return f, nil
}

func (r *AnnouncementAttachmentRepositoryGCS) Update(
	ctx context.Context,
	announcementID,
	fileName string,
	_ aa.AttachmentFilePatch,
) (aa.AttachmentFile, error) {
	return r.Get(ctx, announcementID, fileName)
}

func (r *AnnouncementAttachmentRepositoryGCS) Delete(
	ctx context.Context,
	announcementID,
	fileName string,
) error {
	b, err := r.bucketName()
	if err != nil {
		return err
	}
	path, err := objectPath(announcementID, fileName)
	if err != nil {
		return err
	}

	if err := r.Client.Bucket(b).Object(path).Delete(ctx); err != nil &&
		!errors.Is(err, storage.ErrObjectNotExist) {
		return err
	}
	return nil
}

func (r *AnnouncementAttachmentRepositoryGCS) DeleteAllByAnnouncementID(
	ctx context.Context,
	announcementID string,
) error {
	b, err := r.bucketName()
	if err != nil {
		return err
	}

	aid := strings.TrimSpace(announcementID)
	if aid == "" {
		return aa.ErrNotFound
	}

	prefix := aid + "/"
	it := r.Client.Bucket(b).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	var errs []error
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		if err := r.Client.Bucket(b).Object(attrs.Name).Delete(ctx); err != nil &&
			!errors.Is(err, storage.ErrObjectNotExist) {
			errs = append(errs, fmt.Errorf("%s: %w", attrs.Name, err))
		}
	}
	if len(errs) > 0 {
		return dbcommon.JoinErrors(errs)
	}
	return nil
}

func (r *AnnouncementAttachmentRepositoryGCS) Save(
	ctx context.Context,
	f aa.AttachmentFile,
	_ *aa.SaveOptions,
) (aa.AttachmentFile, error) {
	ok, err := r.Exists(ctx, f.AnnouncementID, f.FileName)
	if err != nil {
		return aa.AttachmentFile{}, err
	}
	if !ok {
		return aa.AttachmentFile{}, aa.ErrNotFound
	}
	return f, nil
}

// =====================================================
// Object Storage Port (GCS operations)
// =====================================================

type AnnouncementAttachmentStorageGCS struct {
	Client          *storage.Client
	Bucket          string
	SignedURLExpiry time.Duration
}

func NewAnnouncementAttachmentStorageGCS(client *storage.Client, bucket string) *AnnouncementAttachmentStorageGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = aa.DefaultBucket
	}
	return &AnnouncementAttachmentStorageGCS{
		Client:          client,
		Bucket:          b,
		SignedURLExpiry: 15 * time.Minute,
	}
}

func (s *AnnouncementAttachmentStorageGCS) effectiveBucket(b string) (string, error) {
	if s.Client == nil {
		return "", errors.New("announcementAttachment: GCS client is nil")
	}
	b = strings.TrimSpace(b)
	if b == "" {
		if s.Bucket != "" {
			b = s.Bucket
		} else {
			b = aa.DefaultBucket
		}
	}
	if b == "" {
		return "", errors.New("announcementAttachment: bucket is empty")
	}
	return b, nil
}

func (s *AnnouncementAttachmentStorageGCS) DeleteObject(ctx context.Context, bucket, objectPath string) error {
	b, err := s.effectiveBucket(bucket)
	if err != nil {
		return err
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return fmt.Errorf("invalid objectPath: %q", objectPath)
	}
	if err := s.Client.Bucket(b).Object(obj).Delete(ctx); err != nil &&
		!errors.Is(err, storage.ErrObjectNotExist) {
		return err
	}
	return nil
}

func (s *AnnouncementAttachmentStorageGCS) DeleteObjects(ctx context.Context, ops []aa.GCSDeleteOp) error {
	if len(ops) == 0 {
		return nil
	}
	var errs []error
	for _, op := range ops {
		b, err := s.effectiveBucket(op.Bucket)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := s.DeleteObject(ctx, b, op.ObjectPath); err != nil {
			errs = append(errs, fmt.Errorf("%s/%s: %w", b, op.ObjectPath, err))
		}
	}
	if len(errs) > 0 {
		return dbcommon.JoinErrors(errs)
	}
	return nil
}
