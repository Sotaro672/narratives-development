// backend/internal/adapters/out/firestore/mall/avatar_name_icon_reader_fs.go
package mall

import (
	"context"

	"cloud.google.com/go/firestore"

	outfs "narratives/internal/adapters/out/firestore"
)

type AvatarNameIconReaderFS struct {
	repo *outfs.AvatarRepositoryFS
}

func NewAvatarNameIconReaderFS(client *firestore.Client) *AvatarNameIconReaderFS {
	return &AvatarNameIconReaderFS{
		repo: outfs.NewAvatarRepositoryFS(client),
	}
}

func (r *AvatarNameIconReaderFS) TryGetAvatarNameIcon(ctx context.Context, avatarID string) (string, string, bool, error) {
	if r == nil || r.repo == nil || avatarID == "" {
		return "", "", false, nil
	}

	a, err := r.repo.GetByID(ctx, avatarID)
	if err != nil {
		// avatar domain に ErrNotFound が無い前提なので、
		// ここでは preview 用 reader として「取れなければ表示しない」挙動に寄せる。
		return "", "", false, nil
	}

	name := a.AvatarName
	if name == "" {
		return "", "", false, nil
	}

	icon := ""
	if a.AvatarIcon != nil {
		icon = *a.AvatarIcon
	}

	return name, icon, true, nil
}
