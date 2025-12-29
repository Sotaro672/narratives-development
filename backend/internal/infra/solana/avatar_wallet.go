// backend/internal/infra/solana/avatar_wallet.go
package solana

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretspb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	avatardom "narratives/internal/domain/avatar"
)

// AvatarWalletService は Avatar 用 Solana ウォレットの生成・管理を行う実装です。
// - 新規鍵ペアを生成
// - 秘密鍵を GCP Secret Manager に保存
// - 公開鍵(base58) を walletAddress として返却
//
// ✅ IMPORTANT (Idempotent):
//   - すでに Secret Manager に秘密鍵が存在する場合は「再生成しない」。
//     latest を読み出して公開鍵を復元し、同じ wallet を返す。
//   - これによりリトライ / 二重実行でも wallet が変わらず安全になる。
//
// NOTE:
// - BrandWalletService と同様に「鍵の実体（秘密鍵）は Secret Manager、公開鍵だけを Firestore に持たせる」前提。
// - Secret ID は "avatar-wallet-%s"（%s=avatarID）で決定的に生成します。
type AvatarWalletService struct {
	// デフォルトの GCP プロジェクト ID（未指定なら環境変数 GCP_PROJECT を使う）
	projectID string
}

// NewAvatarWalletService は AvatarWalletService のコンストラクタです。
// projectID が空文字の場合は、実行時に os.Getenv("GCP_PROJECT") を使用します。
func NewAvatarWalletService(projectID string) *AvatarWalletService {
	return &AvatarWalletService{
		projectID: strings.TrimSpace(projectID),
	}
}

// resolveProjectID は DI 時に渡された projectID が空なら GCP_PROJECT 環境変数を使います。
func (s *AvatarWalletService) resolveProjectID() (string, error) {
	if s.projectID != "" {
		return s.projectID, nil
	}
	if v := os.Getenv("GCP_PROJECT"); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v), nil
	}
	return "", fmt.Errorf("AvatarWalletService: projectID is empty and GCP_PROJECT env is not set")
}

// OpenAvatarWallet は avatar ごとに専用の Solana ウォレットを発行します。
// ただし、既に Secret Manager に秘密鍵が保存済みの場合はそれを再利用し、wallet を再生成しません。
//
// avatarID は Firestore の docID（Avatar.ID）想定。
// userId/firebaseUid は鍵管理に不要なので参照しません。
func (s *AvatarWalletService) OpenAvatarWallet(
	ctx context.Context,
	avatarID string,
) (avatardom.SolanaAvatarWallet, error) {
	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return avatardom.SolanaAvatarWallet{}, fmt.Errorf("OpenAvatarWallet: avatarID is empty")
	}

	projectID, err := s.resolveProjectID()
	if err != nil {
		return avatardom.SolanaAvatarWallet{}, err
	}

	// Secret ID は決定的（idempotent）
	secretID := fmt.Sprintf("avatar-wallet-%s", aid)
	secretName := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretID)
	latestVersionName := fmt.Sprintf("%s/versions/latest", secretName)
	parent := fmt.Sprintf("projects/%s", projectID)

	smClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		return avatardom.SolanaAvatarWallet{}, fmt.Errorf("OpenAvatarWallet: secretmanager.NewClient: %w", err)
	}
	defer smClient.Close()

	// ============================================================
	// 0) 既存 wallet があるならそれを復元して返す（冪等）
	// ============================================================
	if res, aerr := smClient.AccessSecretVersion(ctx, &secretspb.AccessSecretVersionRequest{
		Name: latestVersionName,
	}); aerr == nil {
		addr, derr := deriveAddressFromSecretPayload(res.GetPayload().GetData())
		if derr != nil {
			return avatardom.SolanaAvatarWallet{}, fmt.Errorf("OpenAvatarWallet: deriveAddressFromSecretPayload: %w", derr)
		}
		return avatardom.SolanaAvatarWallet{
			AvatarID:   aid,
			Address:    addr,
			SecretName: res.GetName(), // "projects/<p>/secrets/<s>/versions/<v>" が返る（latest の実体）
		}, nil
	} else {
		// NotFound のみ「新規作成」へ。それ以外はエラー。
		if status.Code(aerr) != codes.NotFound {
			return avatardom.SolanaAvatarWallet{}, fmt.Errorf("OpenAvatarWallet: AccessSecretVersion latest: %w", aerr)
		}
	}

	// ============================================================
	// 1) Secret が存在しない場合は作成（NotFound のみ）
	// ============================================================
	_, err = smClient.GetSecret(ctx, &secretspb.GetSecretRequest{
		Name: secretName,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			_, cerr := smClient.CreateSecret(ctx, &secretspb.CreateSecretRequest{
				Parent:   parent,
				SecretId: secretID,
				Secret: &secretspb.Secret{
					Replication: &secretspb.Replication{
						Replication: &secretspb.Replication_Automatic_{
							Automatic: &secretspb.Replication_Automatic{},
						},
					},
				},
			})
			if cerr != nil {
				return avatardom.SolanaAvatarWallet{}, fmt.Errorf("OpenAvatarWallet: CreateSecret %s: %w", secretID, cerr)
			}
		} else {
			return avatardom.SolanaAvatarWallet{}, fmt.Errorf("OpenAvatarWallet: GetSecret %s: %w", secretID, err)
		}
	}

	// ============================================================
	// 2) 新規鍵ペア生成 → Secret Version 追加
	// ============================================================
	acc := types.NewAccount()
	priv := acc.PrivateKey // []byte (len 64)
	pub := acc.PublicKey

	// mint_authority の decodeKeypairJSON と同じく [int,int,...] 形式
	ints := make([]int, len(priv))
	for i, v := range priv {
		ints[i] = int(v)
	}
	payload, err := json.Marshal(ints)
	if err != nil {
		return avatardom.SolanaAvatarWallet{}, fmt.Errorf("OpenAvatarWallet: marshal private key: %w", err)
	}

	addRes, err := smClient.AddSecretVersion(ctx, &secretspb.AddSecretVersionRequest{
		Parent: secretName,
		Payload: &secretspb.SecretPayload{
			Data: payload,
		},
	})
	if err != nil {
		return avatardom.SolanaAvatarWallet{}, fmt.Errorf("OpenAvatarWallet: AddSecretVersion: %w", err)
	}

	return avatardom.SolanaAvatarWallet{
		AvatarID:   aid,
		Address:    pub.ToBase58(),
		SecretName: addRes.GetName(), // "projects/<project>/secrets/<secret>/versions/<version>"
	}, nil
}

