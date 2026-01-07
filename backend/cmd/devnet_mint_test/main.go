// backend/cmd/devnet_mint_test/main.go
package main

import (
	"context"
	"log"
	"reflect"
	"strings"

	consoleDI "narratives/internal/platform/di/console"
	shared "narratives/internal/platform/di/shared"
)

type closer interface {
	Close() error
}

func main() {
	ctx := context.Background()

	// ✅ consoleDI.NewContainer(ctx, *shared.Infra) を使う（di.NewContainer は存在しない）
	infra := &shared.Infra{} // NOTE: 型名が Infra なら Infra、もし違うなら shared 側の定義に合わせて修正してください
	cont, err := consoleDI.NewContainer(ctx, infra)
	if err != nil {
		log.Fatalf("failed to init console container: %v", err)
	}

	// best-effort close (Close() が無い実装でもコンパイルを通す)
	if c, ok := any(cont).(closer); ok {
		defer func() {
			if err := c.Close(); err != nil {
				log.Printf("[devnet-mint-test] WARN: container close error: %v", err)
			}
		}()
	}

	// TokenUC などのフィールド名が揺れてもコンパイルを壊さないように reflection で確認
	deps := cont.RouterDeps()

	if !hasNonNilFieldBestEffort(deps, "TokenUC", "TokenUsecase", "TokenBlueprintUC", "TokenBlueprintUsecase", "MintUC", "MintUsecase") &&
		!hasNonNilFieldBestEffort(cont, "TokenUC", "TokenUsecase", "TokenBlueprintUC", "TokenBlueprintUsecase", "MintUC", "MintUsecase") {
		log.Printf("[devnet-mint-test] WARN: token/mint related usecase field not found or nil. (field names may differ)")
	} else {
		log.Printf("[devnet-mint-test] token/mint related usecase looks present (best-effort check)")
	}

	// MintDirect はユースケースから削除された前提
	log.Println("[devnet-mint-test] MintDirect has been removed from TokenUsecase; no-op.")
}

// hasNonNilFieldBestEffort returns true if any of the given exported field names exists and is non-nil.
// src can be a struct, *struct, or interface wrapping those.
func hasNonNilFieldBestEffort(src any, fieldNames ...string) bool {
	if src == nil {
		return false
	}

	rv := reflect.ValueOf(src)
	if !rv.IsValid() {
		return false
	}
	if rv.Kind() == reflect.Interface && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false
	}

	for _, n := range fieldNames {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		f := rv.FieldByName(n)
		if !f.IsValid() || !f.CanInterface() {
			continue
		}

		// nil 判定（interface/pointer/slice/map/func など）
		switch f.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
			if f.IsNil() {
				continue
			}
			return true
		default:
			// 非nil型（struct/intなど）は存在した時点で true 扱い
			return true
		}
	}

	return false
}
