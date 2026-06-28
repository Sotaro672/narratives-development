// backend\internal\adapters\in\http\mall\handler\message_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	messageuc "narratives/internal/application/usecase"
	avatardom "narratives/internal/domain/avatar"
	messagedom "narratives/internal/domain/message"
)

// Endpoints:
// - POST   /mall/me/messages
// - GET    /mall/me/messages?peerAvatarId=...
// - GET    /mall/me/messages/received
// - GET    /mall/me/messages/sent
// - GET    /mall/me/messages/{messageId}
// - PATCH  /mall/me/messages/{messageId}
// - DELETE /mall/me/messages/{messageId}
// - POST   /mall/me/messages/{messageId}/read
// - PATCH  /mall/me/messages/{messageId}/read

type MessageAvatarResolver interface {
	ResolveAvatarByUID(ctx context.Context, uid string) (avatarID string, walletAddress string, err error)
}

type MessageHandler struct {
	Repo      MessageAvatarResolver
	MessageUC *messageuc.MessageUsecase
}

func NewMessageHandler(
	repo MessageAvatarResolver,
	messageUC *messageuc.MessageUsecase,
) http.Handler {
	return &MessageHandler{
		Repo:      repo,
		MessageUC: messageUC,
	}
}

const (
	meMessagesPath         = "/mall/me/messages"
	meMessagesReceivedPath = "/mall/me/messages/received"
	meMessagesSentPath     = "/mall/me/messages/sent"

	maxMessageMultipartMemory = 32 << 20
)

func (h *MessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if h == nil || h.Repo == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "message_handler_not_initialized"})
		return
	}

	if h.MessageUC == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "message_usecase_not_configured"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing uid"})
		return
	}

	myAvatarID, _, err := h.Repo.ResolveAvatarByUID(r.Context(), uid)
	if err != nil {
		writeMessageErr(w, err)
		return
	}
	if myAvatarID == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodPost && path0 == meMessagesPath:
		h.handleSend(w, r, myAvatarID)
		return

	case r.Method == http.MethodGet && path0 == meMessagesPath:
		h.handleList(w, r, myAvatarID)
		return

	case r.Method == http.MethodGet && path0 == meMessagesReceivedPath:
		h.handleListReceived(w, r, myAvatarID)
		return

	case r.Method == http.MethodGet && path0 == meMessagesSentPath:
		h.handleListSent(w, r, myAvatarID)
		return
	}

	messageID, action, ok := parseMessageSubpath(path0)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		h.handleGet(w, r, myAvatarID, messageID)
		return

	case r.Method == http.MethodPatch && action == "":
		h.handlePatch(w, r, myAvatarID, messageID)
		return

	case r.Method == http.MethodDelete && action == "":
		h.handleDelete(w, r, myAvatarID, messageID)
		return

	case (r.Method == http.MethodPost || r.Method == http.MethodPatch) && action == "read":
		h.handleMarkAsRead(w, r, myAvatarID, messageID)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

func (h *MessageHandler) handleSend(w http.ResponseWriter, r *http.Request, myAvatarID string) {
	in, err := parseSendMessageInput(r, myAvatarID)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	message, err := h.MessageUC.SendMessage(r.Context(), in)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(message)
}

func (h *MessageHandler) handleList(w http.ResponseWriter, r *http.Request, myAvatarID string) {
	filter, err := messageListFilterFromRequest(r)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	peerAvatarID := r.URL.Query().Get("peerAvatarId")
	if peerAvatarID != "" {
		messages, err := h.MessageUC.ListThread(r.Context(), myAvatarID, peerAvatarID, filter)
		if err != nil {
			writeMessageErr(w, err)
			return
		}
		writeMessageList(w, messages)
		return
	}

	messages, err := h.MessageUC.ListReceived(r.Context(), myAvatarID, filter)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	writeMessageList(w, messages)
}

func (h *MessageHandler) handleListReceived(w http.ResponseWriter, r *http.Request, myAvatarID string) {
	filter, err := messageListFilterFromRequest(r)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	messages, err := h.MessageUC.ListReceived(r.Context(), myAvatarID, filter)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	writeMessageList(w, messages)
}

func (h *MessageHandler) handleListSent(w http.ResponseWriter, r *http.Request, myAvatarID string) {
	filter, err := messageListFilterFromRequest(r)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	messages, err := h.MessageUC.ListSent(r.Context(), myAvatarID, filter)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	writeMessageList(w, messages)
}

