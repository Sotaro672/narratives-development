// backend\internal\application\usecase\message_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	avatardom "narratives/internal/domain/avatar"
	avatarstate "narratives/internal/domain/avatarState"
	messagedom "narratives/internal/domain/message"
)

type MessageUsecase struct {
	msgRepo      MessageRepo
	avatarRepo   MessageAvatarRepo
	stRepo       MessageAvatarStateRepo
	imageStorage MessageImageStorageService
	idGen        MessageIDGenerator

	now func() time.Time
}

func NewMessageUsecase(
	msgRepo MessageRepo,
	avatarRepo MessageAvatarRepo,
	stRepo MessageAvatarStateRepo,
	imageStorage MessageImageStorageService,
	idGen MessageIDGenerator,
	now func() time.Time,
) *MessageUsecase {
	if now == nil {
		now = time.Now
	}

	return &MessageUsecase{
		msgRepo:      msgRepo,
		avatarRepo:   avatarRepo,
		stRepo:       stRepo,
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

	ListThread(ctx context.Context, avatarID, peerAvatarID string, filter messagedom.ListFilter) ([]messagedom.Message, error)
	ListReceived(ctx context.Context, receiverAvatarID string, filter messagedom.ListFilter) ([]messagedom.Message, error)
	ListSent(ctx context.Context, senderAvatarID string, filter messagedom.ListFilter) ([]messagedom.Message, error)

	MarkAsRead(ctx context.Context, id string, readAt time.Time) error
}

type MessageAvatarRepo interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

type MessageAvatarStateRepo interface {
	GetByAvatarID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error)
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
	SenderAvatarID   string
	ReceiverAvatarID string

	FileName    string
	ContentType string
	SizeBytes   int64
	Width       *int64
	Height      *int64
	Data        []byte
}

type SendMessageInput struct {
	ID               string                              `json:"id,omitempty"`
	SenderAvatarID   string                              `json:"senderAvatarId"`
	ReceiverAvatarID string                              `json:"receiverAvatarId"`
	Body             string                              `json:"body,omitempty"`
	Images           []messagedom.MessageImageAttachment `json:"images,omitempty"`

	// ImageUploads are saved to Firebase Storage before the message metadata is saved.
	ImageUploads []MessageImageUploadInput `json:"-"`
}

type MessageView struct {
	ID                 string                              `json:"id"`
	SenderAvatarID     string                              `json:"senderAvatarId"`
	SenderAvatarName   string                              `json:"senderAvatarName,omitempty"`
	SenderAvatarIcon   *string                             `json:"senderAvatarIcon,omitempty"`
	ReceiverAvatarID   string                              `json:"receiverAvatarId"`
	ReceiverAvatarName string                              `json:"receiverAvatarName,omitempty"`
	ReceiverAvatarIcon *string                             `json:"receiverAvatarIcon,omitempty"`
	PeerAvatarID       string                              `json:"peerAvatarId,omitempty"`
	PeerAvatarName     string                              `json:"peerAvatarName,omitempty"`
	PeerAvatarIcon     *string                             `json:"peerAvatarIcon,omitempty"`
	Body               string                              `json:"body,omitempty"`
	Images             []messagedom.MessageImageAttachment `json:"images,omitempty"`
	IsRead             bool                                `json:"isRead"`
	ReadAt             *time.Time                          `json:"readAt,omitempty"`
	CreatedAt          time.Time                           `json:"createdAt"`
	UpdatedAt          time.Time                           `json:"updatedAt"`
}

var (
	ErrMessageRepoNotConfigured            = errors.New("message: repo not configured")
	ErrMessageAvatarRepoNotConfigured      = errors.New("message: avatar repo not configured")
	ErrMessageAvatarStateRepoNotConfigured = errors.New("message: avatarState repo not configured")
	ErrMessageImageStorageServiceMissing   = errors.New("message: image storage service not configured")
)

