// generate_narratives_mint_wallet.go
//
// Narratives 専用の Solana ミント権限ウォレットを生成する小さなツールです。
// - Solana 互換の ed25519 keypair を生成
// - 公開鍵を base58 文字列として表示（これが mintAuthorityAddress）
// - 秘密鍵を Solana CLI 互換の JSON 配列としてファイルに保存します。
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// Solana / Bitcoin と同じ Base58 alphabet
const b58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// Base58 エンコード（外部ライブラリなし版）
func base58Encode(input []byte) string {
	if len(input) == 0 {
		return ""
	}

	// 大まかに 2倍サイズのバッファを用意
	size := len(input) * 2
	digits := make([]byte, 0, size)

	// 数値として base58 に変換
	for _, b := range input {
		carry := int(b)
		for i := 0; i < len(digits); i++ {
			carry += int(digits[i]) << 8
			digits[i] = byte(carry % 58)
			carry /= 58
		}
		for carry > 0 {
			digits = append(digits, byte(carry%58))
			carry /= 58
		}
	}

	// 先頭の 0 バイトは Base58 では '1' として表現
	zeros := 0
	for zeros < len(input) && input[zeros] == 0 {
		zeros++
	}

	result := make([]byte, zeros+len(digits))
	for i := 0; i < zeros; i++ {
		result[i] = '1'
	}
	for i := 0; i < len(digits); i++ {
		result[zeros+i] = b58Alphabet[digits[len(digits)-1-i]]
	}
	return string(result)
}

func main() {
	// 1. ed25519 keypair を生成
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate ed25519 keypair: %v", err)
	}

	// 2. 公開鍵を Base58 エンコード（これが Solana のアドレス）
	pubKeyBase58 := base58Encode(pub)

	// 3. 秘密鍵を Solana CLI と互換の JSON 配列形式に変換
	//    Solana の keypair ファイルは [64 byte] の secret key をそのまま配列にしたものです。
	secret := make([]int, len(priv))
	for i, b := range priv {
		secret[i] = int(b)
	}

	data, err := json.MarshalIndent(secret, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal secret key json: %v", err)
	}

	const fileName = "narratives-mint-authority.json"

	// 4. ファイルとして保存（上書き注意）
	if err := os.WriteFile(fileName, data, 0o600); err != nil {
		log.Fatalf("failed to write %s: %v", fileName, err)
	}

	fmt.Println("============================================")
	fmt.Println("✅ Narratives Mint Authority Wallet generated")
	fmt.Println("============================================")
	fmt.Printf("Public Key (mintAuthorityAddress):\n  %s\n\n", pubKeyBase58)
	fmt.Printf("Secret key file (Solana-compatible JSON):\n  %s\n\n", fileName)
	fmt.Println("⚠ IMPORTANT:")
	fmt.Println("  - この JSON ファイルは Git に絶対にコミットしないでください。")
	fmt.Println("  - 後で GCP Secret Manager に登録し、ローカルのコピーは安全な場所に退避してください。")
}
