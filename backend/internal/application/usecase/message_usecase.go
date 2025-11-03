package usecase

import (
    "context"
    "strings"
    "time"

    msgdom "narratives/internal/domain/message"
    imgdom "narratives/internal/domain/messageImage"
)

// MessageUsecase wires domain entities Message and MessageImage via their repository ports.
type MessageUsecase struct {
    msgRepo  msgdom.Repository
    imgRepo  imgdom.RepositoryPort       // optional: for storing image metadata
    objStore imgdom.ObjectStoragePort    // optional: for deleting GCS objects

    now   func() time.Time
    idGen func() string
}

func NewMessageUsecase(
    msgRepo msgdom.Repository,
    imgRepo imgdom.RepositoryPort,
    objStore imgdom.ObjectStoragePort,
) *MessageUsecase {
    return &MessageUsecase{
        msgRepo:  msgRepo,
        imgRepo:  imgRepo,
        objStore: objStore,
        now:      time.Now,
        idGen:    defaultIDGen,
    }
}

func (u *MessageUsecase) WithNow(now func() time.Time) *MessageUsecase {
    u.now = now
    return u
}

func (u *MessageUsecase) WithIDGen(idGen func() string) *MessageUsecase {
    u.idGen = idGen
    return u
}

// =======================
// Create Draft
// =======================

type NewImageInput struct {
    // If ObjectPath is empty, it will be built as messages/{messageID}/{fileName}
    FileName   string
    ObjectPath string
    FileURL    string
    FileSize   int64
    MimeType   string
    Width      *int
    Height     *int
    UploadedAt time.Time
}

type CreateDraftInput struct {
    // If ID is empty, a new one is generated and used for both message and image paths.
    ID         string
    SenderID   string
    ReceiverID string
    Content    string
    Images     []NewImageInput
}

type CreateDraftOutput struct {
    Message msgdom.Message
    Images  []imgdom.ImageFile // stored metadata if imgRepo is provided
}

// CreateDraftMessage builds a Message with draft status and optional image references,
// persists the message, and (optionally) persists image metadata for later use.
func (u *MessageUsecase) CreateDraftMessage(ctx context.Context, in CreateDraftInput) (CreateDraftOutput, error) {
    now := u.now().UTC()

    // Prepare ID
    id := strings.TrimSpace(in.ID)
    if id == "" {
        id = u.idGen()
    }

    // Build image refs (for Firestore/Message entity)
    imgRefs := make([]msgdom.ImageRef, 0, len(in.Images))
    for _, im := range in.Images {
        objPath := strings.TrimLeft(strings.TrimSpace(im.ObjectPath), "/")
        if objPath == "" {
            // follow convention: messages/{messageId}/{fileName}
            p, err := imgdom.BuildObjectPath(id, im.FileName)
            if err != nil {
                return CreateDraftOutput{}, err
            }
            objPath = p
        }
        ref := msgdom.ImageRef{
            ObjectPath: objPath,
            URL:        strings.TrimSpace(im.FileURL),
            FileName:   strings.TrimSpace(im.FileName),
            FileSize:   im.FileSize,
            MimeType:   strings.TrimSpace(im.MimeType),
            Width:      im.Width,
            Height:     im.Height,
            UploadedAt: im.UploadedAt.UTC(),
        }
        imgRefs = append(imgRefs, ref)
    }

    // Build domain Message
    m, err := msgdom.CreateDraftMessage(
        id,
        in.SenderID,
        in.ReceiverID,
        in.Content,
        imgRefs,
        now,
    )
    if err != nil {
        return CreateDraftOutput{}, err
    }

    // Persist Message
    savedMsg, err := u.msgRepo.CreateDraft(ctx, m)
    if err != nil {
        return CreateDraftOutput{}, err
    }

    // Optionally persist image metadata
    var savedImgs []imgdom.ImageFile
    if u.imgRepo != nil && len(in.Images) > 0 {
        meta := make([]imgdom.ImageFile, 0, len(in.Images))
        for _, im := range in.Images {
            // Compute ObjectPath consistently with Message refs
            objPath := strings.TrimLeft(strings.TrimSpace(im.ObjectPath), "/")
            if objPath == "" {
                p, err := imgdom.BuildObjectPath(savedMsg.ID, im.FileName)
                if err != nil {
                    return CreateDraftOutput{}, err
                }
                objPath = p
            }
            file, err := imgdom.NewImageFileWithBucket(
                imgdom.DefaultBucket,
                savedMsg.ID,
                im.FileName,
                im.FileURL,
                im.FileSize,
                im.MimeType,
                im.Width,
                im.Height,
                now,
                nil,
                nil,
            )
            if err != nil {
                return CreateDraftOutput{}, err
            }
            // enforce computed ObjectPath (NewImageFileWithBucket also sets it by convention)
            file.ObjectPath = objPath
            meta = append(meta, file)
        }
        // Replace all metadata for the message (idempotent for initial insert)
        saved, err := u.imgRepo.ReplaceAll(ctx, savedMsg.ID, meta)
        if err != nil {
            return CreateDraftOutput{}, err
        }
        savedImgs = saved
    }

    return CreateDraftOutput{
        Message: savedMsg,
        Images:  savedImgs,
    }, nil
}