func (u *MessageUsecase) SendMessage(ctx context.Context, in SendMessageInput) (messagedom.Message, error) {
	if u.msgRepo == nil {
		return messagedom.Message{}, ErrMessageRepoNotConfigured
	}

	messageID := in.ID
	if messageID == "" && u.idGen != nil {
		messageID = u.idGen.NewMessageID()
	}
	if messageID == "" {
		return messagedom.Message{}, messagedom.ErrInvalidID
	}

	senderState, err := u.ensureCanSend(ctx, in.SenderAvatarID, in.ReceiverAvatarID)
	if err != nil {
		return messagedom.Message{}, err
	}

	now := u.now().UTC()

	images := cloneMessageImages(in.Images, now)
	uploadedImages, err := u.saveMessageImages(
		ctx,
		messageID,
		in.SenderAvatarID,
		in.ReceiverAvatarID,
		in.ImageUploads,
		now,
	)
	if err != nil {
		return messagedom.Message{}, err
	}

	images = append(images, uploadedImages...)

	message, err := messagedom.NewForCreate(
		messageID,
		in.SenderAvatarID,
		in.ReceiverAvatarID,
		in.Body,
		images,
		senderState,
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
	avatarID string,
	peerAvatarID string,
	filter messagedom.ListFilter,
) ([]messagedom.Message, error) {
	if avatarID == "" {
		return nil, messagedom.ErrInvalidSenderAvatarID
	}
	if peerAvatarID == "" {
		return nil, messagedom.ErrInvalidReceiverAvatarID
	}
	if avatarID == peerAvatarID {
		return nil, messagedom.ErrSelfMessageNotAllowed
	}
	if u.msgRepo == nil {
		return nil, ErrMessageRepoNotConfigured
	}

	return u.msgRepo.ListThread(ctx, avatarID, peerAvatarID, filter)
}

func (u *MessageUsecase) ListThreadViews(
	ctx context.Context,
	avatarID string,
	peerAvatarID string,
	filter messagedom.ListFilter,
) ([]MessageView, error) {
	messages, err := u.ListThread(ctx, avatarID, peerAvatarID, filter)
	if err != nil {
		return nil, err
	}

	return u.decorateMessagesWithAvatars(ctx, avatarID, messages), nil
}

func (u *MessageUsecase) ListReceived(
	ctx context.Context,
	receiverAvatarID string,
	filter messagedom.ListFilter,
) ([]messagedom.Message, error) {
	if receiverAvatarID == "" {
		return nil, messagedom.ErrInvalidReceiverAvatarID
	}
	if u.msgRepo == nil {
		return nil, ErrMessageRepoNotConfigured
	}

	return u.msgRepo.ListReceived(ctx, receiverAvatarID, filter)
}

func (u *MessageUsecase) ListReceivedViews(
	ctx context.Context,
	receiverAvatarID string,
	filter messagedom.ListFilter,
) ([]MessageView, error) {
	messages, err := u.ListReceived(ctx, receiverAvatarID, filter)
	if err != nil {
		return nil, err
	}

	return u.decorateMessagesWithAvatars(ctx, receiverAvatarID, messages), nil
}

func (u *MessageUsecase) ListSent(
	ctx context.Context,
	senderAvatarID string,
	filter messagedom.ListFilter,
) ([]messagedom.Message, error) {
	if senderAvatarID == "" {
		return nil, messagedom.ErrInvalidSenderAvatarID
	}
	if u.msgRepo == nil {
		return nil, ErrMessageRepoNotConfigured
	}

	return u.msgRepo.ListSent(ctx, senderAvatarID, filter)
}

func (u *MessageUsecase) ListSentViews(
	ctx context.Context,
	senderAvatarID string,
	filter messagedom.ListFilter,
) ([]MessageView, error) {
	messages, err := u.ListSent(ctx, senderAvatarID, filter)
	if err != nil {
		return nil, err
	}

	return u.decorateMessagesWithAvatars(ctx, senderAvatarID, messages), nil
}

func (u *MessageUsecase) MarkAsRead(ctx context.Context, id string) error {
	if id == "" {
		return messagedom.ErrInvalidID
	}
	if u.msgRepo == nil {
		return ErrMessageRepoNotConfigured
	}

	return u.msgRepo.MarkAsRead(ctx, id, u.now().UTC())
}

func (u *MessageUsecase) UpdateBody(ctx context.Context, id string, body string) (messagedom.Message, error) {
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
	senderAvatarID string,
	receiverAvatarID string,
) (avatarstate.AvatarState, error) {
	if senderAvatarID == "" {
		return avatarstate.AvatarState{}, messagedom.ErrInvalidSenderAvatarID
	}
	if receiverAvatarID == "" {
		return avatarstate.AvatarState{}, messagedom.ErrInvalidReceiverAvatarID
	}
	if senderAvatarID == receiverAvatarID {
		return avatarstate.AvatarState{}, messagedom.ErrSelfMessageNotAllowed
	}
	if u.avatarRepo == nil {
		return avatarstate.AvatarState{}, ErrMessageAvatarRepoNotConfigured
	}
	if u.stRepo == nil {
		return avatarstate.AvatarState{}, ErrMessageAvatarStateRepoNotConfigured
	}

	if _, err := u.avatarRepo.GetByID(ctx, senderAvatarID); err != nil {
		return avatarstate.AvatarState{}, err
	}
	if _, err := u.avatarRepo.GetByID(ctx, receiverAvatarID); err != nil {
		return avatarstate.AvatarState{}, err
	}

	senderState, err := u.stRepo.GetByAvatarID(ctx, senderAvatarID)
	if err != nil {
		return avatarstate.AvatarState{}, err
	}
	if senderState.ID == "" {
		senderState.ID = senderAvatarID
	}

	if err := messagedom.ValidateMessageRelation(senderState, senderAvatarID, receiverAvatarID); err != nil {
		return avatarstate.AvatarState{}, err
	}

	return senderState, nil
}

func (u *MessageUsecase) saveMessageImages(
	ctx context.Context,
	messageID string,
	senderAvatarID string,
	receiverAvatarID string,
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
		upload = normalizeMessageImageUploadInput(upload, messageID, senderAvatarID, receiverAvatarID)

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

func (u *MessageUsecase) decorateMessagesWithAvatars(
	ctx context.Context,
	currentAvatarID string,
	messages []messagedom.Message,
) []MessageView {
	avatars := u.avatarMapByMessages(ctx, messages)

	views := make([]MessageView, 0, len(messages))
	for _, message := range messages {
		views = append(views, newMessageView(message, currentAvatarID, avatars))
	}

	return views
}

func (u *MessageUsecase) avatarMapByMessages(
	ctx context.Context,
	messages []messagedom.Message,
) map[string]avatardom.Avatar {
	avatars := make(map[string]avatardom.Avatar)

	if u.avatarRepo == nil {
		return avatars
	}

	for _, message := range messages {
		for _, avatarID := range []string{message.SenderAvatarID, message.ReceiverAvatarID} {
			avatarID = strings.TrimSpace(avatarID)
			if avatarID == "" {
				continue
			}
			if _, ok := avatars[avatarID]; ok {
				continue
			}

			avatar, err := u.avatarRepo.GetByID(ctx, avatarID)
			if err != nil {
				continue
			}

			avatars[avatarID] = avatar
		}
	}

	return avatars
}

func newMessageView(
	message messagedom.Message,
	currentAvatarID string,
	avatars map[string]avatardom.Avatar,
) MessageView {
	sender := avatars[message.SenderAvatarID]
	receiver := avatars[message.ReceiverAvatarID]

	peerAvatarID := message.SenderAvatarID
	if peerAvatarID == currentAvatarID {
		peerAvatarID = message.ReceiverAvatarID
	}

	peer := avatars[peerAvatarID]

	return MessageView{
		ID:                 message.ID,
		SenderAvatarID:     message.SenderAvatarID,
		SenderAvatarName:   avatarNameOrID(sender, message.SenderAvatarID),
		SenderAvatarIcon:   cloneStringPtr(sender.AvatarIcon),
		ReceiverAvatarID:   message.ReceiverAvatarID,
		ReceiverAvatarName: avatarNameOrID(receiver, message.ReceiverAvatarID),
		ReceiverAvatarIcon: cloneStringPtr(receiver.AvatarIcon),
		PeerAvatarID:       peerAvatarID,
		PeerAvatarName:     avatarNameOrID(peer, peerAvatarID),
		PeerAvatarIcon:     cloneStringPtr(peer.AvatarIcon),
		Body:               message.Body,
		Images:             message.Images,
		IsRead:             message.IsRead,
		ReadAt:             message.ReadAt,
		CreatedAt:          message.CreatedAt,
		UpdatedAt:          message.UpdatedAt,
	}
}

func avatarNameOrID(avatar avatardom.Avatar, fallbackID string) string {
	if name := strings.TrimSpace(avatar.AvatarName); name != "" {
		return name
	}

	return strings.TrimSpace(fallbackID)
}

func normalizeMessageImageUploadInput(
	in MessageImageUploadInput,
	messageID string,
	senderAvatarID string,
	receiverAvatarID string,
) MessageImageUploadInput {
	if in.MessageID == "" {
		in.MessageID = messageID
	}
	if in.SenderAvatarID == "" {
		in.SenderAvatarID = senderAvatarID
	}
	if in.ReceiverAvatarID == "" {
		in.ReceiverAvatarID = receiverAvatarID
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
