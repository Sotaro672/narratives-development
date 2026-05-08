// backend/internal/adapters/out/firestore/system_config_repository_fs.go
package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	syscfg "narratives/internal/domain/systemconfig"
)

const (
	systemConfigCollection = "system_config"
	mintAuthorityDocID     = "mintAuthority"
	mintAuthorityField     = "mintAuthorityPubkey"
)

type SystemConfigRepositoryFS struct {
	Client *firestore.Client
}

func NewSystemConfigRepositoryFS(client *firestore.Client) *SystemConfigRepositoryFS {
	return &SystemConfigRepositoryFS{Client: client}
}

func (r *SystemConfigRepositoryFS) GetMintAuthorityPubkey(ctx context.Context) (string, error) {
	docRef := r.Client.Collection(systemConfigCollection).Doc(mintAuthorityDocID)

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", syscfg.ErrMintAuthorityNotConfigured
		}
		return "", fmt.Errorf("systemconfig: failed to get mint authority config: %w", err)
	}

	val, err := snap.DataAt(mintAuthorityField)
	if err != nil {
		return "", fmt.Errorf("systemconfig: failed to read field %q: %w", mintAuthorityField, err)
	}

	pubkey, ok := val.(string)
	if !ok || pubkey == "" {
		return "", fmt.Errorf("systemconfig: invalid mintAuthorityPubkey value")
	}

	return pubkey, nil
}
