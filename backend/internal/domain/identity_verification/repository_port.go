// backend/internal/domain/identity_verification/repository_port.go
package identityverification

import (
	"context"
	"errors"
)

// ============================================================
// Repository errors
// ============================================================

var (
	ErrNotFound = errors.New(
		"identity verification: not found",
	)
)

// ============================================================
// Repository
// ============================================================

// Repository はユーザー本人確認情報の永続化契約です。
//
// Firestore 実装では以下の構造を前提とします:
//
//	identityVerifications/{userId}
//
// userId は Firebase Auth UID です。
//
// 本 Repository は個人番号、マイナンバーカード画像、暗証番号、
// IC チップから取得した生データなどを扱いません。
// 永続化するのは本人確認状態と、外部本人確認サービスを参照するための
// 最小限のメタデータのみです。
type Repository interface {
	// GetByUserID は userId に対応する本人確認情報を取得します。
	//
	// 対象ドキュメントが存在しない場合は ErrNotFound を返します。
	//
	// Application 層では ErrNotFound を
	// 「本人確認未登録 = unverified」として扱うことができます。
	GetByUserID(
		ctx context.Context,
		userID string,
	) (IdentityVerification, error)

	// Upsert は userId に対応する本人確認情報を作成または更新します。
	//
	// Firestore document ID は verification.UserID とし、
	// verification.UserID と異なる識別子をクライアントから
	// 指定できる実装にしてはいけません。
	//
	// 同一内容を繰り返し保存しても成功する冪等な操作とします。
	//
	// IdentityVerification.Validate() を通過できない値は
	// 永続化してはいけません。
	//
	// verified -> unverified のような本人確認状態の巻き戻しを
	// Repository 自身が生成してはいけません。
	// 状態遷移の判断は Domain / Application 層を正とします。
	Upsert(
		ctx context.Context,
		verification IdentityVerification,
	) (IdentityVerification, error)
}
