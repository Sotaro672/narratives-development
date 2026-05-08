// backend/internal/adapters/out/firestore/resolved_token_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	appusecase "narratives/internal/application/usecase"
)

// ResolvedTokenRepositoryFS は ResolveTokenByMintAddressWithBrandNameResult の Firestore キャッシュ実装です。
//
// ✅ Collection design:
// - collection: wallets
// - docId: avatarId
// - subcollection: resolvedTokens
// - docId: mintAddress
// - fields: resolved token DTO (mintAddress is stored for query convenience; docId is the source of truth)
type ResolvedTokenRepositoryFS struct {
	Client *firestore.Client
}

// NewResolvedTokenRepositoryFS は ResolvedTokenRepositoryFS を生成します。
func NewResolvedTokenRepositoryFS(client *firestore.Client) *ResolvedTokenRepositoryFS {
	return &ResolvedTokenRepositoryFS{Client: client}
}

func (r *ResolvedTokenRepositoryFS) walletCol() *firestore.CollectionRef {
	return r.Client.Collection("wallets")
}

func (r *ResolvedTokenRepositoryFS) resolvedTokensCol(avatarID string) *firestore.CollectionRef {
	return r.walletCol().Doc(avatarID).Collection("resolvedTokens")
}

var (
	ErrResolvedTokenRepoInvalidAvatarID    = errors.New("resolved_token_repository_fs: invalid avatarId")
	ErrResolvedTokenRepoInvalidMintAddress = errors.New("resolved_token_repository_fs: invalid mintAddress")
	ErrResolvedTokenRepoClientNil          = errors.New("resolved_token_repository_fs: firestore client is nil")
)

// Firestore 上のスキーマ用 DTO
type resolvedTokenDoc struct {
	ProductID          string `firestore:"productId"`
	BrandID            string `firestore:"brandId"`
	BrandName          string `firestore:"brandName"`
	MetadataURI        string `firestore:"metadataUri"`
	MintAddress        string `firestore:"mintAddress"`
	ProductBlueprintID string `firestore:"productBlueprintId"`
	ProductName        string `firestore:"productName"`

	TokenBlueprintID string                    `firestore:"tokenBlueprintId"`
	TokenContents    []resolvedTokenContentDoc `firestore:"tokenContentsFiles"`

	// キャッシュ更新時刻（任意）
	ResolvedAt time.Time `firestore:"resolvedAt"`
	// スキーマ互換のためのバージョン（任意）
	SourceVersion int `firestore:"sourceVersion"`
}

type resolvedTokenContentDoc struct {
	// ✅ stable keys（推奨：Firestoreに保存してOK）
	FileName   string `firestore:"fileName"`
	Bucket     string `firestore:"bucket"`
	ObjectPath string `firestore:"objectPath"`
	Type       string `firestore:"type"`
	PublicURI  string `firestore:"publicUri"`

	// ✅ 期限付き（原則 Firestore には保存しない想定だが、互換のため許容）
	ViewURI       string    `firestore:"viewUri"`
	ViewExpiresAt time.Time `firestore:"viewExpiresAt"`
}

func toResolvedTokenDoc(src appusecase.ResolveTokenByMintAddressWithBrandNameResult, now time.Time) resolvedTokenDoc {
	at := now
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}

	files := make([]resolvedTokenContentDoc, 0, len(src.TokenContentsFiles))
	for _, f := range src.TokenContentsFiles {
		var exp time.Time
		if f.ViewExpiresAt != nil {
			exp = f.ViewExpiresAt.UTC()
		}

		files = append(files, resolvedTokenContentDoc{
			FileName:      f.FileName,
			Bucket:        f.Bucket,
			ObjectPath:    f.ObjectPath,
			Type:          f.Type,
			PublicURI:     f.PublicURI,
			ViewURI:       f.ViewURI,
			ViewExpiresAt: exp,
		})
	}

	return resolvedTokenDoc{
		ProductID:          src.ProductID,
		BrandID:            src.BrandID,
		BrandName:          src.BrandName,
		MetadataURI:        src.MetadataURI,
		MintAddress:        src.MintAddress,
		ProductBlueprintID: src.ProductBlueprintID,
		ProductName:        src.ProductName,

		TokenBlueprintID: src.TokenBlueprintID,
		TokenContents:    files,

		ResolvedAt:    at,
		SourceVersion: 1,
	}
}

func fromResolvedTokenDoc(d resolvedTokenDoc) appusecase.ResolveTokenByMintAddressWithBrandNameResult {
	files := make([]appusecase.SignedTokenContentFile, 0, len(d.TokenContents))
	for _, f := range d.TokenContents {
		var exp *time.Time
		if !f.ViewExpiresAt.IsZero() {
			t := f.ViewExpiresAt.UTC()
			exp = &t
		}

		files = append(files, appusecase.SignedTokenContentFile{
			FileName:   f.FileName,
			Bucket:     f.Bucket,
			ObjectPath: f.ObjectPath,
			Type:       f.Type,
			PublicURI:  f.PublicURI,
			ViewURI:    f.ViewURI,
			// nil の場合は省略
			ViewExpiresAt: exp,
		})
	}

	return appusecase.ResolveTokenByMintAddressWithBrandNameResult{
		ProductID:          d.ProductID,
		BrandID:            d.BrandID,
		BrandName:          d.BrandName,
		MetadataURI:        d.MetadataURI,
		MintAddress:        d.MintAddress,
		ProductBlueprintID: d.ProductBlueprintID,
		ProductName:        d.ProductName,
		TokenBlueprintID:   d.TokenBlueprintID,
		TokenContentsFiles: files,
	}
}

