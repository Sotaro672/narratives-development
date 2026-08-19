// backend/cmd/seed_category/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
)

const collectionName = "productBlueprintCategories"

type CategorySeed struct {
	ProductBlueprintCategoryPath []string `firestore:"productBlueprintCategoryPath"`
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

	categories := buildCategories()

	batch := client.Batch()

	for _, category := range categories {
		documentID := categoryDocumentID(
			category.ProductBlueprintCategoryPath,
		)

		ref := client.Collection(collectionName).Doc(documentID)

		batch.Set(ref, category)
	}

	if _, err := batch.Commit(ctx); err != nil {
		log.Fatalf("failed to seed categories: %v", err)
	}

	fmt.Printf("seeded %d product blueprint categories into %s\n", len(categories), collectionName)
}

func categoryDocumentID(
	productBlueprintCategoryPath []string,
) string {
	return strings.Join(
		productBlueprintCategoryPath,
		".",
	)
}

func buildCategories() []CategorySeed {
	return []CategorySeed{
		// ------------------------------------------------------------
		// apparel
		// ------------------------------------------------------------
		{
			ProductBlueprintCategoryPath: []string{
				"apparel",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"apparel",
				"tops",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"apparel",
				"bottoms",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"apparel",
				"outerwear",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"apparel",
				"dress",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"apparel",
				"shoes",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"apparel",
				"bag",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"apparel",
				"accessory",
			},
		},

		// ------------------------------------------------------------
		// alcohol
		// ------------------------------------------------------------
		{
			ProductBlueprintCategoryPath: []string{
				"alcohol",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"alcohol",
				"sake",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"alcohol",
				"wine",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"alcohol",
				"beer",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"alcohol",
				"whisky",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"alcohol",
				"shochu",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"alcohol",
				"spirits",
			},
		},

		// ------------------------------------------------------------
		// cosmetics
		// ------------------------------------------------------------
		{
			ProductBlueprintCategoryPath: []string{
				"cosmetics",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"cosmetics",
				"skincare",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"cosmetics",
				"makeup",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"cosmetics",
				"fragrance",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"cosmetics",
				"haircare",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"cosmetics",
				"bodycare",
			},
		},

		// ------------------------------------------------------------
		// healthcare
		// ------------------------------------------------------------
		{
			ProductBlueprintCategoryPath: []string{
				"healthcare",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"healthcare",
				"supplement",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"healthcare",
				"wellness",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"healthcare",
				"medical_device",
			},
		},

		// ------------------------------------------------------------
		// other
		// ------------------------------------------------------------
		{
			ProductBlueprintCategoryPath: []string{
				"other",
			},
		},
		{
			ProductBlueprintCategoryPath: []string{
				"other",
				"general",
			},
		},
	}
}
