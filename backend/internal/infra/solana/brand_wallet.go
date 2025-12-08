// backend/internal/infra/solana/brand_wallet_service.go
package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretspb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/blocto/solana-go-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	branddom "narratives/internal/domain/brand"
)

// BrandWalletService は Brand 用 Solana ウォレットの生成・管理を行う実装です。
// - 新規鍵ペアを生成
// - 秘密鍵を GCP Secret Manager に保存
// - 公開鍵(base58) を walletAddress として返却
type BrandWalletService struct {
	// デフォルトの GCP プロジェクト ID（未指定なら環境変数 GCP_PROJECT を使う）
	projectID string
}

// NewBrandWalletService は BrandWalletService のコンストラクタです。
// projectID が空文字の場合は、実行時に os.Getenv("GCP_PROJECT") を使用します。
func NewBrandWalletService(projectID string) *BrandWalletService {
	return &BrandWalletService{
		projectID: strings.TrimSpace(projectID),
	}
}

// resolveProjectID は DI 時に渡された projectID が空なら GCP_PROJECT 環境変数を使います。
func (s *BrandWalletService) resolveProjectID() (string, error) {
	if s.projectID != "" {
		return s.projectID, nil
	}
	if v := os.Getenv("GCP_PROJECT"); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v), nil
	}
	return "", fmt.Errorf("BrandWalletService: projectID is empty and GCP_PROJECT env is not set")
}

// OpenBrandWallet は brand ごとに専用の Solana ウォレットを新規発行し、
// 秘密鍵を Secret Manager に保存した上で、公開鍵を SolanaBrandWallet として返却します。
func (s *BrandWalletService) OpenBrandWallet(
	ctx context.Context,
	b branddom.Brand,
) (branddom.SolanaBrandWallet, error) {
	brandID := strings.TrimSpace(b.ID)
	if brandID == "" {
		return branddom.SolanaBrandWallet{}, fmt.Errorf("OpenBrandWallet: brand ID is empty")
	}

	projectID, err := s.resolveProjectID()
	if err != nil {
		return branddom.SolanaBrandWallet{}, err
	}

	// 1. Solana アカウント生成（types.NewAccount は 64byte の秘密鍵を持つ）
	acc := types.NewAccount()
	priv := acc.PrivateKey // []byte (len 64)
	pub := acc.PublicKey   // common.PublicKey

	// 2. 秘密鍵を JSON にシリアライズ
	//    mint_authority の decodeKeypairJSON と同じく [int,int,...] 形式にしておく。
	ints := make([]int, len(priv))
	for i, v := range priv {
		ints[i] = int(v)
	}
	payload, err := json.Marshal(ints)
	if err != nil {
		return branddom.SolanaBrandWallet{}, fmt.Errorf("OpenBrandWallet: marshal private key: %w", err)
	}

	// 3. Secret Manager に保存
	//    Secret ID は "brand-wallet-<brandID>" の形式とする（brandID は Firestore の docID 想定）。
	secretID := fmt.Sprintf("brand-wallet-%s", brandID)

	smClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		return branddom.SolanaBrandWallet{}, fmt.Errorf("OpenBrandWallet: secretmanager.NewClient: %w", err)
	}
	defer smClient.Close()

	parent := fmt.Sprintf("projects/%s", projectID)
	secretName := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretID)

	// 3-1. Secret が存在しない場合は作成
	_, err = smClient.GetSecret(ctx, &secretspb.GetSecretRequest{
		Name: secretName,
	})
	if err != nil {
		// NotFound の場合のみ作成。それ以外のエラーはそのまま返す。
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
				return branddom.SolanaBrandWallet{}, fmt.Errorf("OpenBrandWallet: CreateSecret %s: %w", secretID, cerr)
			}
		} else {
			return branddom.SolanaBrandWallet{}, fmt.Errorf("OpenBrandWallet: GetSecret %s: %w", secretID, err)
		}
	}

	// 3-2. 新しい Version として秘密鍵を書き込む
	addRes, err := smClient.AddSecretVersion(ctx, &secretspb.AddSecretVersionRequest{
		Parent: secretName,
		Payload: &secretspb.SecretPayload{
			Data: payload,
		},
	})
	if err != nil {
		return branddom.SolanaBrandWallet{}, fmt.Errorf("OpenBrandWallet: AddSecretVersion: %w", err)
	}

	// addRes.Name は "projects/<project>/secrets/<secret>/versions/<version>" 形式
	versionName := addRes.Name

	// 4. ドメインで扱う BrandWallet 情報を返却
	wallet := branddom.SolanaBrandWallet{
		BrandID:    brandID,
		Address:    pub.ToBase58(),
		SecretName: versionName, // もしくは secretName にして latest で読む設計でも可
	}

	return wallet, nil
}

// FreezeBrandWallet は将来的な拡張用（いまは未実装）。
// 例: Brand ウォレットを「凍結」扱いにするオンチェーン操作・メタ情報更新など。
func (s *BrandWalletService) FreezeBrandWallet(
	ctx context.Context,
	wallet branddom.SolanaBrandWallet,
) error {
	// TODO: 必要になったタイミングで実装（トークンの凍結/フラグ管理など）
	_ = ctx
	_ = wallet
	return nil
}

// DelegateTokenOperation は、マスターウォレット（MintAuthority）から
// ブランドウォレットへトークン運用権限を委譲する処理のためのフック。
// いまは未実装で、必要になったら Solana プログラム（Token / Metadata）との
// 連携ロジックを追加する。
func (s *BrandWalletService) DelegateTokenOperation(
	ctx context.Context,
	brandWallet branddom.SolanaBrandWallet,
	master branddom.MintAuthorityKey,
) error {
	// TODO: 必要になったタイミングで、マスターウォレットから Brand ウォレットへの
	//       delegate / authority 移譲ロジックを実装する。
	_ = ctx
	_ = brandWallet
	_ = master
	return nil
}