// ============================================================
// ✅ Usecase adapter (統合)
// - application/usecase 側が期待する `OpenAvatarWallet(ctx, Avatar)` 形に合わせる
// - infra 実装の `OpenAvatarWallet(ctx, avatarID)` を呼ぶだけの薄いアダプタ
// ============================================================

// AvatarWalletUsecaseAdapter は application/usecase.AvatarWalletService を満たすための薄いアダプタ。
// - usecase から渡される Avatar から ID を取り出し、infra の OpenAvatarWallet(avatarID) を呼ぶ。
type AvatarWalletUsecaseAdapter struct {
	svc *AvatarWalletService
}

func NewAvatarWalletUsecaseAdapter(svc *AvatarWalletService) *AvatarWalletUsecaseAdapter {
	return &AvatarWalletUsecaseAdapter{svc: svc}
}

// OpenAvatarWallet implements usecase.AvatarWalletService.
func (a *AvatarWalletUsecaseAdapter) OpenAvatarWallet(
	ctx context.Context,
	av avatardom.Avatar,
) (avatardom.SolanaAvatarWallet, error) {
	if a == nil || a.svc == nil {
		return avatardom.SolanaAvatarWallet{}, fmt.Errorf("AvatarWalletUsecaseAdapter: service is nil")
	}

	id := strings.TrimSpace(av.ID)
	if id == "" {
		return avatardom.SolanaAvatarWallet{}, fmt.Errorf("AvatarWalletUsecaseAdapter: avatar.ID is empty")
	}

	return a.svc.OpenAvatarWallet(ctx, id)
}

// deriveAddressFromSecretPayload は Secret Manager の payload（JSON: [int,int,...]）から
// Solana の base58 public key を復元します。
func deriveAddressFromSecretPayload(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty secret payload")
	}

	var ints []int
	if err := json.Unmarshal(data, &ints); err != nil {
		return "", fmt.Errorf("unmarshal payload json: %w", err)
	}
	if len(ints) != 64 {
		return "", fmt.Errorf("invalid private key length: got=%d want=64", len(ints))
	}

	priv := make([]byte, 64)
	for i, v := range ints {
		if v < 0 || v > 255 {
			return "", fmt.Errorf("invalid byte value at index %d: %d", i, v)
		}
		priv[i] = byte(v)
	}

	// ed25519 private key (64 bytes) -> public key (32 bytes)
	pk := ed25519.PrivateKey(priv).Public()
	pubEd, ok := pk.(ed25519.PublicKey)
	if !ok || len(pubEd) != ed25519.PublicKeySize {
		return "", fmt.Errorf("invalid derived public key")
	}

	pub := common.PublicKeyFromBytes(pubEd)
	return pub.ToBase58(), nil
}

// FreezeAvatarWallet は将来的な拡張用（いまは未実装）。
// 例: Avatar ウォレットを「凍結」扱いにするオンチェーン操作・メタ情報更新など。
func (s *AvatarWalletService) FreezeAvatarWallet(
	ctx context.Context,
	wallet avatardom.SolanaAvatarWallet,
) error {
	// TODO: 必要になったタイミングで実装（トークンの凍結/フラグ管理など）
	_ = ctx
	_ = wallet
	return nil
}
