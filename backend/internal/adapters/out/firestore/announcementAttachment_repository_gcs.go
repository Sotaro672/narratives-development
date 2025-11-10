package db

import (
    "context"
    "database/sql" // add: used by AnnouncementAttachmentRepositoryPG
    "errors"
    "fmt"
    "strings"
    "time"

    "cloud.google.com/go/storage"
    dbcommon "narratives/internal/adapters/out/db/common"
    aa "narratives/internal/domain/announcementAttachment"
)

// PostgreSQL implementation of announcementAttachment.Repository.
type AnnouncementAttachmentRepositoryPG struct {
    db *sql.DB
}

func NewAnnouncementAttachmentRepositoryPG(db *sql.DB) *AnnouncementAttachmentRepositoryPG {
    return &AnnouncementAttachmentRepositoryPG{db: db}
}

// List returns paginated attachment files.
// TODO: Implement with proper SQL SELECT + WHERE/ORDER/LIMIT/OFFSET.
func (r *AnnouncementAttachmentRepositoryPG) List(ctx context.Context, filter aa.Filter, sort aa.Sort, page aa.Page) (aa.PageResult[aa.AttachmentFile], error) {
    return aa.PageResult[aa.AttachmentFile]{}, errors.New("List: not implemented")
}

// ListByCursor returns cursor-paginated attachment files.
// TODO: Implement with proper SQL and cursor strategy.
func (r *AnnouncementAttachmentRepositoryPG) ListByCursor(ctx context.Context, filter aa.Filter, sort aa.Sort, cpage aa.CursorPage) (aa.CursorPageResult[aa.AttachmentFile], error) {
    return aa.CursorPageResult[aa.AttachmentFile]{}, errors.New("ListByCursor: not implemented")
}

// Get fetches a single attachment by composite key.
// TODO: Implement with SELECT ... WHERE announcement_id = $1 AND file_name = $2.
func (r *AnnouncementAttachmentRepositoryPG) Get(ctx context.Context, announcementID, fileName string) (aa.AttachmentFile, error) {
    return aa.AttachmentFile{}, aa.ErrNotFound
}

// GetByAnnouncementID fetches all attachments for an announcement.
// TODO: Implement with SELECT ... WHERE announcement_id = $1.
func (r *AnnouncementAttachmentRepositoryPG) GetByAnnouncementID(ctx context.Context, announcementID string) ([]aa.AttachmentFile, error) {
    return nil, errors.New("GetByAnnouncementID: not implemented")
}

// Exists checks if an attachment exists by composite key.
// TODO: Implement with SELECT 1 FROM ... LIMIT 1.
func (r *AnnouncementAttachmentRepositoryPG) Exists(ctx context.Context, announcementID, fileName string) (bool, error) {
    return false, errors.New("Exists: not implemented")
}

// Count counts attachments matching a filter.
// TODO: Implement with SELECT COUNT(*) and same WHERE builder used in List.
func (r *AnnouncementAttachmentRepositoryPG) Count(ctx context.Context, filter aa.Filter) (int, error) {
    return 0, errors.New("Count: not implemented")
}

// Create inserts a new attachment file.
// TODO: Implement with INSERT and return the created record.
func (r *AnnouncementAttachmentRepositoryPG) Create(ctx context.Context, f aa.AttachmentFile) (aa.AttachmentFile, error) {
    return aa.AttachmentFile{}, errors.New("Create: not implemented")
}

// Update applies a partial update to an attachment file.
// TODO: Implement dynamic UPDATE based on non-nil patch fields.
func (r *AnnouncementAttachmentRepositoryPG) Update(ctx context.Context, announcementID, fileName string, patch aa.AttachmentFilePatch) (aa.AttachmentFile, error) {
    return aa.AttachmentFile{}, errors.New("Update: not implemented")
}

// Delete removes an attachment by composite key.
// TODO: Implement with DELETE ... WHERE announcement_id = $1 AND file_name = $2.
func (r *AnnouncementAttachmentRepositoryPG) Delete(ctx context.Context, announcementID, fileName string) error {
    return errors.New("Delete: not implemented")
}

// DeleteAllByAnnouncementID removes all attachments for an announcement.
// TODO: Implement with DELETE ... WHERE announcement_id = $1.
func (r *AnnouncementAttachmentRepositoryPG) DeleteAllByAnnouncementID(ctx context.Context, announcementID string) error {
    return errors.New("DeleteAllByAnnouncementID: not implemented")
}

// Save performs an upsert (insert or update).
// TODO: Implement with INSERT ... ON CONFLICT (announcement_id, file_name) DO UPDATE ...
func (r *AnnouncementAttachmentRepositoryPG) Save(ctx context.Context, f aa.AttachmentFile, _ *aa.SaveOptions) (aa.AttachmentFile, error) {
    return aa.AttachmentFile{}, errors.New("Save: not implemented")
}

// AnnouncementAttachmentStorageGCS implements announcementAttachment.ObjectStoragePort using Google Cloud Storage.
type AnnouncementAttachmentStorageGCS struct {
    Client          *storage.Client
    Bucket          string
    SignedURLExpiry time.Duration
}

// NewAnnouncementAttachmentStorageGCS creates a storage adapter with the provided client.
// If bucket is empty, it falls back to aa.DefaultBucket ("narratives_development_announcement_attachment").
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

// DeleteObject deletes a single GCS object.
// bucket or objectPath can be empty/relative; bucket falls back to adapter's default, objectPath gets trimmed.
func (s *AnnouncementAttachmentStorageGCS) DeleteObject(ctx context.Context, bucket, objectPath string) error {
    if s.Client == nil {
        return errors.New("AnnouncementAttachmentStorageGCS: nil storage client")
    }
    b := strings.TrimSpace(bucket)
    if b == "" {
        // prefer adapter's default; if empty, use domain default
        if s.Bucket != "" {
            b = s.Bucket
        } else {
            b = aa.DefaultBucket
        }
    }
    obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
    if b == "" || obj == "" {
        return fmt.Errorf("invalid bucket/objectPath: bucket=%q, objectPath=%q", b, objectPath)
    }
    err := s.Client.Bucket(b).Object(obj).Delete(ctx)
    if err != nil {
        if errors.Is(err, storage.ErrObjectNotExist) {
            // Treat as success (idempotent delete)
            return nil
        }
        return err
    }
    return nil
}

// DeleteObjects deletes multiple GCS objects best-effort.
// It continues on errors and returns a combined error if any failures occurred.
func (s *AnnouncementAttachmentStorageGCS) DeleteObjects(ctx context.Context, ops []aa.GCSDeleteOp) error {
    if len(ops) == 0 {
        return nil
    }
    var errs []error
    for _, op := range ops {
        b := strings.TrimSpace(op.Bucket)
        if b == "" {
            if s.Bucket != "" {
                b = s.Bucket
            } else {
                b = aa.DefaultBucket
            }
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