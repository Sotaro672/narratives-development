// backend/internal/adapters/out/firestore/inquiry_reply_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	idom "narratives/internal/domain/inquiry"
)

// InquiryReplyRepositoryFS implements Inquiry reply repository using Firestore.
//
// 保存先:
//
//	inquiries/{inquiryId}/replies/{replyId}
type InquiryReplyRepositoryFS struct {
	Client *firestore.Client
}

func NewInquiryReplyRepositoryFS(client *firestore.Client) *InquiryReplyRepositoryFS {
	return &InquiryReplyRepositoryFS{Client: client}
}

// Compile-time check.
var _ idom.ReplyRepository = (*InquiryReplyRepositoryFS)(nil)

func (r *InquiryReplyRepositoryFS) col(inquiryID string) *firestore.CollectionRef {
	return r.Client.
		Collection("inquiries").
		Doc(inquiryID).
		Collection("replies")
}

func (r *InquiryReplyRepositoryFS) Create(
	ctx context.Context,
	reply idom.Reply,
) (idom.Reply, error) {
	if r.Client == nil {
		return idom.Reply{}, errors.New("firestore client is nil")
	}

	if err := reply.Validate(); err != nil {
		return idom.Reply{}, err
	}

	docRef := r.col(reply.InquiryID).Doc(reply.ID)

	_, err := docRef.Create(ctx, replyToDocData(reply))
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return idom.Reply{}, idom.ErrConflict
		}
		return idom.Reply{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return idom.Reply{}, err
	}

	return docToReplyWithFallbackInquiryID(snap, reply.InquiryID)
}

func (r *InquiryReplyRepositoryFS) ListByInquiryID(
	ctx context.Context,
	inquiryID string,
) ([]idom.Reply, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if inquiryID == "" {
		return nil, idom.ErrInvalidReplyInquiryID
	}

	it := r.col(inquiryID).
		OrderBy("createdAt", firestore.Asc).
		OrderBy("id", firestore.Asc).
		Documents(ctx)
	defer it.Stop()

	replies := make([]idom.Reply, 0)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		reply, err := docToReplyWithFallbackInquiryID(doc, inquiryID)
		if err != nil {
			return nil, err
		}

		replies = append(replies, reply)
	}

	return replies, nil
}

func (r *InquiryReplyRepositoryFS) CountUnreadByAvatarID(
	ctx context.Context,
	avatarID string,
	filter idom.Filter,
) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}
	if avatarID == "" {
		return 0, idom.ErrInvalidAvatarID
	}

	filter.AvatarID = &avatarID

	inquiryIt := r.Client.
		Collection("inquiries").
		Where("avatarId", "==", avatarID).
		Documents(ctx)
	defer inquiryIt.Stop()

	count := 0

	for {
		inquiryDoc, err := inquiryIt.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}

		in, err := docToInquiry(inquiryDoc)
		if err != nil {
			return 0, err
		}

		if !matchInquiryFilter(in, filter) {
			continue
		}

		replyCount, err := r.CountUnreadByInquiryIDExcludingSender(
			ctx,
			in.ID,
			idom.ReplySenderTypeAvatar,
			avatarID,
		)
		if err != nil {
			return 0, err
		}

		count += replyCount
	}

	return count, nil
}

func (r *InquiryReplyRepositoryFS) MarkAsReadByInquiryID(
	ctx context.Context,
	inquiryID string,
	readerSenderType idom.ReplySenderType,
	readerSenderID string,
	readAt time.Time,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	if inquiryID == "" {
		return idom.ErrInvalidReplyInquiryID
	}
	if readerSenderType == "" {
		return idom.ErrInvalidReplySenderType
	}
	if readerSenderID == "" {
		return idom.ErrInvalidReplySenderID
	}
	if readAt.IsZero() {
		return idom.ErrInvalidReplyUpdatedAt
	}

	updatedAt := readAt.UTC()

	it := r.col(inquiryID).Documents(ctx)
	defer it.Stop()

	bulk := r.Client.BulkWriter(ctx)
	defer bulk.End()

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		data := doc.Data()
		if data == nil {
			return fmt.Errorf("empty inquiry reply document: %s", doc.Ref.ID)
		}

		senderType := idom.ReplySenderType(asString(data["senderType"]))
		senderID := asString(data["senderId"])
		isRead := asBool(data["isRead"])

		if senderType == readerSenderType && senderID == readerSenderID {
			continue
		}

		if isRead {
			continue
		}

		_, err = bulk.Update(doc.Ref, []firestore.Update{
			{Path: "isRead", Value: true},
			{Path: "updatedAt", Value: updatedAt},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// CountUnreadByInquiryIDExcludingSender は、指定 inquiry 配下の未読 reply 数を返します。
//
// count 条件:
//
//	!reply.isRead
//	&& !(reply.senderType == excludedSenderType && reply.senderId == excludedSenderID)
//
// usecase 側では member / avatar どちらもこの考え方で集計できます。
// 現在の usecase interface では必須メソッドではありませんが、
// repository 側の任意拡張として用意しています。
func (r *InquiryReplyRepositoryFS) CountUnreadByInquiryIDExcludingSender(
	ctx context.Context,
	inquiryID string,
	excludedSenderType idom.ReplySenderType,
	excludedSenderID string,
) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}
	if inquiryID == "" {
		return 0, idom.ErrInvalidReplyInquiryID
	}
	if excludedSenderType == "" {
		return 0, idom.ErrInvalidReplySenderType
	}
	if excludedSenderID == "" {
		return 0, idom.ErrInvalidReplySenderID
	}

	it := r.col(inquiryID).Documents(ctx)
	defer it.Stop()

	count := 0

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}

		data := doc.Data()
		if data == nil {
			return 0, fmt.Errorf("empty inquiry reply document: %s", doc.Ref.ID)
		}

		senderType := idom.ReplySenderType(asString(data["senderType"]))
		senderID := asString(data["senderId"])
		isRead := asBool(data["isRead"])

		if isRead {
			continue
		}

		if senderType == excludedSenderType && senderID == excludedSenderID {
			continue
		}

		count++
	}

	return count, nil
}
