// backend/internal/application/usecase/message_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	memberdom "narratives/internal/domain/member"
	messagedom "narratives/internal/domain/message"
)

type MessageUsecase struct {
	msgRepo      MessageRepo
	memberRepo   MessageMemberRepo
	imageStorage MessageImageStorageService
	idGen        MessageIDGenerator

	now func() time.Time
}

func NewMessageUsecase(
	msgRepo MessageRepo,
	memberRepo MessageMemberRepo,
	imageStorage MessageImageStorageService,
	idGen MessageIDGenerator,
	now func() time.Time,
) *MessageUsecase {
	if now == nil {
		now = time.Now
	}

	return &MessageUsecase{
		msgRepo:      msgRepo,
		memberRepo:   memberRepo,
		imageStorage: imageStorage,
		idGen:        idGen,
		now:          now,
	}
}

type MessageRepo interface {
	Create(ctx context.Context, message messagedom.Message) error
	FindByID(ctx context.Context, id string) (messagedom.Message, error)
	Update(ctx context.Context, message messagedom.Message) error
	Delete(ctx context.Context, id string) error

	ListThread(ctx context.Context, memberID, peerMemberID string, filter messagedom.ListFilter) ([]messagedom.Message, error)
	ListReceived(ctx context.Context, receiverMemberID string, filter messagedom.ListFilter) ([]messagedom.Message, error)
	ListSent(ctx context.Context, senderMemberID string, filter messagedom.ListFilter) ([]messagedom.Message, error)

	MarkAsRead(ctx context.Context, id string, readAt time.Time) error
}

type MessageMemberRepo interface {
	GetByID(ctx context.Context, id string) (memberdom.Member, error)
}

type MessageIDGenerator interface {
	NewMessageID() string
}

type MessageImageStorageService interface {
	SaveMessageImage(ctx context.Context, in MessageImageUploadInput) (messagedom.MessageImageAttachment, error)
	DeleteMessageImage(ctx context.Context, storagePath string) error
}

type MessageImageUploadInput struct {
	MessageID        string
	CompanyID        string
	SenderMemberID   string
	ReceiverMemberID string

	FileName    string
	ContentType string
	SizeBytes   int64
	Width       *int64
	Height      *int64
	Data        []byte
}

type SendMessageInput struct {
	ID               string                              `json:"id,omitempty"`
	CompanyID        string                              `json:"companyId,omitempty"`
	SenderMemberID   string                              `json:"senderMemberId"`
	ReceiverMemberID string                              `json:"receiverMemberId"`
	Body             string                              `json:"body,omitempty"`
	Images           []messagedom.MessageImageAttachment `json:"images,omitempty"`

	// ImageUploads are saved to Firebase Storage before the message metadata is saved.
	ImageUploads []MessageImageUploadInput `json:"-"`
}

type MessageView struct {
	ID                  string                              `json:"id"`
	CompanyID           string                              `json:"companyId"`
	SenderMemberID      string                              `json:"senderMemberId"`
	SenderMemberName    string                              `json:"senderMemberName,omitempty"`
	SenderMemberEmail   string                              `json:"senderMemberEmail,omitempty"`
	ReceiverMemberID    string                              `json:"receiverMemberId"`
	ReceiverMemberName  string                              `json:"receiverMemberName,omitempty"`
	ReceiverMemberEmail string                              `json:"receiverMemberEmail,omitempty"`
	PeerMemberID        string                              `json:"peerMemberId,omitempty"`
	PeerMemberName      string                              `json:"peerMemberName,omitempty"`
	PeerMemberEmail     string                              `json:"peerMemberEmail,omitempty"`
	Body                string                              `json:"body,omitempty"`
	Images              []messagedom.MessageImageAttachment `json:"images,omitempty"`
	IsRead              bool                                `json:"isRead"`
	ReadAt              *time.Time                          `json:"readAt,omitempty"`
	CreatedAt           time.Time                           `json:"createdAt"`
	UpdatedAt           time.Time                           `json:"updatedAt"`
}