// GetByAvatarIDAndMint は wallets/{avatarId}/resolvedTokens/{mintAddress} を 1 件取得します。
func (r *ResolvedTokenRepositoryFS) GetByAvatarIDAndMint(
	ctx context.Context,
	avatarID string,
	mintAddress string,
) (appusecase.ResolveTokenByMintAddressWithBrandNameResult, error) {

	if r == nil || r.Client == nil {
		return appusecase.ResolveTokenByMintAddressWithBrandNameResult{}, ErrResolvedTokenRepoClientNil
	}
	if avatarID == "" {
		return appusecase.ResolveTokenByMintAddressWithBrandNameResult{}, ErrResolvedTokenRepoInvalidAvatarID
	}
	if mintAddress == "" {
		return appusecase.ResolveTokenByMintAddressWithBrandNameResult{}, ErrResolvedTokenRepoInvalidMintAddress
	}

	snap, err := r.resolvedTokensCol(avatarID).Doc(mintAddress).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return appusecase.ResolveTokenByMintAddressWithBrandNameResult{}, errors.New("resolved_token_repository_fs: not found")
		}
		return appusecase.ResolveTokenByMintAddressWithBrandNameResult{}, err
	}

	var d resolvedTokenDoc
	if err := snap.DataTo(&d); err != nil {
		return appusecase.ResolveTokenByMintAddressWithBrandNameResult{}, err
	}

	// docId が正なので、フィールドが空でも docId を優先して補完
	if d.MintAddress == "" {
		d.MintAddress = mintAddress
	}

	return fromResolvedTokenDoc(d), nil
}

// ListByAvatarID は wallets/{avatarId}/resolvedTokens を全件取得します（件数が多い場合はページングを検討）。
func (r *ResolvedTokenRepositoryFS) ListByAvatarID(
	ctx context.Context,
	avatarID string,
) ([]appusecase.ResolveTokenByMintAddressWithBrandNameResult, error) {

	if r == nil || r.Client == nil {
		return nil, ErrResolvedTokenRepoClientNil
	}
	if avatarID == "" {
		return nil, ErrResolvedTokenRepoInvalidAvatarID
	}

	it := r.resolvedTokensCol(avatarID).Documents(ctx)
	defer it.Stop()

	out := make([]appusecase.ResolveTokenByMintAddressWithBrandNameResult, 0)
	for {
		doc, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}

		var d resolvedTokenDoc
		if err := doc.DataTo(&d); err != nil {
			return nil, err
		}

		// docId を優先して補完
		if d.MintAddress == "" {
			d.MintAddress = doc.Ref.ID
		}

		out = append(out, fromResolvedTokenDoc(d))
	}

	return out, nil
}

// Upsert は wallets/{avatarId}/resolvedTokens/{mintAddress} を upsert します。
func (r *ResolvedTokenRepositoryFS) Upsert(
	ctx context.Context,
	avatarID string,
	mintAddress string,
	res appusecase.ResolveTokenByMintAddressWithBrandNameResult,
	now time.Time,
) error {

	if r == nil || r.Client == nil {
		return ErrResolvedTokenRepoClientNil
	}
	if avatarID == "" {
		return ErrResolvedTokenRepoInvalidAvatarID
	}
	if mintAddress == "" {
		return ErrResolvedTokenRepoInvalidMintAddress
	}

	// docId = mintAddress を正とする（payload と不一致でも docId 優先）
	res.MintAddress = mintAddress

	d := toResolvedTokenDoc(res, now)
	_, err := r.resolvedTokensCol(avatarID).Doc(mintAddress).Set(ctx, d)
	return err
}

// DeleteByAvatarIDAndMint は wallets/{avatarId}/resolvedTokens/{mintAddress} を削除します。
func (r *ResolvedTokenRepositoryFS) DeleteByAvatarIDAndMint(
	ctx context.Context,
	avatarID string,
	mintAddress string,
) error {

	if r == nil || r.Client == nil {
		return ErrResolvedTokenRepoClientNil
	}
	if avatarID == "" {
		return ErrResolvedTokenRepoInvalidAvatarID
	}
	if mintAddress == "" {
		return ErrResolvedTokenRepoInvalidMintAddress
	}

	_, err := r.resolvedTokensCol(avatarID).Doc(mintAddress).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return err
	}
	return nil
}

// DeleteAllByAvatarID は wallets/{avatarId}/resolvedTokens を全削除します（管理用）。
// 注意：Firestore はコレクションを一括削除できないため、全件列挙して削除します。
// 件数が多い場合は batch / BulkWriter 等に置き換えてください。
func (r *ResolvedTokenRepositoryFS) DeleteAllByAvatarID(ctx context.Context, avatarID string) error {
	if r == nil || r.Client == nil {
		return ErrResolvedTokenRepoClientNil
	}
	if avatarID == "" {
		return ErrResolvedTokenRepoInvalidAvatarID
	}

	it := r.resolvedTokensCol(avatarID).Documents(ctx)
	defer it.Stop()

	for {
		doc, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return err
		}
		_, err = doc.Ref.Delete(ctx)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
	}
	return nil
}
