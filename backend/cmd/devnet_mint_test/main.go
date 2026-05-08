package main

import (
	"context"
	"log"
	"reflect"

	consoleDI "narratives/internal/platform/di/console"
	shared "narratives/internal/platform/di/shared"
)

type closer interface {
	Close() error
}

func main() {
	ctx := context.Background()

	infra := &shared.Infra{}
	cont, err := consoleDI.NewContainer(ctx, infra)
	if err != nil {
		log.Fatalf("failed to init console container: %v", err)
	}

	if c, ok := any(cont).(closer); ok {
		defer func() {
			if err := c.Close(); err != nil {
				log.Printf("[devnet-mint-test] WARN: container close error: %v", err)
			}
		}()
	}

	deps := cont.RouterDeps()

	if !hasNonNilFieldBestEffort(deps, "TokenUC", "TokenUsecase", "TokenBlueprintUC", "TokenBlueprintUsecase", "MintUC", "MintUsecase") &&
		!hasNonNilFieldBestEffort(cont, "TokenUC", "TokenUsecase", "TokenBlueprintUC", "TokenBlueprintUsecase", "MintUC", "MintUsecase") {
		log.Printf("[devnet-mint-test] WARN: token/mint related usecase field not found or nil. (field names may differ)")
	} else {
		log.Printf("[devnet-mint-test] token/mint related usecase looks present (best-effort check)")
	}

	log.Println("[devnet-mint-test] MintDirect has been removed from TokenUsecase; no-op.")
}

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
		if n == "" {
			continue
		}
		f := rv.FieldByName(n)
		if !f.IsValid() || !f.CanInterface() {
			continue
		}

		switch f.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
			if f.IsNil() {
				continue
			}
			return true
		default:
			return true
		}
	}

	return false
}
