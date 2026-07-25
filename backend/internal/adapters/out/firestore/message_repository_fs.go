package firestore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	fs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	messagedom "narratives/internal/domain/message"
)

const messageCollection = "messages"

type MessageRepository struct {
	client     *fs.Client
	collection string
}

// Compile-time check.
var _ messagedom.Repository = (*MessageRepository)(nil)

func NewMessageRepository(client *fs.Client) *MessageRepository {
	return NewMessageRepositoryWithCollection(client, messageCollection)
}

func NewMessageRepositoryWithCollection(client *fs.Client, collection string) *MessageRepository {
	if collection == "" {
		collection = messageCollection
	}

	return &MessageRepository{
		client:     client,
		collection: collection,
	}
}

func (r *MessageRepository) Create(ctx context.Context, m messagedom.Message) error {
	if err := r.validateReady(); err != nil {
		return err
	}
	if err := validateMessage(m); err != nil {
		return err
	}

	_, err := r.collectionRef().Doc(m.ID).Create(ctx, toMessageDoc(m))
	return mapFirestoreError(err)
}

func (r *MessageRepository) FindByID(ctx context.Context, id string) (messagedom.Message, error) {
	if err := r.validateReady(); err != nil {
		return messagedom.Message{}, err
	}
	if id == "" {
		return messagedom.Message{}, messagedom.ErrInvalidID
	}

	snapshot, err := r.collectionRef().Doc(id).Get(ctx)
	if err != nil {
		return messagedom.Message{}, mapFirestoreError(err)
	}

	return messageFromSnapshot(snapshot)
}

func (r *MessageRepository) Update(ctx context.Context, m messagedom.Message) error {
	if err := r.validateReady(); err != nil {
		return err
	}
	if err := validateMessage(m); err != nil {
		return err
	}

	doc := toMessageDoc(m)
	_, err := r.collectionRef().Doc(m.ID).Update(ctx, messageDocUpdates(doc))
	return mapFirestoreError(err)
}

func (r *MessageRepository) Delete(ctx context.Context, id string) error {
	if err := r.validateReady(); err != nil {
		return err
	}
	if id == "" {
		return messagedom.ErrInvalidID
	}

	_, err := r.collectionRef().Doc(id).Delete(ctx)
	return mapFirestoreError(err)
}

func (r *MessageRepository) ListThread(
	ctx context.Context,
	memberID string,
	peerMemberID string,
	filter messagedom.ListFilter,
) ([]messagedom.Message, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}
	if memberID == "" {
		return nil, messagedom.ErrInvalidSenderMemberID
	}
	if peerMemberID == "" {
		return nil, messagedom.ErrInvalidReceiverMemberID
	}
	if memberID == peerMemberID {
		return nil, messagedom.ErrSelfMessageNotAllowed
	}

	sent, err := r.ListSent(ctx, memberID, listFilterWithoutLimit(filter))
	if err != nil {
		return nil, err
	}

	received, err := r.ListReceived(ctx, memberID, listFilterWithoutLimit(filter))
	if err != nil {
		return nil, err
	}

	messages := make([]messagedom.Message, 0, len(sent)+len(received))

	for _, message := range sent {
		if message.ReceiverMemberID == peerMemberID {
			messages = append(messages, message)
		}
	}

	for _, message := range received {
		if message.SenderMemberID == peerMemberID {
			messages = append(messages, message)
		}
	}

	messages = dedupeMessages(messages)
	sortMessagesDesc(messages)

	limit := normalizeLimit(filter.Limit)
	if len(messages) > limit {
		messages = messages[:limit]
	}

	return messages, nil
}

func (r *MessageRepository) ListReceived(
	ctx context.Context,
	receiverMemberID string,
	filter messagedom.ListFilter,
) ([]messagedom.Message, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}
	if receiverMemberID == "" {
		return nil, messagedom.ErrInvalidReceiverMemberID
	}

	query := r.collectionRef().Where("receiverMemberId", "==", receiverMemberID)
	return r.queryMessages(ctx, query, filter)
}

func (r *MessageRepository) ListSent(
	ctx context.Context,
	senderMemberID string,
	filter messagedom.ListFilter,
) ([]messagedom.Message, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}
	if senderMemberID == "" {
		return nil, messagedom.ErrInvalidSenderMemberID
	}

	query := r.collectionRef().Where("senderMemberId", "==", senderMemberID)
	return r.queryMessages(ctx, query, filter)
}