var (
	ErrMessageRepoNotConfigured          = errors.New("message: repo not configured")
	ErrMessageMemberRepoNotConfigured    = errors.New("message: member repo not configured")
	ErrMessageImageStorageServiceMissing = errors.New("message: image storage service not configured")
)

func (u *MessageUsecase) SendMessage(ctx context.Context, in SendMessageInput) (messagedom.Message, error) {
	if u.msgRepo == nil {
		return messagedom.Message{}, ErrMessageRepoNotConfigured
	}

	messageID := strings.TrimSpace(in.ID)
	if messageID == "" && u.idGen != nil {
		messageID = strings.TrimSpace(u.idGen.NewMessageID())
	}
	if messageID == "" {
		return messagedom.Message{}, messagedom.ErrInvalidID
	}

	senderMemberID := strings.TrimSpace(in.SenderMemberID)
	receiverMemberID := strings.TrimSpace(in.ReceiverMemberID)

	_, _, companyID, err := u.ensureCanSend(
		ctx,
		senderMemberID,
		receiverMemberID,
		in.CompanyID,
	)
	if err != nil {
		return messagedom.Message{}, err
	}

	now := u.now().UTC()

	images := cloneMessageImages(in.Images, now)
	uploadedImages, err := u.saveMessageImages(
		ctx,
		messageID,
		companyID,
		senderMemberID,
		receiverMemberID,
		in.ImageUploads,
		now,
	)
	if err != nil {
		return messagedom.Message{}, err
	}

	images = append(images, uploadedImages...)

	message, err := messagedom.NewForCreate(
		messageID,
		senderMemberID,
		receiverMemberID,
		companyID,
		companyID,
		in.Body,
		images,
		now,
	)
	if err != nil {
		u.cleanupUploadedImages(ctx, uploadedImages)
		return messagedom.Message{}, err
	}

	if err := u.msgRepo.Create(ctx, message); err != nil {
		u.cleanupUploadedImages(ctx, uploadedImages)
		return messagedom.Message{}, err
	}

	return message, nil
}

func (u *MessageUsecase) GetByID(ctx context.Context, id string) (messagedom.Message, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return messagedom.Message{}, messagedom.ErrInvalidID
	}
	if u.msgRepo == nil {
		return messagedom.Message{}, ErrMessageRepoNotConfigured
	}

	return u.msgRepo.FindByID(ctx, id)
}

func (u *MessageUsecase) ListThread(
	ctx context.Context,
	memberID string,
	peerMemberID string,
	filter messagedom.ListFilter,
) ([]messagedom.Message, error) {
	memberID = strings.TrimSpace(memberID)
	peerMemberID = strings.TrimSpace(peerMemberID)

	if memberID == "" {
		return nil, messagedom.ErrInvalidSenderMemberID
	}
	if peerMemberID == "" {
		return nil, messagedom.ErrInvalidReceiverMemberID
	}
	if memberID == peerMemberID {
		return nil, messagedom.ErrSelfMessageNotAllowed
	}
	if u.msgRepo == nil {
		return nil, ErrMessageRepoNotConfigured
	}

	return u.msgRepo.ListThread(ctx, memberID, peerMemberID, filter)
}

func (u *MessageUsecase) ListThreadViews(
	ctx context.Context,
	memberID string,
	peerMemberID string,
	filter messagedom.ListFilter,
) ([]MessageView, error) {
	messages, err := u.ListThread(ctx, memberID, peerMemberID, filter)
	if err != nil {
		return nil, err
	}

	return u.decorateMessagesWithMembers(ctx, memberID, messages), nil
}