func (h *MessageHandler) handleGet(w http.ResponseWriter, r *http.Request, myAvatarID string, messageID string) {
	message, err := h.MessageUC.GetByID(r.Context(), messageID)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	if !isMessageParticipant(message, myAvatarID) {
		writeMessageAccessDenied(w)
		return
	}

	_ = json.NewEncoder(w).Encode(message)
}

func (h *MessageHandler) handlePatch(w http.ResponseWriter, r *http.Request, myAvatarID string, messageID string) {
	var req struct {
		Body   *string                              `json:"body,omitempty"`
		Images *[]messagedom.MessageImageAttachment `json:"images,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.Body == nil && req.Images == nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no_fields_to_update"})
		return
	}

	message, err := h.MessageUC.GetByID(r.Context(), messageID)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	if message.SenderAvatarID != myAvatarID {
		writeMessageAccessDenied(w)
		return
	}

	if req.Body != nil {
		message, err = h.MessageUC.UpdateBody(r.Context(), messageID, *req.Body)
		if err != nil {
			writeMessageErr(w, err)
			return
		}
	}

	if req.Images != nil {
		message, err = h.MessageUC.SetImages(r.Context(), messageID, *req.Images)
		if err != nil {
			writeMessageErr(w, err)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(message)
}

func (h *MessageHandler) handleDelete(w http.ResponseWriter, r *http.Request, myAvatarID string, messageID string) {
	message, err := h.MessageUC.GetByID(r.Context(), messageID)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	if message.SenderAvatarID != myAvatarID {
		writeMessageAccessDenied(w)
		return
	}

	if err := h.MessageUC.Delete(r.Context(), messageID); err != nil {
		writeMessageErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MessageHandler) handleMarkAsRead(w http.ResponseWriter, r *http.Request, myAvatarID string, messageID string) {
	message, err := h.MessageUC.GetByID(r.Context(), messageID)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	if message.ReceiverAvatarID != myAvatarID {
		writeMessageAccessDenied(w)
		return
	}

	if err := h.MessageUC.MarkAsRead(r.Context(), messageID); err != nil {
		writeMessageErr(w, err)
		return
	}

	updated, err := h.MessageUC.GetByID(r.Context(), messageID)
	if err != nil {
		writeMessageErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
}

func parseSendMessageInput(r *http.Request, myAvatarID string) (messageuc.SendMessageInput, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return parseMultipartSendMessageInput(r, myAvatarID)
	}

	var req struct {
		ID               string                              `json:"id,omitempty"`
		ReceiverAvatarID string                              `json:"receiverAvatarId"`
		Body             string                              `json:"body,omitempty"`
		Images           []messagedom.MessageImageAttachment `json:"images,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return messageuc.SendMessageInput{}, errors.New("invalid_json")
	}

	return messageuc.SendMessageInput{
		ID:               req.ID,
		SenderAvatarID:   myAvatarID,
		ReceiverAvatarID: req.ReceiverAvatarID,
		Body:             req.Body,
		Images:           req.Images,
	}, nil
}

func parseMultipartSendMessageInput(r *http.Request, myAvatarID string) (messageuc.SendMessageInput, error) {
	if err := r.ParseMultipartForm(maxMessageMultipartMemory); err != nil {
		return messageuc.SendMessageInput{}, err
	}

	messageID := r.FormValue("id")
	receiverAvatarID := r.FormValue("receiverAvatarId")
	body := r.FormValue("body")

	fileHeaders := r.MultipartForm.File["images"]
	if len(fileHeaders) == 0 {
		fileHeaders = r.MultipartForm.File["image"]
	}

	if len(fileHeaders) > messagedom.MaxImageAttachmentCount {
		return messageuc.SendMessageInput{}, messagedom.ErrTooManyImages
	}

	uploads := make([]messageuc.MessageImageUploadInput, 0, len(fileHeaders))
	for _, fh := range fileHeaders {
		data, contentType, err := readMessageUploadFile(fh.Open, fh.Header.Get("Content-Type"))
		if err != nil {
			return messageuc.SendMessageInput{}, err
		}

		if !isAllowedMessageImageContentType(contentType) {
			return messageuc.SendMessageInput{}, messagedom.ErrInvalidImageContentType
		}

		uploads = append(uploads, messageuc.MessageImageUploadInput{
			MessageID:        messageID,
			SenderAvatarID:   myAvatarID,
			ReceiverAvatarID: receiverAvatarID,
			FileName:         fh.Filename,
			ContentType:      contentType,
			SizeBytes:        int64(len(data)),
			Data:             data,
		})
	}

	return messageuc.SendMessageInput{
		ID:               messageID,
		SenderAvatarID:   myAvatarID,
		ReceiverAvatarID: receiverAvatarID,
		Body:             body,
		ImageUploads:     uploads,
	}, nil
}

