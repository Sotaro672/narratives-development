// backend/cmd/seed_category/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/firestore"
)

const collectionName = "productBlueprintCategories"

type CategoryAttributes struct {
	RequiresExpirationDate bool `firestore:"requiresExpirationDate"`
	RequiresLotNumber      bool `firestore:"requiresLotNumber"`
	RequiresIngredients    bool `firestore:"requiresIngredients"`
	RequiresAlcoholNotice  bool `firestore:"requiresAlcoholNotice"`
	RequiresCosmeticNotice bool `firestore:"requiresCosmeticNotice"`
	RequiresStorageMethod  bool `firestore:"requiresStorageMethod"`
}

type CategorySeed struct {
	ID           string             `firestore:"id"`
	Code         string             `firestore:"code"`
	NameJa       string             `firestore:"nameJa"`
	NameEn       string             `firestore:"nameEn"`
	ParentID     *string            `firestore:"parentId,omitempty"`
	Path         []string           `firestore:"path"`
	Kind         string             `firestore:"kind"`
	DisplayOrder int                `firestore:"displayOrder"`
	Attributes   CategoryAttributes `firestore:"attributes"`
	CreatedAt    time.Time          `firestore:"createdAt"`
	UpdatedAt    time.Time          `firestore:"updatedAt"`
}

