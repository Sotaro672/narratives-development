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

	fromWalletAddress, _ := data["fromWalletAddress"].(string)
	fromAddress, _ := data["fromAddress"].(string)
	toWalletAddress, _ := data["toWalletAddress"].(string)
	toAddress, _ := data["toAddress"].(string)

	out.FromWalletAddress = firstNonEmptyString(
		fromWalletAddress,
		fromAddress,
	)
	out.ToWalletAddress = firstNonEmptyString(
		toWalletAddress,
		toAddress,
	)
	out.FromAvatarID, _ = data["fromAvatarId"].(string)
	out.ToAvatarID, _ = data["toAvatarId"].(string)
	out.FromBrandID, _ = data["fromBrandId"].(string)
	out.ToBrandID, _ = data["toBrandId"].(string)
	out.FromAvatarName, _ = data["fromAvatarName"].(string)
	out.ToAvatarName, _ = data["toAvatarName"].(string)
	out.FromBrandName, _ = data["fromBrandName"].(string)
	out.ToBrandName, _ = data["toBrandName"].(string)
	out.FromAvatarIcon, _ = data["fromAvatarIcon"].(string)
	out.ToAvatarIcon, _ = data["toAvatarIcon"].(string)
	out.FromBrandIcon, _ = data["fromBrandIcon"].(string)
	out.ToBrandIcon, _ = data["toBrandIcon"].(string)

	return out
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