func readMessageUploadFile(
	open func() (multipart.File, error),
	contentType string,
) ([]byte, string, error) {
	file, err := open()
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, messagedom.MaxImageSizeBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > messagedom.MaxImageSizeBytes {
		return nil, "", messagedom.ErrInvalidImageSize
	}

	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

func messageListFilterFromRequest(r *http.Request) (messagedom.ListFilter, error) {
	q := r.URL.Query()

	filter := messagedom.ListFilter{}

	if rawLimit := q.Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			return messagedom.ListFilter{}, errors.New("invalid_limit")
		}
		filter.Limit = limit
	}

	rawBefore := q.Get("beforeCreatedAt")
	if rawBefore == "" {
		rawBefore = q.Get("before")
	}
	if rawBefore != "" {
		t, err := time.Parse(time.RFC3339, rawBefore)
		if err != nil {
			return messagedom.ListFilter{}, errors.New("invalid_beforeCreatedAt")
		}
		t = t.UTC()
		filter.BeforeCreatedAt = &t
	}

	return filter, nil
}

func parseMessageSubpath(path0 string) (messageID string, action string, ok bool) {
	prefix := meMessagesPath + "/"
	if !strings.HasPrefix(path0, prefix) {
		return "", "", false
	}

	rest := strings.TrimPrefix(path0, prefix)
	parts := strings.Split(rest, "/")

	if len(parts) == 1 && parts[0] != "" {
		return parts[0], "", true
	}

	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], true
	}

	return "", "", false
}

func isMessageParticipant(message messagedom.Message, avatarID string) bool {
	return message.SenderAvatarID == avatarID || message.ReceiverAvatarID == avatarID
}

func writeMessageList(w http.ResponseWriter, messages []messagedom.Message) {
	type messageListResponse struct {
		Messages []messagedom.Message `json:"messages"`
		Count    int                  `json:"count"`
	}

	_ = json.NewEncoder(w).Encode(messageListResponse{
		Messages: messages,
		Count:    len(messages),
	})
}

func writeMessageAccessDenied(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "message_access_denied"})
}

func writeMessageErr(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	msg := err.Error()

	switch {
	case msg == "invalid_json",
		msg == "invalid_limit",
		msg == "invalid_beforeCreatedAt":
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return

	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		w.WriteHeader(http.StatusRequestTimeout)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "request_timeout"})
		return

	case errors.Is(err, messagedom.ErrNotFound), isNotFoundLike(err):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "message_not_found"})
		return

	case errors.Is(err, messagedom.ErrMessageNotAllowed):
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	case errors.Is(err, messagedom.ErrInvalidID),
		errors.Is(err, messagedom.ErrInvalidSenderAvatarID),
		errors.Is(err, messagedom.ErrInvalidReceiverAvatarID),
		errors.Is(err, messagedom.ErrSelfMessageNotAllowed),
		errors.Is(err, messagedom.ErrInvalidBody),
		errors.Is(err, messagedom.ErrEmptyMessage),
		errors.Is(err, messagedom.ErrInvalidReadAt),
		errors.Is(err, messagedom.ErrTooManyImages),
		errors.Is(err, messagedom.ErrInvalidImageStoragePath),
		errors.Is(err, messagedom.ErrInvalidImageDownloadURL),
		errors.Is(err, messagedom.ErrInvalidImageContentType),
		errors.Is(err, messagedom.ErrInvalidImageSize),
		errors.Is(err, messagedom.ErrInvalidImageDimensions),
		errors.Is(err, messagedom.ErrInvalidImageUploadedAt),
		errors.Is(err, messagedom.ErrDuplicateImage),
		errors.Is(err, avatardom.ErrInvalidID):
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	case errors.Is(err, messageuc.ErrMessageRepoNotConfigured),
		errors.Is(err, messageuc.ErrMessageAvatarRepoNotConfigured),
		errors.Is(err, messageuc.ErrMessageAvatarStateRepoNotConfigured),
		errors.Is(err, messageuc.ErrMessageImageStorageServiceMissing):
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	default:
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}
}

func isAllowedMessageImageContentType(contentType string) bool {
	switch strings.ToLower(contentType) {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}
