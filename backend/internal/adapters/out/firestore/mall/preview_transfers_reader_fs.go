// backend/internal/adapters/out/firestore/mall/preview_transfers_reader_fs.go
package mall

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	dto "narratives/internal/application/query/mall/dto"
)

const defaultPreviewTransfersCollection = "transfers"

type PreviewTransfersReaderFS struct {
	client         *firestore.Client
	collectionName string
}

func NewPreviewTransfersReaderFS(client *firestore.Client) *PreviewTransfersReaderFS {
	return &PreviewTransfersReaderFS{
		client:         client,
		collectionName: defaultPreviewTransfersCollection,
	}
}

func NewPreviewTransfersReaderFSWithCollection(
	client *firestore.Client,
	collectionName string,
) *PreviewTransfersReaderFS {
	if collectionName == "" {
		collectionName = defaultPreviewTransfersCollection
	}
	return &PreviewTransfersReaderFS{
		client:         client,
		collectionName: collectionName,
	}
}

func (r *PreviewTransfersReaderFS) ListByMintAddress(
	ctx context.Context,
	mintAddress string,
) ([]dto.PreviewTransferInfo, error) {
	if r == nil || r.client == nil || mintAddress == "" {
		return []dto.PreviewTransferInfo{}, nil
	}

	q := r.client.
		Collection(r.collectionName).
		Where("mintAddress", "==", mintAddress).
		OrderBy("createdAt", firestore.Desc)

	iter := q.Documents(ctx)
	defer iter.Stop()

	items := make([]dto.PreviewTransferInfo, 0, 8)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		items = append(items, decodePreviewTransferInfo(doc))
	}

	return items, nil
}

func decodePreviewTransferInfo(doc *firestore.DocumentSnapshot) dto.PreviewTransferInfo {
	var out dto.PreviewTransferInfo
	if doc == nil {
		return out
	}

	data := doc.Data()

	out.FromWalletAddress = firstNonEmptyString(
		asString(data["fromWalletAddress"]),
		asString(data["fromAddress"]),
	)
	out.ToWalletAddress = firstNonEmptyString(
		asString(data["toWalletAddress"]),
		asString(data["toAddress"]),
	)
	out.FromAvatarID = asString(data["fromAvatarId"])
	out.ToAvatarID = asString(data["toAvatarId"])
	out.FromBrandID = asString(data["fromBrandId"])
	out.ToBrandID = asString(data["toBrandId"])
	out.FromAvatarName = asString(data["fromAvatarName"])
	out.ToAvatarName = asString(data["toAvatarName"])
	out.FromBrandName = asString(data["fromBrandName"])
	out.ToBrandName = asString(data["toBrandName"])
	out.FromAvatarIcon = asString(data["fromAvatarIcon"])
	out.ToAvatarIcon = asString(data["toAvatarIcon"])
	out.FromBrandIcon = asString(data["fromBrandIcon"])
	out.ToBrandIcon = asString(data["toBrandIcon"])
	return out
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t.UTC()
	case *time.Time:
		if t == nil {
			return time.Time{}
		}
		return t.UTC()
	default:
		return time.Time{}
	}
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
