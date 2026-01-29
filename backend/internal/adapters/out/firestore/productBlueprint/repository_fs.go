// backend/internal/adapters/out/firestore/productBlueprint/repository_fs.go
// Responsibility: Firestore を用いた ProductBlueprint リポジトリの本体（コレクション参照・構造体定義・生成・ポート適合）を提供する。
package productBlueprint

import (
	"cloud.google.com/go/firestore"

	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintRepositoryFS implements pbdom.Repository using Firestore.
type ProductBlueprintRepositoryFS struct {
	Client *firestore.Client
}

func NewProductBlueprintRepositoryFS(client *firestore.Client) *ProductBlueprintRepositoryFS {
	return &ProductBlueprintRepositoryFS{Client: client}
}

func (r *ProductBlueprintRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("product_blueprints")
}

// history コレクション: product_blueprints_history/{blueprintId}/versions/{version}
func (r *ProductBlueprintRepositoryFS) historyCol(blueprintID string) *firestore.CollectionRef {
	return r.Client.Collection("product_blueprints_history").
		Doc(blueprintID).
		Collection("versions")
}

// Compile-time check: ensure this satisfies domain port
var (
	_ pbdom.Repository = (*ProductBlueprintRepositoryFS)(nil)
)