func (r *MessageRepository) MarkAsRead(ctx context.Context, id string, readAt time.Time) error {
	m, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := m.MarkAsRead(readAt); err != nil {
		return err
	}

	return r.Update(ctx, m)
}

func (r *MessageRepository) collectionRef() *fs.CollectionRef {
	return r.client.Collection(r.collection)
}

func (r *MessageRepository) validateReady() error {
	if r == nil || r.client == nil {
		return errors.New("message_repository_fs: firestore client is nil")
	}
	if r.collection == "" {
		return errors.New("message_repository_fs: collection is empty")
	}

	return nil
}

func (r *MessageRepository) queryMessages(
	ctx context.Context,
	query fs.Query,
	filter messagedom.ListFilter,
) ([]messagedom.Message, error) {
	limit := normalizeLimit(filter.Limit)

	// First try the efficient query. This needs a composite index:
	//   equality field ASC + createdAt DESC
	orderedQuery := query.OrderBy("createdAt", fs.Desc).Limit(limit)
	if filter.BeforeCreatedAt != nil {
		before := filter.BeforeCreatedAt.UTC()
		orderedQuery = orderedQuery.Where("createdAt", "<", before)
	}

	messages, err := r.collectMessages(ctx, orderedQuery)
	if err == nil {
		return messages, nil
	}

	// If the index is not ready yet, fall back to a single-field query and sort
	// in memory. This keeps the endpoint alive while Firestore indexes are being
	// created, at the cost of reading more documents.
	if !isMissingFirestoreIndexError(err) {
		return nil, err
	}

	fallbackMessages, fallbackErr := r.collectMessages(ctx, query)
	if fallbackErr != nil {
		return nil, fallbackErr
	}

	fallbackMessages = filterMessagesByCreatedAt(fallbackMessages, filter.BeforeCreatedAt)
	sortMessagesDesc(fallbackMessages)

	if len(fallbackMessages) > limit {
		fallbackMessages = fallbackMessages[:limit]
	}

	return fallbackMessages, nil
}

func (r *MessageRepository) collectMessages(
	ctx context.Context,
	query fs.Query,
) ([]messagedom.Message, error) {
	iter := query.Documents(ctx)
	defer iter.Stop()

	messages := make([]messagedom.Message, 0)

	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, mapFirestoreError(err)
		}

		message, err := messageFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	return messages, nil
}

type messageDoc struct {
	ID               string                      `firestore:"id,omitempty"`
	CompanyID        string                      `firestore:"companyId"`
	SenderMemberID   string                      `firestore:"senderMemberId"`
	ReceiverMemberID string                      `firestore:"receiverMemberId"`
	Body             string                      `firestore:"body,omitempty"`
	Images           []messageImageAttachmentDoc `firestore:"images,omitempty"`
	IsRead           bool                        `firestore:"isRead"`
	ReadAt           *time.Time                  `firestore:"readAt,omitempty"`
	CreatedAt        time.Time                   `firestore:"createdAt"`
	UpdatedAt        time.Time                   `firestore:"updatedAt"`
}

type messageImageAttachmentDoc struct {
	StoragePath string    `firestore:"storagePath"`
	DownloadURL *string   `firestore:"downloadUrl,omitempty"`
	ContentType string    `firestore:"contentType"`
	SizeBytes   int64     `firestore:"sizeBytes"`
	Width       *int64    `firestore:"width,omitempty"`
	Height      *int64    `firestore:"height,omitempty"`
	UploadedAt  time.Time `firestore:"uploadedAt"`
}

func toMessageDoc(m messagedom.Message) messageDoc {
	images := make([]messageImageAttachmentDoc, len(m.Images))
	for i, img := range m.Images {
		images[i] = messageImageAttachmentDoc{
			StoragePath: img.StoragePath,
			DownloadURL: img.DownloadURL,
			ContentType: img.ContentType,
			SizeBytes:   img.SizeBytes,
			Width:       img.Width,
			Height:      img.Height,
			UploadedAt:  img.UploadedAt.UTC(),
		}
	}

	return messageDoc{
		ID:               m.ID,
		CompanyID:        m.CompanyID,
		SenderMemberID:   m.SenderMemberID,
		ReceiverMemberID: m.ReceiverMemberID,
		Body:             m.Body,
		Images:           images,
		IsRead:           m.IsRead,
		ReadAt:           utcTimePtr(m.ReadAt),
		CreatedAt:        m.CreatedAt.UTC(),
		UpdatedAt:        m.UpdatedAt.UTC(),
	}
}