func (u *MessageUsecase) ListReceived(
	ctx context.Context,
	receiverMemberID string,
	filter messagedom.ListFilter,
) ([]messagedom.Message, error) {
	receiverMemberID = strings.TrimSpace(receiverMemberID)

	if receiverMemberID == "" {
		return nil, messagedom.ErrInvalidReceiverMemberID
	}
	if u.msgRepo == nil {
		return nil, ErrMessageRepoNotConfigured
	}

	return u.msgRepo.ListReceived(ctx, receiverMemberID, filter)
}

func (u *MessageUsecase) ListReceivedViews(
	ctx context.Context,
	receiverMemberID string,
	filter messagedom.ListFilter,
) ([]MessageView, error) {
	messages, err := u.ListReceived(ctx, receiverMemberID, filter)
	if err != nil {
		return nil, err
	}

	return u.decorateMessagesWithMembers(ctx, receiverMemberID, messages), nil
}

func (u *MessageUsecase) ListSent(
	ctx context.Context,
	senderMemberID string,
	filter messagedom.ListFilter,
) ([]messagedom.Message, error) {
	senderMemberID = strings.TrimSpace(senderMemberID)

	if senderMemberID == "" {
		return nil, messagedom.ErrInvalidSenderMemberID
	}
	if u.msgRepo == nil {
		return nil, ErrMessageRepoNotConfigured
	}

	return u.msgRepo.ListSent(ctx, senderMemberID, filter)
}

func (u *MessageUsecase) ListSentViews(
	ctx context.Context,
	senderMemberID string,
	filter messagedom.ListFilter,
) ([]MessageView, error) {
	messages, err := u.ListSent(ctx, senderMemberID, filter)
	if err != nil {
		return nil, err
	}

	return u.decorateMessagesWithMembers(ctx, senderMemberID, messages), nil
}

func (u *MessageUsecase) MarkAsRead(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return messagedom.ErrInvalidID
	}
	if u.msgRepo == nil {
		return ErrMessageRepoNotConfigured
	}

	return u.msgRepo.MarkAsRead(ctx, id, u.now().UTC())
}

func (u *MessageUsecase) UpdateBody(ctx context.Context, id string, body string) (messagedom.Message, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return messagedom.Message{}, messagedom.ErrInvalidID
	}
	if u.msgRepo == nil {
		return messagedom.Message{}, ErrMessageRepoNotConfigured
	}

	message, err := u.msgRepo.FindByID(ctx, id)
	if err != nil {
		return messagedom.Message{}, err
	}

	if err := message.UpdateBody(body, u.now().UTC()); err != nil {
		return messagedom.Message{}, err
	}

	if err := u.msgRepo.Update(ctx, message); err != nil {
		return messagedom.Message{}, err
	}

	return message, nil
}

func (u *MessageUsecase) SetImages(
	ctx context.Context,
	id string,
	images []messagedom.MessageImageAttachment,
) (messagedom.Message, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return messagedom.Message{}, messagedom.ErrInvalidID
	}
	if u.msgRepo == nil {
		return messagedom.Message{}, ErrMessageRepoNotConfigured
	}

	message, err := u.msgRepo.FindByID(ctx, id)
	if err != nil {
		return messagedom.Message{}, err
	}

	now := u.now().UTC()
	if err := message.SetImages(cloneMessageImages(images, now), now); err != nil {
		return messagedom.Message{}, err
	}

	if err := u.msgRepo.Update(ctx, message); err != nil {
		return messagedom.Message{}, err
	}

	return message, nil
}

func (u *MessageUsecase) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return messagedom.ErrInvalidID
	}
	if u.msgRepo == nil {
		return ErrMessageRepoNotConfigured
	}

	return u.msgRepo.Delete(ctx, id)
}

