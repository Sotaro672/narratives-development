// backend/cmd/devnet_mint_test/main.go
package main

import (
	"context"
	"log"
	"os"

	"narratives/internal/application/usecase"
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

	// ====== ここをテスト値で埋める ======

	// 1. devnet 用の受取ウォレットアドレス
	//    - Phantom 等で devnet に切り替えたウォレットのアドレスを使う
	toAddress := os.Getenv("DEVNET_TEST_TO_ADDRESS")
	if toAddress == "" {
		log.Fatalf("DEVNET_TEST_TO_ADDRESS is not set")
	}

	// 2. MetadataURI
	//    - ひとまず GCS などに置いた簡易 JSON で OK
	//    - 例: {"name":"Narratives Devnet Test","symbol":"NDVT","description":"test"}
	metadataURI := os.Getenv("DEVNET_TEST_METADATA_URI")
	if metadataURI == "" {
		log.Fatalf("DEVNET_TEST_METADATA_URI is not set")
	}

	// 3. Name / Symbol
	//    - 本番では TokenBlueprint("1GpOEwaqHiBbs5BgWzi8") から取得
	//    - ここではテスト用に直書き or 後で repo 経由に差し替えても OK
	name := "Narratives Devnet Test"
	symbol := "NDVT"

	// 4. Amount
	//    - Mint テーブルの products が 2 件なので 2 にしてもいい
	amount := uint64(2) // len(products) = 2

	input := usecase.MintDirectInput{
		ToAddress:   toAddress,
		Amount:      amount,
		MetadataURI: metadataURI,
		Name:        name,
		Symbol:      symbol,
	}

	log.Printf("[devnet-mint-test] start mint: to=%s amount=%d uri=%s name=%s symbol=%s",
		input.ToAddress, input.Amount, input.MetadataURI, input.Name, input.Symbol,
	)

	result, err := container.TokenUC.MintDirect(ctx, input)
	if err != nil {
		log.Fatalf("MintDirect failed: %v", err)
	}

	log.Printf("[devnet-mint-test] mint success: signature=%s mintAddress=%s slot=%d",
		result.Signature, result.MintAddress, result.Slot,
	)
}
