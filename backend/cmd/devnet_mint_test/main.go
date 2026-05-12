// backend/cmd/devnet_mint_test/main.go
package main

import (
	"context"
	"log"

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

	log.Println("[devnet-mint-test] MintDirect has been removed from TokenUsecase; no-op.")
}
