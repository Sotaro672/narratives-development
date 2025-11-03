// backend/internal/infra/firestore/client.go
package firestoreinfra

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

// ClientWrapper は Firestore クライアントとその設定をラップします。
type ClientWrapper struct {
	Client    *firestore.Client
	ProjectID string
}

// NewClient は Firestore クライアントを初期化します。
// credentialsFile が空文字の場合、ADC(Application Default Credentials)を使用します。
func NewClient(ctx context.Context, projectID string, credentialsFile string) (*ClientWrapper, error) {
	var (
		client *firestore.Client
		err    error
	)
	if credentialsFile != "" {
		client, err = firestore.NewClient(ctx, projectID, option.WithCredentialsFile(credentialsFile))
	} else {
		client, err = firestore.NewClient(ctx, projectID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %w", err)
	}

	log.Printf("✅ Firestore connected (project: %s)", projectID)
	return &ClientWrapper{Client: client, ProjectID: projectID}, nil
}

// Ping は Firestore 接続をテストします。
// 通常 Firestore は Ping API を持たないため、シンプルな読み取りを試みます。
func (cw *ClientWrapper) Ping(ctx context.Context) error {
	if cw == nil || cw.Client == nil {
		return fmt.Errorf("firestore client is nil")
	}
	_, err := cw.Client.Collections(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("firestore ping failed: %w", err)
	}
	return nil
}

// Close は Firestore クライアントをクローズします。
func (cw *ClientWrapper) Close() error {
	if cw == nil || cw.Client == nil {
		return nil
	}
	return cw.Client.Close()
}