// =======================
// Transitions
// =======================

func (u *MessageUsecase) SendMessage(ctx context.Context, messageID string) (msgdom.Message, error) {
    return u.msgRepo.Send(ctx, strings.TrimSpace(messageID), u.now())
}

func (u *MessageUsecase) CancelMessage(ctx context.Context, messageID string) (msgdom.Message, error) {
    return u.msgRepo.Cancel(ctx, strings.TrimSpace(messageID), u.now())
}

func (u *MessageUsecase) MarkDelivered(ctx context.Context, messageID string) (msgdom.Message, error) {
    return u.msgRepo.MarkDelivered(ctx, strings.TrimSpace(messageID), u.now())
}

func (u *MessageUsecase) MarkRead(ctx context.Context, messageID string) (msgdom.Message, error) {
    return u.msgRepo.MarkRead(ctx, strings.TrimSpace(messageID), u.now())
}

// =======================
// Delete with cascade (Message -> MessageImage in GCS)
// =======================

// DeleteMessageCascade deletes the message and also removes related message images:
// - delete GCS objects via ObjectStoragePort (if provided)
// - delete image metadata via RepositoryPort (if provided)
// - finally delete the message via msgRepo.Delete
// If any step fails, it aborts and returns the error.
func (u *MessageUsecase) DeleteMessageCascade(ctx context.Context, messageID string) error {
    messageID = strings.TrimSpace(messageID)
    if messageID == "" {
        return msgdom.ErrInvalid
    }

    // Load the message to obtain image refs when available.
    m, err := u.msgRepo.GetByID(ctx, messageID)
    if err != nil && !isNotFound(err) {
        return err
    }

    // Build delete ops from Message refs if present, otherwise from metadata.
    var ops []imgdom.GCSDeleteOp
    if len(m.Images) > 0 {
        ops = imgdom.BuildGCSDeleteOpsFromMessage(m)
    } else if u.imgRepo != nil {
        if imgs, e := u.imgRepo.ListByMessageID(ctx, messageID); e == nil {
            ops = imgdom.BuildGCSDeleteOps(imgs)
        } else if !isNotFound(e) {
            return e
        }
    }

    // Delete objects on GCS (best effort happens inside adapter; still return error if any)
    if u.objStore != nil && len(ops) > 0 {
        if err := u.objStore.DeleteObjects(ctx, ops); err != nil {
            return err
        }
    }

    // Delete metadata
    if u.imgRepo != nil {
        if err := u.imgRepo.DeleteAll(ctx, messageID); err != nil && !isNotFound(err) {
            return err
        }
    }

    // Delete message (source of truth)
    if err := u.msgRepo.Delete(ctx, messageID); err != nil {
        return err
    }
    return nil
}

// =======================
// Utilities
// =======================

func isNotFound(err error) bool {
    return err == msgdom.ErrNotFound || err == imgdom.ErrNotFound
}

// very simple ID generator placeholder; replace with ULID/UUID in infra
func defaultIDGen() string {
    return strings.ReplaceAll(time.Now().UTC().Format("20060102T150405.000000000Z07:00"), ":", "")
}