func (u *MessageUsecase) ensureCanSend(
	ctx context.Context,
	senderMemberID string,
	receiverMemberID string,
	requestCompanyID string,
) (memberdom.Member, memberdom.Member, string, error) {
	senderMemberID = strings.TrimSpace(senderMemberID)
	receiverMemberID = strings.TrimSpace(receiverMemberID)
	requestCompanyID = strings.TrimSpace(requestCompanyID)

	if senderMemberID == "" {
		return memberdom.Member{}, memberdom.Member{}, "", messagedom.ErrInvalidSenderMemberID
	}
	if receiverMemberID == "" {
		return memberdom.Member{}, memberdom.Member{}, "", messagedom.ErrInvalidReceiverMemberID
	}
	if senderMemberID == receiverMemberID {
		return memberdom.Member{}, memberdom.Member{}, "", messagedom.ErrSelfMessageNotAllowed
	}
	if u.memberRepo == nil {
		return memberdom.Member{}, memberdom.Member{}, "", ErrMessageMemberRepoNotConfigured
	}

	senderMember, err := u.memberRepo.GetByID(ctx, senderMemberID)
	if err != nil {
		return memberdom.Member{}, memberdom.Member{}, "", err
	}

	receiverMember, err := u.memberRepo.GetByID(ctx, receiverMemberID)
	if err != nil {
		return memberdom.Member{}, memberdom.Member{}, "", err
	}

	senderCompanyID := strings.TrimSpace(senderMember.CompanyID)
	receiverCompanyID := strings.TrimSpace(receiverMember.CompanyID)

	if requestCompanyID != "" && senderCompanyID != requestCompanyID {
		return memberdom.Member{}, memberdom.Member{}, "", messagedom.ErrMessageNotAllowed
	}
	if requestCompanyID != "" && receiverCompanyID != requestCompanyID {
		return memberdom.Member{}, memberdom.Member{}, "", messagedom.ErrMessageNotAllowed
	}

	if err := messagedom.ValidateMessageRelation(
		senderMemberID,
		receiverMemberID,
		senderCompanyID,
		receiverCompanyID,
	); err != nil {
		return memberdom.Member{}, memberdom.Member{}, "", err
	}

	return senderMember, receiverMember, senderCompanyID, nil
}

func (u *MessageUsecase) saveMessageImages(
	ctx context.Context,
	messageID string,
	companyID string,
	senderMemberID string,
	receiverMemberID string,
	uploads []MessageImageUploadInput,
	now time.Time,
) ([]messagedom.MessageImageAttachment, error) {
	if len(uploads) == 0 {
		return []messagedom.MessageImageAttachment{}, nil
	}
	if u.imageStorage == nil {
		return nil, ErrMessageImageStorageServiceMissing
	}

	images := make([]messagedom.MessageImageAttachment, 0, len(uploads))

	for _, upload := range uploads {
		upload = normalizeMessageImageUploadInput(
			upload,
			messageID,
			companyID,
			senderMemberID,
			receiverMemberID,
		)

		image, err := u.imageStorage.SaveMessageImage(ctx, upload)
		if err != nil {
			u.cleanupUploadedImages(ctx, images)
			return nil, err
		}

		if image.UploadedAt.IsZero() {
			image.UploadedAt = now.UTC()
		} else {
			image.UploadedAt = image.UploadedAt.UTC()
		}

		images = append(images, image)
	}

	return images, nil
}

func (u *MessageUsecase) cleanupUploadedImages(ctx context.Context, images []messagedom.MessageImageAttachment) {
	if u.imageStorage == nil {
		return
	}

	for _, image := range images {
		if image.StoragePath == "" {
			continue
		}
		_ = u.imageStorage.DeleteMessageImage(ctx, image.StoragePath)
	}
}

func (u *MessageUsecase) decorateMessagesWithMembers(
	ctx context.Context,
	currentMemberID string,
	messages []messagedom.Message,
) []MessageView {
	members := u.memberMapByMessages(ctx, messages)

	views := make([]MessageView, 0, len(messages))
	for _, message := range messages {
		views = append(views, newMessageView(message, currentMemberID, members))
	}

	return views
}

