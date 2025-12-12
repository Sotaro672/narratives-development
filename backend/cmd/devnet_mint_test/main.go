// backend/cmd/devnet_mint_test/main.go
package main

import (
	"context"
	"log"

	"narratives/internal/platform/di"
)

func main() {
	ctx := context.Background()

	// コンテナを初期化（Cloud Run と同じ Config / Secret Manager 設定を利用）
	container, err := di.NewContainer(ctx)
	if err != nil {
		log.Fatalf("failed to init container: %v", err)
	}
	defer container.Close()

	if container.TokenUC == nil {
		log.Fatalf("TokenUsecase is nil (mint authority key may not be loaded)")
	}

	// MintDirect はユースケースから削除されたため、このコマンドは現在 no-op です。
	// 必要であれば、MintRequest を経由した MintFromMintRequest を使う
	// 別のテストコマンドを用意してください。
	log.Println("[devnet-mint-test] MintDirect has been removed from TokenUsecase; no-op.")
}
