// backend/cmd/seed_permissions/main.go
package main

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"

	perm "narratives/internal/domain/permission"
)

func main() {
	ctx := context.Background()

	// 環境に合わせてプロジェクトIDや認証情報は調整してください
	projectID := "narratives-development-26c2d"

	client, err := firestore.NewClient(ctx, projectID /*, option.WithCredentialsFile("...")*/)
	if err != nil {
		log.Fatalf("firestore.NewClient: %v", err)
	}
	defer client.Close()

	col := client.Collection("permissions")
	now := time.Now().UTC()

	batch := client.Batch()

	for _, p := range perm.AllPermissions() {
		// catalog.go の ID をそのまま doc ID に使う
		docRef := col.Doc(p.ID)

		// domain/permission/permission_repository_fs.go で定義した helper を再利用するなら
		data := map[string]any{
			"name":        p.Name,
			"category":    string(p.Category),
			"description": p.Description,
			"createdAt":   now,
			"updatedAt":   now,
		}

		batch.Set(docRef, data, firestore.MergeAll)
	}

	_, err = batch.Commit(ctx)
	if err != nil {
		log.Fatalf("batch.Commit: %v", err)
	}

	log.Println("permissions seeded from catalog.go")
}