func main() {
	ctx := context.Background()

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("FIREBASE_PROJECT_ID")
	}
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT_ID")
	}
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT or FIREBASE_PROJECT_ID or GCP_PROJECT_ID is required")
	}

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("failed to create firestore client: %v", err)
	}
	defer client.Close()

	now := time.Now().UTC()
	categories := buildCategories(now)

	batch := client.Batch()

	for _, category := range categories {
		ref := client.Collection(collectionName).Doc(category.ID)

		// NOTE:
		// firestore.MergeAll は map data 専用。
		// CategorySeed は struct なので MergeAll を付けずに Set する。
		batch.Set(ref, category)
	}

	if _, err := batch.Commit(ctx); err != nil {
		log.Fatalf("failed to seed categories: %v", err)
	}

	fmt.Printf("seeded %d product blueprint categories into %s\n", len(categories), collectionName)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func buildCategories(now time.Time) []CategorySeed {
	return []CategorySeed{
		// ------------------------------------------------------------
		// apparel
		// ------------------------------------------------------------
		category(
			"apparel",
			"apparel",
			"衣類",
			"Apparel",
			nil,
			[]string{"apparel"},
			"apparel",
			100,
			CategoryAttributes{},
			now,
		),
		category(
			"apparel.tops",
			"apparel.tops",
			"トップス",
			"Tops",
			strPtr("apparel"),
			[]string{"apparel", "tops"},
			"apparel",
			110,
			CategoryAttributes{},
			now,
		),
		category(
			"apparel.bottoms",
			"apparel.bottoms",
			"ボトムス",
			"Bottoms",
			strPtr("apparel"),
			[]string{"apparel", "bottoms"},
			"apparel",
			120,
			CategoryAttributes{},
			now,
		),
		category(
			"apparel.outerwear",
			"apparel.outerwear",
			"アウター",
			"Outerwear",
			strPtr("apparel"),
			[]string{"apparel", "outerwear"},
			"apparel",
			130,
			CategoryAttributes{},
			now,
		),
		category(
			"apparel.dress",
			"apparel.dress",
			"ワンピース",
			"Dress",
			strPtr("apparel"),
			[]string{"apparel", "dress"},
			"apparel",
			140,
			CategoryAttributes{},
			now,
		),
		category(
			"apparel.shoes",
			"apparel.shoes",
			"靴",
			"Shoes",
			strPtr("apparel"),
			[]string{"apparel", "shoes"},
			"apparel",
			150,
			CategoryAttributes{},
			now,
		),
		category(
			"apparel.bag",
			"apparel.bag",
			"バッグ",
			"Bags",
			strPtr("apparel"),
			[]string{"apparel", "bag"},
			"apparel",
			160,
			CategoryAttributes{},
			now,
		),
		category(
			"apparel.accessory",
			"apparel.accessory",
			"アクセサリー",
			"Accessories",
			strPtr("apparel"),
			[]string{"apparel", "accessory"},
			"apparel",
			170,
			CategoryAttributes{},
			now,
		),

		// ------------------------------------------------------------
		// alcohol
		// ------------------------------------------------------------
		category(
			"alcohol",
			"alcohol",
			"酒類",
			"Alcohol",
			nil,
			[]string{"alcohol"},
			"alcohol",
			200,
			CategoryAttributes{
				RequiresAlcoholNotice: true,
			},
			now,
		),
		category(
			"alcohol.sake",
			"alcohol.sake",
			"日本酒",
			"Sake",
			strPtr("alcohol"),
			[]string{"alcohol", "sake"},
			"alcohol",
			210,
			CategoryAttributes{
				RequiresLotNumber:      true,
				RequiresIngredients:    true,
				RequiresAlcoholNotice:  true,
				RequiresStorageMethod:  true,
				RequiresExpirationDate: false,
			},
			now,
		),
		category(
			"alcohol.wine",
			"alcohol.wine",
			"ワイン",
			"Wine",
			strPtr("alcohol"),
			[]string{"alcohol", "wine"},
			"alcohol",
			220,
			CategoryAttributes{
				RequiresLotNumber:      true,
				RequiresIngredients:    false,
				RequiresAlcoholNotice:  true,
				RequiresStorageMethod:  true,
				RequiresExpirationDate: false,
			},
			now,
		),
		category(
			"alcohol.beer",
			"alcohol.beer",
			"ビール",
			"Beer",
			strPtr("alcohol"),
			[]string{"alcohol", "beer"},
			"alcohol",
			230,
			CategoryAttributes{
				RequiresExpirationDate: true,
				RequiresLotNumber:      true,
				RequiresIngredients:    true,
				RequiresAlcoholNotice:  true,
				RequiresStorageMethod:  true,
			},
			now,
		),
		category(
			"alcohol.whisky",
			"alcohol.whisky",
			"ウイスキー",
			"Whisky",
			strPtr("alcohol"),
			[]string{"alcohol", "whisky"},
			"alcohol",
			240,
			CategoryAttributes{
				RequiresLotNumber:     true,
				RequiresAlcoholNotice: true,
				RequiresStorageMethod: true,
			},
			now,
		),
		category(
			"alcohol.shochu",
			"alcohol.shochu",
			"焼酎",
			"Shochu",
			strPtr("alcohol"),
			[]string{"alcohol", "shochu"},
			"alcohol",
			250,
			CategoryAttributes{
				RequiresLotNumber:     true,
				RequiresIngredients:   true,
				RequiresAlcoholNotice: true,
				RequiresStorageMethod: true,
			},
			now,
		),
		category(
			"alcohol.spirits",
			"alcohol.spirits",
			"スピリッツ",
			"Spirits",
			strPtr("alcohol"),
			[]string{"alcohol", "spirits"},
			"alcohol",
			260,
			CategoryAttributes{
				RequiresLotNumber:     true,
				RequiresAlcoholNotice: true,
				RequiresStorageMethod: true,
			},
			now,
		),

		// ------------------------------------------------------------
		// cosmetics
		// ------------------------------------------------------------
		category(
			"cosmetics",
			"cosmetics",
			"化粧品",
			"Cosmetics",
			nil,
			[]string{"cosmetics"},
			"cosmetics",
			400,
			CategoryAttributes{
				RequiresIngredients:    true,
				RequiresCosmeticNotice: true,
			},
			now,
		),
		category(
			"cosmetics.skincare",
			"cosmetics.skincare",
			"スキンケア",
			"Skincare",
			strPtr("cosmetics"),
			[]string{"cosmetics", "skincare"},
			"cosmetics",
			410,
			CategoryAttributes{
				RequiresExpirationDate: true,
				RequiresLotNumber:      true,
				RequiresIngredients:    true,
				RequiresCosmeticNotice: true,
				RequiresStorageMethod:  true,
			},
			now,
		),
		category(
			"cosmetics.makeup",
			"cosmetics.makeup",
			"メイクアップ",
			"Makeup",
			strPtr("cosmetics"),
			[]string{"cosmetics", "makeup"},
			"cosmetics",
			420,
			CategoryAttributes{
				RequiresExpirationDate: true,
				RequiresLotNumber:      true,
				RequiresIngredients:    true,
				RequiresCosmeticNotice: true,
				RequiresStorageMethod:  true,
			},
			now,
		),
		category(
			"cosmetics.fragrance",
			"cosmetics.fragrance",
			"香水",
			"Fragrance",
			strPtr("cosmetics"),
			[]string{"cosmetics", "fragrance"},
			"cosmetics",
			430,
			CategoryAttributes{
				RequiresLotNumber:      true,
				RequiresIngredients:    true,
				RequiresCosmeticNotice: true,
				RequiresStorageMethod:  true,
			},
			now,
		),
		category(
			"cosmetics.haircare",
			"cosmetics.haircare",
			"ヘアケア",
			"Haircare",
			strPtr("cosmetics"),
			[]string{"cosmetics", "haircare"},
			"cosmetics",
			440,
			CategoryAttributes{
				RequiresExpirationDate: true,
				RequiresLotNumber:      true,
				RequiresIngredients:    true,
				RequiresCosmeticNotice: true,
				RequiresStorageMethod:  true,
			},
			now,
		),
		category(
			"cosmetics.bodycare",
			"cosmetics.bodycare",
			"ボディケア",
			"Bodycare",
			strPtr("cosmetics"),
			[]string{"cosmetics", "bodycare"},
			"cosmetics",
			450,
			CategoryAttributes{
				RequiresExpirationDate: true,
				RequiresLotNumber:      true,
				RequiresIngredients:    true,
				RequiresCosmeticNotice: true,
				RequiresStorageMethod:  true,
			},
			now,
		),

		// ------------------------------------------------------------
		// healthcare
		// ------------------------------------------------------------
		category(
			"healthcare",
			"healthcare",
			"ヘルスケア",
			"Healthcare",
			nil,
			[]string{"healthcare"},
			"healthcare",
			600,
			CategoryAttributes{},
			now,
		),
		category(
			"healthcare.supplement",
			"healthcare.supplement",
			"サプリメント",
			"Supplements",
			strPtr("healthcare"),
			[]string{"healthcare", "supplement"},
			"healthcare",
			610,
			CategoryAttributes{
				RequiresExpirationDate: true,
				RequiresLotNumber:      true,
				RequiresIngredients:    true,
				RequiresStorageMethod:  true,
			},
			now,
		),
		category(
			"healthcare.wellness",
			"healthcare.wellness",
			"ウェルネス用品",
			"Wellness Goods",
			strPtr("healthcare"),
			[]string{"healthcare", "wellness"},
			"healthcare",
			620,
			CategoryAttributes{},
			now,
		),
		category(
			"healthcare.medical_device",
			"healthcare.medical_device",
			"医療・衛生用品",
			"Medical & Hygiene Goods",
			strPtr("healthcare"),
			[]string{"healthcare", "medical_device"},
			"healthcare",
			630,
			CategoryAttributes{
				RequiresLotNumber: true,
			},
			now,
		),

		// ------------------------------------------------------------
		// other
		// ------------------------------------------------------------
		category(
			"other",
			"other",
			"その他",
			"Other",
			nil,
			[]string{"other"},
			"other",
			900,
			CategoryAttributes{},
			now,
		),
		category(
			"other.general",
			"other.general",
			"その他一般",
			"General Other",
			strPtr("other"),
			[]string{"other", "general"},
			"other",
			910,
			CategoryAttributes{},
			now,
		),
	}
}

func category(
	id string,
	code string,
	nameJa string,
	nameEn string,
	parentID *string,
	path []string,
	kind string,
	displayOrder int,
	attributes CategoryAttributes,
	now time.Time,
) CategorySeed {
	return CategorySeed{
		ID:           id,
		Code:         code,
		NameJa:       nameJa,
		NameEn:       nameEn,
		ParentID:     parentID,
		Path:         append([]string(nil), path...),
		Kind:         kind,
		DisplayOrder: displayOrder,
		Attributes:   attributes,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
