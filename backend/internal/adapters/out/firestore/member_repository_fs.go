// backend/internal/adapters/out/firestore/member_repository_fs.go
package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	memdom "narratives/internal/domain/member"
)

// MemberRepositoryFS は Firestore に保存された Member データを扱うリポジトリ実装です。
// Firestore の "members" コレクションを使用します。
type MemberRepositoryFS struct {
	Client *firestore.Client
}

// NewMemberRepositoryFS は Firestore クライアントを受け取りリポジトリを初期化します。
func NewMemberRepositoryFS(client *firestore.Client) *MemberRepositoryFS {
	return &MemberRepositoryFS{Client: client}
}

// ========================================
// GetByID
// ========================================
// Firestore のドキュメント ID または "id" フィールドに対応。
func (r *MemberRepositoryFS) GetByID(ctx context.Context, id string) (memdom.Member, error) {
	doc, err := r.Client.Collection("members").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return memdom.Member{}, memdom.ErrNotFound
		}
		return memdom.Member{}, err
	}

	var m memdom.Member
	if err := doc.DataTo(&m); err != nil {
		return memdom.Member{}, err
	}

	// FirestoreのDocIDをIDに反映（念のため）
	if m.ID == "" {
		m.ID = doc.Ref.ID
	}

	return m, nil
}

// ========================================
// Exists
// ========================================
// Firestore上に対象ドキュメントが存在するかをチェック。
func (r *MemberRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	_, err := r.Client.Collection("members").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ========================================
// Create
// ========================================
// 指定された Member を Firestore に新規作成します。
// id が空の場合は Firestore 側で自動採番します。
func (r *MemberRepositoryFS) Create(ctx context.Context, m memdom.Member) (memdom.Member, error) {
	ref := r.Client.Collection("members").Doc(m.ID)
	if m.ID == "" {
		ref = r.Client.Collection("members").NewDoc()
		m.ID = ref.ID
	}

	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = &now

	_, err := ref.Set(ctx, m)
	if err != nil {
		return memdom.Member{}, err
	}

	return m, nil
}

// ========================================
// Save (upsert 相当)
// ========================================
// 既存ドキュメントが存在すれば上書き、なければ新規作成します。
func (r *MemberRepositoryFS) Save(ctx context.Context, m memdom.Member, _ *memdom.SaveOptions) (memdom.Member, error) {
	if m.ID == "" {
		ref := r.Client.Collection("members").NewDoc()
		m.ID = ref.ID
	}

	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = &now

	ref := r.Client.Collection("members").Doc(m.ID)
	_, err := ref.Set(ctx, m)
	if err != nil {
		return memdom.Member{}, err
	}

	return m, nil
}

// ========================================
// Delete
// ========================================
// 指定された ID のメンバーを削除します。
// 存在しない場合は ErrNotFound を返します。
func (r *MemberRepositoryFS) Delete(ctx context.Context, id string) error {
	ref := r.Client.Collection("members").Doc(id)
	_, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return memdom.ErrNotFound
		}
		return err
	}

	_, err = ref.Delete(ctx)
	if err != nil {
		return err
	}

	return nil
}