func messageFromSnapshot(snapshot *fs.DocumentSnapshot) (messagedom.Message, error) {
	var doc messageDoc
	if err := snapshot.DataTo(&doc); err != nil {
		return messagedom.Message{}, err
	}

	if doc.ID == "" {
		doc.ID = snapshot.Ref.ID
	}

	return doc.toDomain()
}

func (d messageDoc) toDomain() (messagedom.Message, error) {
	images := make([]messagedom.MessageImageAttachment, len(d.Images))
	for i, img := range d.Images {
		images[i] = messagedom.MessageImageAttachment{
			StoragePath: img.StoragePath,
			DownloadURL: img.DownloadURL,
			ContentType: img.ContentType,
			SizeBytes:   img.SizeBytes,
			Width:       img.Width,
			Height:      img.Height,
			UploadedAt:  img.UploadedAt.UTC(),
		}
	}

	return messagedom.New(
		d.ID,
		d.CompanyID,
		d.SenderMemberID,
		d.ReceiverMemberID,
		d.Body,
		images,
		d.IsRead,
		utcTimePtr(d.ReadAt),
		d.CreatedAt.UTC(),
		d.UpdatedAt.UTC(),
	)
}

func messageDocUpdates(doc messageDoc) []fs.Update {
	var readAt any = fs.Delete
	if doc.ReadAt != nil {
		readAt = doc.ReadAt.UTC()
	}

	return []fs.Update{
		{Path: "id", Value: doc.ID},
		{Path: "companyId", Value: doc.CompanyID},
		{Path: "senderMemberId", Value: doc.SenderMemberID},
		{Path: "receiverMemberId", Value: doc.ReceiverMemberID},
		{Path: "body", Value: doc.Body},
		{Path: "images", Value: doc.Images},
		{Path: "isRead", Value: doc.IsRead},
		{Path: "readAt", Value: readAt},
		{Path: "createdAt", Value: doc.CreatedAt.UTC()},
		{Path: "updatedAt", Value: doc.UpdatedAt.UTC()},
	}
}

func validateMessage(m messagedom.Message) error {
	_, err := messagedom.New(
		m.ID,
		m.CompanyID,
		m.SenderMemberID,
		m.ReceiverMemberID,
		m.Body,
		m.Images,
		m.IsRead,
		m.ReadAt,
		m.CreatedAt,
		m.UpdatedAt,
	)

	return err
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return messagedom.DefaultListLimit
	}
	if limit > messagedom.MaxListLimit {
		return messagedom.MaxListLimit
	}

	return limit
}

func listFilterWithoutLimit(filter messagedom.ListFilter) messagedom.ListFilter {
	filter.Limit = messagedom.MaxListLimit
	return filter
}

func filterMessagesByCreatedAt(
	messages []messagedom.Message,
	before *time.Time,
) []messagedom.Message {
	if before == nil {
		return messages
	}

	beforeUTC := before.UTC()
	out := messages[:0]

	for _, message := range messages {
		if message.CreatedAt.Before(beforeUTC) {
			out = append(out, message)
		}
	}

	return out
}

func dedupeMessages(messages []messagedom.Message) []messagedom.Message {
	seen := make(map[string]struct{}, len(messages))
	result := messages[:0]

	for _, message := range messages {
		key := message.ID
		if key == "" {
			key = message.CompanyID + ":" +
				message.SenderMemberID + ":" +
				message.ReceiverMemberID + ":" +
				message.CreatedAt.UTC().Format(time.RFC3339Nano)
		}

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		result = append(result, message)
	}

	return result
}

func sortMessagesDesc(messages []messagedom.Message) {
	sort.Slice(messages, func(i, j int) bool {
		left := messages[i].CreatedAt
		right := messages[j].CreatedAt

		if left.Equal(right) {
			return messages[i].ID > messages[j].ID
		}

		return left.After(right)
	})
}

func utcTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	utc := t.UTC()
	return &utc
}

func isMissingFirestoreIndexError(err error) bool {
	if err == nil {
		return false
	}

	if status.Code(err) != codes.FailedPrecondition {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "requires an index") ||
		strings.Contains(msg, "create it here") ||
		strings.Contains(msg, "failedprecondition")
}

func mapFirestoreError(err error) error {
	if err == nil {
		return nil
	}

	switch status.Code(err) {
	case codes.NotFound:
		return messagedom.ErrNotFound
	case codes.AlreadyExists:
		return messagedom.ErrAlreadyExists
	default:
		return err
	}
}