func (u *MessageUsecase) memberMapByMessages(
	ctx context.Context,
	messages []messagedom.Message,
) map[string]memberdom.Member {
	members := make(map[string]memberdom.Member)

	if u.memberRepo == nil {
		return members
	}

	for _, message := range messages {
		for _, memberID := range []string{message.SenderMemberID, message.ReceiverMemberID} {
			memberID = strings.TrimSpace(memberID)
			if memberID == "" {
				continue
			}
			if _, ok := members[memberID]; ok {
				continue
			}

			member, err := u.memberRepo.GetByID(ctx, memberID)
			if err != nil {
				continue
			}

			members[memberID] = member
		}
	}

	return members
}

func newMessageView(
	message messagedom.Message,
	currentMemberID string,
	members map[string]memberdom.Member,
) MessageView {
	sender := members[message.SenderMemberID]
	receiver := members[message.ReceiverMemberID]

	peerMemberID := message.SenderMemberID
	if peerMemberID == currentMemberID {
		peerMemberID = message.ReceiverMemberID
	}

	peer := members[peerMemberID]

	return MessageView{
		ID:                  message.ID,
		CompanyID:           message.CompanyID,
		SenderMemberID:      message.SenderMemberID,
		SenderMemberName:    memberNameOrID(sender, message.SenderMemberID),
		SenderMemberEmail:   strings.TrimSpace(sender.Email),
		ReceiverMemberID:    message.ReceiverMemberID,
		ReceiverMemberName:  memberNameOrID(receiver, message.ReceiverMemberID),
		ReceiverMemberEmail: strings.TrimSpace(receiver.Email),
		PeerMemberID:        peerMemberID,
		PeerMemberName:      memberNameOrID(peer, peerMemberID),
		PeerMemberEmail:     strings.TrimSpace(peer.Email),
		Body:                message.Body,
		Images:              message.Images,
		IsRead:              message.IsRead,
		ReadAt:              message.ReadAt,
		CreatedAt:           message.CreatedAt,
		UpdatedAt:           message.UpdatedAt,
	}
}

func memberNameOrID(member memberdom.Member, fallbackID string) string {
	name := strings.TrimSpace(memberdom.FormatLastFirst(member.LastName, member.FirstName))
	if name != "" {
		return name
	}

	email := strings.TrimSpace(member.Email)
	if email != "" {
		return email
	}

	return strings.TrimSpace(fallbackID)
}

func normalizeMessageImageUploadInput(
	in MessageImageUploadInput,
	messageID string,
	companyID string,
	senderMemberID string,
	receiverMemberID string,
) MessageImageUploadInput {
	if in.MessageID == "" {
		in.MessageID = messageID
	}
	if in.CompanyID == "" {
		in.CompanyID = companyID
	}
	if in.SenderMemberID == "" {
		in.SenderMemberID = senderMemberID
	}
	if in.ReceiverMemberID == "" {
		in.ReceiverMemberID = receiverMemberID
	}

	return in
}

func cloneMessageImages(
	in []messagedom.MessageImageAttachment,
	defaultUploadedAt time.Time,
) []messagedom.MessageImageAttachment {
	if len(in) == 0 {
		return []messagedom.MessageImageAttachment{}
	}

	out := make([]messagedom.MessageImageAttachment, 0, len(in))

	for _, item := range in {
		uploadedAt := item.UploadedAt
		if uploadedAt.IsZero() {
			uploadedAt = defaultUploadedAt
		}

		out = append(out, messagedom.MessageImageAttachment{
			StoragePath: item.StoragePath,
			DownloadURL: cloneStringPtr(item.DownloadURL),
			ContentType: item.ContentType,
			SizeBytes:   item.SizeBytes,
			Width:       cloneInt64Ptr(item.Width),
			Height:      cloneInt64Ptr(item.Height),
			UploadedAt:  uploadedAt.UTC(),
		})
	}

	return out
}

func cloneStringPtr(in *string) *string {
	if in == nil {
		return nil
	}

	v := *in
	return &v
}

func cloneInt64Ptr(in *int64) *int64 {
	if in == nil {
		return nil
	}

	v := *in
	return &v
}
